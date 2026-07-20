package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
)

func (l *Logic) UserRegisterCount(ctx context.Context, req *pbuser.UserRegisterCountReq) (*pbuser.UserRegisterCountResp, error) {
	total, before, count, err := l.svcCtx.UserModel.RegisterCount(ctx, req.GetStart(), req.GetEnd())
	if err != nil {
		return nil, err
	}

	return &pbuser.UserRegisterCountResp{Total: total, Before: before, Count: count}, nil
}
