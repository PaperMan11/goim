package logic

import (
	"context"

	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
)

func (l *Logic) GetUserClientConfig(ctx context.Context, req *pbuser.GetUserClientConfigReq) (*pbuser.GetUserClientConfigResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	configs, err := l.svcCtx.UserModel.GetUserClientConfig(ctx, req.GetUserID())
	if err != nil {
		l.Errorf("get user client config failed, userID: %s, err: %v", req.GetUserID(), err)
		return nil, err
	}

	return &pbuser.GetUserClientConfigResp{
		Configs: configs,
	}, nil
}

func (l *Logic) SetUserClientConfig(ctx context.Context, req *pbuser.SetUserClientConfigReq) (*pbuser.SetUserClientConfigResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	err := l.svcCtx.UserModel.SetUserClientConfig(ctx, req.GetUserID(), req.GetConfigs())
	if err != nil {
		l.Errorf("set user client config failed, userID: %s, err: %v", req.GetUserID(), err)
		return nil, err
	}

	return &pbuser.SetUserClientConfigResp{}, nil
}

func (l *Logic) DelUserClientConfig(ctx context.Context, req *pbuser.DelUserClientConfigReq) (*pbuser.DelUserClientConfigResp, error) {
	if err := l.requireSelfOrAdmin(req.GetUserID()); err != nil {
		return nil, err
	}

	err := l.svcCtx.UserModel.DelUserClientConfig(ctx, req.GetUserID(), req.GetKeys())
	if err != nil {
		l.Errorf("del user client config failed, userID: %s, keys: %v, err: %v", req.GetUserID(), req.GetKeys(), err)
		return nil, err
	}

	return &pbuser.DelUserClientConfigResp{}, nil
}

func (l *Logic) PageUserClientConfig(ctx context.Context, req *pbuser.PageUserClientConfigReq) (*pbuser.PageUserClientConfigResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	pagination := req.GetPagination()
	page := int64(pagination.GetPageNumber())
	size := int64(pagination.GetShowNumber())

	configs, total, err := l.svcCtx.UserModel.PageUserClientConfig(ctx, req.GetUserID(), req.GetKey(), page, size)
	if err != nil {
		l.Errorf("page user client config failed, userID: %s, key: %s, err: %v", req.GetUserID(), req.GetKey(), err)
		return nil, err
	}

	var respConfigs []*pbuser.ClientConfig
	for _, c := range configs {
		respConfigs = append(respConfigs, &pbuser.ClientConfig{
			Key:    c.Key,
			UserID: c.UserID,
			Value:  c.Value,
		})
	}

	return &pbuser.PageUserClientConfigResp{
		Total:   total,
		Configs: respConfigs,
	}, nil
}
