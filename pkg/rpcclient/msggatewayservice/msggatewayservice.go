package msggatewayservice

import (
	"context"

	pbmsggateway "github.com/PaperMan11/goim/pkg/protocol/msggateway"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type MsgGatewayService interface {
	// 在线推送单条消息给指定用户
	OnlinePushMsg(ctx context.Context, in *pbmsggateway.OnlinePushMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlinePushMsgResp, error)
	// 获取多个用户的在线状态
	GetUsersOnlineStatus(ctx context.Context, in *pbmsggateway.GetUsersOnlineStatusReq, opts ...grpc.CallOption) (*pbmsggateway.GetUsersOnlineStatusResp, error)
	// 批量在线推送单条消息给多个用户
	OnlineBatchPushOneMsg(ctx context.Context, in *pbmsggateway.OnlineBatchPushOneMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlineBatchPushOneMsgResp, error)
	// 超级群批量在线推送单条消息
	SuperGroupOnlineBatchPushOneMsg(ctx context.Context, in *pbmsggateway.OnlineBatchPushOneMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlineBatchPushOneMsgResp, error)
	// 踢用户下线
	KickUserOffline(ctx context.Context, in *pbmsggateway.KickUserOfflineReq, opts ...grpc.CallOption) (*pbmsggateway.KickUserOfflineResp, error)
	// 多端登录检查
	MultiTerminalLoginCheck(ctx context.Context, in *pbmsggateway.MultiTerminalLoginCheckReq, opts ...grpc.CallOption) (*pbmsggateway.MultiTerminalLoginCheckResp, error)
}

type defaultMsgGatewayService struct {
	cli zrpc.Client
}

func NewMsgGatewayService(cli zrpc.Client) MsgGatewayService {
	return &defaultMsgGatewayService{cli: cli}
}

func (s *defaultMsgGatewayService) OnlinePushMsg(ctx context.Context, in *pbmsggateway.OnlinePushMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlinePushMsgResp, error) {
	gatewayClient := pbmsggateway.NewMsgGatewayClient(s.cli.Conn())
	return gatewayClient.OnlinePushMsg(ctx, in, opts...)
}

func (s *defaultMsgGatewayService) GetUsersOnlineStatus(ctx context.Context, in *pbmsggateway.GetUsersOnlineStatusReq, opts ...grpc.CallOption) (*pbmsggateway.GetUsersOnlineStatusResp, error) {
	gatewayClient := pbmsggateway.NewMsgGatewayClient(s.cli.Conn())
	return gatewayClient.GetUsersOnlineStatus(ctx, in, opts...)
}

func (s *defaultMsgGatewayService) OnlineBatchPushOneMsg(ctx context.Context, in *pbmsggateway.OnlineBatchPushOneMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	gatewayClient := pbmsggateway.NewMsgGatewayClient(s.cli.Conn())
	return gatewayClient.OnlineBatchPushOneMsg(ctx, in, opts...)
}

func (s *defaultMsgGatewayService) SuperGroupOnlineBatchPushOneMsg(ctx context.Context, in *pbmsggateway.OnlineBatchPushOneMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	gatewayClient := pbmsggateway.NewMsgGatewayClient(s.cli.Conn())
	return gatewayClient.SuperGroupOnlineBatchPushOneMsg(ctx, in, opts...)
}

func (s *defaultMsgGatewayService) KickUserOffline(ctx context.Context, in *pbmsggateway.KickUserOfflineReq, opts ...grpc.CallOption) (*pbmsggateway.KickUserOfflineResp, error) {
	gatewayClient := pbmsggateway.NewMsgGatewayClient(s.cli.Conn())
	return gatewayClient.KickUserOffline(ctx, in, opts...)
}

func (s *defaultMsgGatewayService) MultiTerminalLoginCheck(ctx context.Context, in *pbmsggateway.MultiTerminalLoginCheckReq, opts ...grpc.CallOption) (*pbmsggateway.MultiTerminalLoginCheckResp, error) {
	gatewayClient := pbmsggateway.NewMsgGatewayClient(s.cli.Conn())
	return gatewayClient.MultiTerminalLoginCheck(ctx, in, opts...)
}

type stubMsgGatewayService struct {
}

func NewStubMsgGatewayService() MsgGatewayService {
	return &stubMsgGatewayService{}
}

func (s *stubMsgGatewayService) OnlinePushMsg(ctx context.Context, in *pbmsggateway.OnlinePushMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlinePushMsgResp, error) {
	return &pbmsggateway.OnlinePushMsgResp{}, nil
}

func (s *stubMsgGatewayService) GetUsersOnlineStatus(ctx context.Context, in *pbmsggateway.GetUsersOnlineStatusReq, opts ...grpc.CallOption) (*pbmsggateway.GetUsersOnlineStatusResp, error) {
	return &pbmsggateway.GetUsersOnlineStatusResp{}, nil
}

func (s *stubMsgGatewayService) OnlineBatchPushOneMsg(ctx context.Context, in *pbmsggateway.OnlineBatchPushOneMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	return &pbmsggateway.OnlineBatchPushOneMsgResp{}, nil
}

func (s *stubMsgGatewayService) SuperGroupOnlineBatchPushOneMsg(ctx context.Context, in *pbmsggateway.OnlineBatchPushOneMsgReq, opts ...grpc.CallOption) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	return &pbmsggateway.OnlineBatchPushOneMsgResp{}, nil
}

func (s *stubMsgGatewayService) KickUserOffline(ctx context.Context, in *pbmsggateway.KickUserOfflineReq, opts ...grpc.CallOption) (*pbmsggateway.KickUserOfflineResp, error) {
	return &pbmsggateway.KickUserOfflineResp{}, nil
}

func (s *stubMsgGatewayService) MultiTerminalLoginCheck(ctx context.Context, in *pbmsggateway.MultiTerminalLoginCheckReq, opts ...grpc.CallOption) (*pbmsggateway.MultiTerminalLoginCheckResp, error) {
	return &pbmsggateway.MultiTerminalLoginCheckResp{}, nil
}
