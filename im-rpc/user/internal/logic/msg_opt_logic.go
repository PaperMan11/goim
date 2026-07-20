package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	userModel "github.com/PaperMan11/goim/pkg/storage/mongo/user"
)

func (l *Logic) SetGlobalRecvMessageOpt(ctx context.Context, req *pbuser.SetGlobalRecvMessageOptReq) (*pbuser.SetGlobalRecvMessageOptResp, error) {
	err := l.svcCtx.UserModel.SetGlobalRecvMsgOpt(ctx, req.GetUserID(), int(req.GetGlobalRecvMsgOpt()))
	if err != nil {
		return nil, err
	}

	return &pbuser.SetGlobalRecvMessageOptResp{}, nil
}

func (l *Logic) GetGlobalRecvMessageOpt(ctx context.Context, req *pbuser.GetGlobalRecvMessageOptReq) (*pbuser.GetGlobalRecvMessageOptResp, error) {
	opt, err := l.svcCtx.UserModel.GetGlobalRecvMsgOpt(ctx, req.GetUserID())
	if err != nil {
		if err == userModel.ErrUserNotFound {
			return &pbuser.GetGlobalRecvMessageOptResp{GlobalRecvMsgOpt: 0}, nil
		}
		return nil, err
	}

	return &pbuser.GetGlobalRecvMessageOptResp{GlobalRecvMsgOpt: int32(opt)}, nil
}
