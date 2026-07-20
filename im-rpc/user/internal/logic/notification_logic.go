package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
)

func (l *Logic) AddNotificationAccount(ctx context.Context, req *pbuser.AddNotificationAccountReq) (*pbuser.AddNotificationAccountResp, error) {
	return &pbuser.AddNotificationAccountResp{}, nil
}

func (l *Logic) UpdateNotificationAccountInfo(ctx context.Context, req *pbuser.UpdateNotificationAccountInfoReq) (*pbuser.UpdateNotificationAccountInfoResp, error) {
	return &pbuser.UpdateNotificationAccountInfoResp{}, nil
}

func (l *Logic) SearchNotificationAccount(ctx context.Context, req *pbuser.SearchNotificationAccountReq) (*pbuser.SearchNotificationAccountResp, error) {
	return &pbuser.SearchNotificationAccountResp{}, nil
}

func (l *Logic) GetNotificationAccount(ctx context.Context, req *pbuser.GetNotificationAccountReq) (*pbuser.GetNotificationAccountResp, error) {
	return &pbuser.GetNotificationAccountResp{}, nil
}
