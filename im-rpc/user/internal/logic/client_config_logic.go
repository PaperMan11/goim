package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
)

func (l *Logic) GetUserClientConfig(ctx context.Context, req *pbuser.GetUserClientConfigReq) (*pbuser.GetUserClientConfigResp, error) {
	return &pbuser.GetUserClientConfigResp{}, nil
}

func (l *Logic) SetUserClientConfig(ctx context.Context, req *pbuser.SetUserClientConfigReq) (*pbuser.SetUserClientConfigResp, error) {
	return &pbuser.SetUserClientConfigResp{}, nil
}

func (l *Logic) DelUserClientConfig(ctx context.Context, req *pbuser.DelUserClientConfigReq) (*pbuser.DelUserClientConfigResp, error) {
	return &pbuser.DelUserClientConfigResp{}, nil
}

func (l *Logic) PageUserClientConfig(ctx context.Context, req *pbuser.PageUserClientConfigReq) (*pbuser.PageUserClientConfigResp, error) {
	return &pbuser.PageUserClientConfigResp{}, nil
}
