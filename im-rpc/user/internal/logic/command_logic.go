package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
)

func (l *Logic) ProcessUserCommandAdd(ctx context.Context, req *pbuser.ProcessUserCommandAddReq) (*pbuser.ProcessUserCommandAddResp, error) {
	cmd := &model.UserCommand{
		UserID:     req.GetUserID(),
		Type:       int(req.GetType()),
		UUID:       req.GetUuid(),
		Value:      req.GetValue().GetValue(),
		CreateTime: timex.Now(),
		UpdatedAt:  timex.Now(),
	}

	err := l.svcCtx.UserModel.InsertUserCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}

	return &pbuser.ProcessUserCommandAddResp{}, nil
}

func (l *Logic) ProcessUserCommandUpdate(ctx context.Context, req *pbuser.ProcessUserCommandUpdateReq) (*pbuser.ProcessUserCommandUpdateResp, error) {
	err := l.svcCtx.UserModel.UpdateUserCommand(ctx, req.GetUserID(), req.GetUuid(), req.GetValue().GetValue())
	if err != nil {
		return nil, err
	}

	return &pbuser.ProcessUserCommandUpdateResp{}, nil
}

func (l *Logic) ProcessUserCommandDelete(ctx context.Context, req *pbuser.ProcessUserCommandDeleteReq) (*pbuser.ProcessUserCommandDeleteResp, error) {
	err := l.svcCtx.UserModel.DeleteUserCommand(ctx, req.GetUserID(), req.GetUuid())
	if err != nil {
		return nil, err
	}

	return &pbuser.ProcessUserCommandDeleteResp{}, nil
}

func (l *Logic) ProcessUserCommandGet(ctx context.Context, req *pbuser.ProcessUserCommandGetReq) (*pbuser.ProcessUserCommandGetResp, error) {
	return &pbuser.ProcessUserCommandGetResp{}, nil
}

func (l *Logic) ProcessUserCommandGetAll(ctx context.Context, req *pbuser.ProcessUserCommandGetAllReq) (*pbuser.ProcessUserCommandGetAllResp, error) {
	return &pbuser.ProcessUserCommandGetAllResp{}, nil
}
