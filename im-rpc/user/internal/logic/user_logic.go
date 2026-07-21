package logic

import (
	"context"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	sdkws "github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

func (l *Logic) GetDesignateUsers(ctx context.Context, req *pbuser.GetDesignateUsersReq) (*pbuser.GetDesignateUsersResp, error) {
	users, err := l.svcCtx.UserModel.FindByIDs(ctx, req.GetUserIDs())
	if err != nil {
		return nil, err
	}

	var usersInfo []*sdkws.UserInfo
	for _, user := range users {
		usersInfo = append(usersInfo, modelToUserInfo(user))
	}

	return &pbuser.GetDesignateUsersResp{UsersInfo: usersInfo}, nil
}

func (l *Logic) UpdateUserInfo(ctx context.Context, req *pbuser.UpdateUserInfoReq) (*pbuser.UpdateUserInfoResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserInfo().UserID); err != nil {
		return nil, err
	}

	userInfo := req.GetUserInfo()
	if userInfo == nil {
		return nil, errx.ArgsError.Wrap("user info is nil")
	}

	user := userInfoToModel(userInfo)
	err := l.svcCtx.UserModel.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	return &pbuser.UpdateUserInfoResp{}, nil
}

func (l *Logic) UpdateUserInfoEx(ctx context.Context, req *pbuser.UpdateUserInfoExReq) (*pbuser.UpdateUserInfoExResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserInfo().UserID); err != nil {
		return nil, err
	}

	userInfo := req.GetUserInfo()
	if userInfo == nil {
		return nil, errx.ArgsError.Wrap("user info is nil")
	}

	updates := make(map[string]any)
	if userInfo.Nickname != nil {
		updates["nickname"] = userInfo.GetNickname().GetValue()
	}
	if userInfo.FaceURL != nil {
		updates["face_url"] = userInfo.GetFaceURL().GetValue()
	}
	if userInfo.Ex != nil {
		updates["extra"] = userInfo.GetEx().GetValue()
	}

	err := l.svcCtx.UserModel.UpdateEx(ctx, userInfo.GetUserID(), updates)
	if err != nil {
		l.Errorf("update user info ex failed, userID: %s, err: %v", userInfo.GetUserID(), err)
		return nil, err
	}

	return &pbuser.UpdateUserInfoExResp{}, nil
}

func (l *Logic) AccountCheck(ctx context.Context, req *pbuser.AccountCheckReq) (*pbuser.AccountCheckResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	results, err := l.svcCtx.UserModel.CheckExists(ctx, req.GetCheckUserIDs())
	if err != nil {
		l.Errorf("check user account failed, err: %v", err)
		return nil, err
	}

	var respResults []*pbuser.AccountCheckRespSingleUserStatus
	for userID, exists := range results {
		respResults = append(respResults, &pbuser.AccountCheckRespSingleUserStatus{
			UserID:        userID,
			AccountStatus: boolToStatus(exists),
		})
	}

	return &pbuser.AccountCheckResp{Results: respResults}, nil
}

func (l *Logic) GetPaginationUsers(ctx context.Context, req *pbuser.GetPaginationUsersReq) (*pbuser.GetPaginationUsersResp, error) {
	pagination := req.GetPagination()
	page := int64(1)
	size := int64(20)
	if pagination != nil {
		page = int64(pagination.GetPageNumber())
		size = int64(pagination.GetShowNumber())
	}

	users, total, err := l.svcCtx.UserModel.Page(ctx, page, size, req.GetUserID(), req.GetNickName())
	if err != nil {
		l.Errorf("get pagination users failed, err: %v", err)
		return nil, err
	}

	var usersInfo []*sdkws.UserInfo
	for _, user := range users {
		usersInfo = append(usersInfo, modelToUserInfo(user))
	}

	return &pbuser.GetPaginationUsersResp{Total: int32(total), Users: usersInfo}, nil
}

func (l *Logic) UserRegister(ctx context.Context, req *pbuser.UserRegisterReq) (*pbuser.UserRegisterResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	// 判断传入的userID是否重复
	userIDSet := make(map[string]struct{})
	newUserIDs := make([]string, 0)
	for _, userInfo := range req.GetUsers() {
		userID := userInfo.GetUserID()
		if userID == "" {
			return nil, errx.ArgsError.Wrap("user id is empty")
		}
		newUserIDs = append(newUserIDs, userID)
		if _, ok := userIDSet[userID]; ok {
			return nil, errx.ArgsError.Wrap("user id is duplicate")
		}
		userIDSet[userID] = struct{}{}
	}

	// 判断用户ID是否存在
	exists, err := l.svcCtx.UserModel.CheckExists(ctx, newUserIDs)
	if err != nil {
		l.Errorf("check user account failed, err: %v", err)
		return nil, err
	}
	for _, exists := range exists {
		if exists {
			return nil, errx.UserRegisteredAlreadyError
		}
	}

	var users []*model.User
	timeNow := timex.Now()
	for _, userInfo := range req.GetUsers() {
		user := userInfoToModel(userInfo)
		user.CreatedAt = timeNow
		user.UpdatedAt = timeNow
		users = append(users, user)
	}

	err = l.svcCtx.UserModel.Insert(ctx, users)
	if err != nil {
		l.Errorf("user register failed, err: %v", err)
		return nil, err
	}

	return &pbuser.UserRegisterResp{}, nil
}

func (l *Logic) GetAllUserID(ctx context.Context, req *pbuser.GetAllUserIDReq) (*pbuser.GetAllUserIDResp, error) {
	pagination := req.GetPagination()
	page := int64(1)
	size := int64(100)
	if pagination != nil {
		page = int64(pagination.GetPageNumber())
		size = int64(pagination.GetShowNumber())
	}

	userIDs, total, err := l.svcCtx.UserModel.GetAllUserIDs(ctx, page, size)
	if err != nil {
		l.Errorf("get all user ids failed, err: %v", err)
		return nil, err
	}

	return &pbuser.GetAllUserIDResp{Total: int32(total), UserIDs: userIDs}, nil
}

func (l *Logic) SortQuery(ctx context.Context, req *pbuser.SortQueryReq) (*pbuser.SortQueryResp, error) {
	users, err := l.svcCtx.UserModel.SortQuery(ctx, req.GetUserIDName(), req.GetAsc())
	if err != nil {
		return nil, err
	}

	var usersInfo []*sdkws.UserInfo
	for _, user := range users {
		usersInfo = append(usersInfo, modelToUserInfo(user))
	}

	return &pbuser.SortQueryResp{Users: usersInfo}, nil
}

func (l *Logic) IsIMAdmin(ctx context.Context, req *pbuser.IsIMAdminReq) (*pbuser.IsIMAdminResp, error) {
	isAdmin, err := l.svcCtx.UserModel.IsIMAdmin(ctx, req.GetUserID())
	if err != nil {
		return nil, err
	}

	return &pbuser.IsIMAdminResp{IsIMAdmin: isAdmin}, nil
}
