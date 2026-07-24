package logic

import (
	"context"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	userModel "github.com/PaperMan11/goim/pkg/storage/mongo/user"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

func (l *Logic) AddNotificationAccount(ctx context.Context, req *pbuser.AddNotificationAccountReq) (*pbuser.AddNotificationAccountResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}
	if req.GetAppMangerLevel() < constant.AppNotificationAdmin {
		return nil, errx.ArgsError.Wrap("app manager level must be >= notification admin")
	}

	var userID string
	if req.GetUserID() == "" {
		randomPart, err := randx.SecureString(10, randx.CharsAlphaNum)
		if err != nil {
			l.Errorf("generate random user id failed, err: %v", err)
			return nil, errx.InternalError.Wrap("generate user id failed")
		}
		userID = "notification_" + randomPart
	} else {
		userID = req.GetUserID()
		_, err := l.svcCtx.UserModel.FindByID(ctx, userID)
		if err != nil {
			if err == userModel.ErrUserNotFound {
				return nil, errx.ArgsError.Wrap("user not found")
			}
			l.Errorf("find user failed, userID: %s, err: %v", userID, err)
			return nil, err
		}
		err = l.svcCtx.UserModel.UpdateEx(ctx, userID, map[string]any{
			"app_manager_level": req.GetAppMangerLevel(),
			"updated_at":        timex.Now(),
		})
		if err != nil {
			l.Errorf("update user permission failed, userID: %s, err: %v", userID, err)
			return nil, err
		}
		return &pbuser.AddNotificationAccountResp{
			UserID:         userID,
			FaceURL:        req.GetFaceURL(),
			NickName:       req.GetNickName(),
			AppMangerLevel: req.GetAppMangerLevel(),
		}, nil
	}

	now := timex.Now()
	err := l.svcCtx.UserModel.Insert(ctx, []*model.User{{
		UserID:          userID,
		Nickname:        req.GetNickName(),
		FaceURL:         req.GetFaceURL(),
		AppManagerLevel: int(req.GetAppMangerLevel()),
		CreatedAt:       now,
		UpdatedAt:       now,
	}})
	if err != nil {
		l.Errorf("insert notification account failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	return &pbuser.AddNotificationAccountResp{
		UserID:         userID,
		FaceURL:        req.GetFaceURL(),
		NickName:       req.GetNickName(),
		AppMangerLevel: req.GetAppMangerLevel(),
	}, nil
}

func (l *Logic) UpdateNotificationAccountInfo(ctx context.Context, req *pbuser.UpdateNotificationAccountInfoReq) (*pbuser.UpdateNotificationAccountInfoResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("user id is empty")
	}

	updates := make(map[string]any)
	if req.GetNickName() != "" {
		updates["nickname"] = req.GetNickName()
	}
	if req.GetFaceURL() != "" {
		updates["face_url"] = req.GetFaceURL()
	}
	if len(updates) == 0 {
		return &pbuser.UpdateNotificationAccountInfoResp{}, nil
	}
	updates["updated_at"] = timex.Now()

	err := l.svcCtx.UserModel.UpdateEx(ctx, userID, updates)
	if err != nil {
		l.Errorf("update notification account info failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	return &pbuser.UpdateNotificationAccountInfoResp{}, nil
}

func (l *Logic) SearchNotificationAccount(ctx context.Context, req *pbuser.SearchNotificationAccountReq) (*pbuser.SearchNotificationAccountResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	keyword := req.GetKeyword()
	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())

	var users []*model.User
	var total int64
	var err error

	if keyword != "" {
		users, total, err = l.svcCtx.UserModel.Page(ctx, page, size, "", keyword, constant.AppNotificationAdmin)
	} else if req.AppManagerLevel != nil {
		users, total, err = l.svcCtx.UserModel.PageByAppManagerLevel(ctx, page, size, int(*req.AppManagerLevel))
	} else {
		users, total, err = l.svcCtx.UserModel.Page(ctx, page, size, "", "", constant.AppNotificationAdmin)
	}
	if err != nil {
		l.Errorf("search notification account failed, keyword: %s, err: %v", keyword, err)
		return nil, err
	}

	var accounts []*pbuser.NotificationAccountInfo
	for _, user := range users {
		accounts = append(accounts, &pbuser.NotificationAccountInfo{
			UserID:         user.UserID,
			NickName:       user.Nickname,
			FaceURL:        user.FaceURL,
			AppMangerLevel: int32(user.AppManagerLevel),
		})
	}

	return &pbuser.SearchNotificationAccountResp{
		Total:                total,
		NotificationAccounts: accounts,
	}, nil
}

func (l *Logic) GetNotificationAccount(ctx context.Context, req *pbuser.GetNotificationAccountReq) (*pbuser.GetNotificationAccountResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("user id is empty")
	}

	user, err := l.svcCtx.UserModel.FindByID(ctx, userID)
	if err != nil {
		if err == userModel.ErrUserNotFound {
			return nil, errx.ArgsError.Wrap("notification account not found")
		}
		l.Errorf("get notification account failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	if user.AppManagerLevel < constant.AppNotificationAdmin {
		return nil, errx.ArgsError.Wrap("user is not a notification account")
	}

	return &pbuser.GetNotificationAccountResp{
		Account: &pbuser.NotificationAccountInfo{
			UserID:         user.UserID,
			NickName:       user.Nickname,
			FaceURL:        user.FaceURL,
			AppMangerLevel: int32(user.AppManagerLevel),
		},
	}, nil
}
