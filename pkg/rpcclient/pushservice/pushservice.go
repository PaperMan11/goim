package pushservice

import (
	"context"

	pbpush "github.com/PaperMan11/goim/pkg/protocol/push"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type PushService interface {
	// 推送消息
	PushMsg(ctx context.Context, in *pbpush.PushMsgReq, opts ...grpc.CallOption) (*pbpush.PushMsgResp, error)
	// 删除用户推送Token
	DelUserPushToken(ctx context.Context, in *pbpush.DelUserPushTokenReq, opts ...grpc.CallOption) (*pbpush.DelUserPushTokenResp, error)
}

type defaultPushService struct {
	cli zrpc.Client
}

func NewPushService(cli zrpc.Client) PushService {
	return &defaultPushService{cli: cli}
}

func (s *defaultPushService) PushMsg(ctx context.Context, in *pbpush.PushMsgReq, opts ...grpc.CallOption) (*pbpush.PushMsgResp, error) {
	pushClient := pbpush.NewPushMsgServiceClient(s.cli.Conn())
	return pushClient.PushMsg(ctx, in, opts...)
}

func (s *defaultPushService) DelUserPushToken(ctx context.Context, in *pbpush.DelUserPushTokenReq, opts ...grpc.CallOption) (*pbpush.DelUserPushTokenResp, error) {
	pushClient := pbpush.NewPushMsgServiceClient(s.cli.Conn())
	return pushClient.DelUserPushToken(ctx, in, opts...)
}

type stubPushService struct {
}

func NewStubPushService() PushService {
	return &stubPushService{}
}

func (s *stubPushService) PushMsg(ctx context.Context, in *pbpush.PushMsgReq, opts ...grpc.CallOption) (*pbpush.PushMsgResp, error) {
	return &pbpush.PushMsgResp{}, nil
}

func (s *stubPushService) DelUserPushToken(ctx context.Context, in *pbpush.DelUserPushTokenReq, opts ...grpc.CallOption) (*pbpush.DelUserPushTokenResp, error) {
	return &pbpush.DelUserPushTokenResp{}, nil
}
