package internal

import (
	"context"

	pbmsggateway "github.com/PaperMan11/goim/pkg/protocol/msggateway"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HubServer struct {
	pbmsggateway.UnimplementedMsgGatewayServer
	config   *MsgGatewayConfig
	wsServer WsServer
}

func NewHubServer(wsServer WsServer) *HubServer {
	return &HubServer{
		wsServer: wsServer,
	}
}

func (h *HubServer) OnlinePushMsg(context.Context, *pbmsggateway.OnlinePushMsgReq) (*pbmsggateway.OnlinePushMsgResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnlinePushMsg not implemented")
}

func (h *HubServer) GetUsersOnlineStatus(context.Context, *pbmsggateway.GetUsersOnlineStatusReq) (*pbmsggateway.GetUsersOnlineStatusResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsersOnlineStatus not implemented")
}

func (h *HubServer) OnlineBatchPushOneMsg(context.Context, *pbmsggateway.OnlineBatchPushOneMsgReq) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnlineBatchPushOneMsg not implemented")
}

func (h *HubServer) SuperGroupOnlineBatchPushOneMsg(context.Context, *pbmsggateway.OnlineBatchPushOneMsgReq) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SuperGroupOnlineBatchPushOneMsg not implemented")
}

func (h *HubServer) KickUserOffline(context.Context, *pbmsggateway.KickUserOfflineReq) (*pbmsggateway.KickUserOfflineResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method KickUserOffline not implemented")
}

func (h *HubServer) MultiTerminalLoginCheck(context.Context, *pbmsggateway.MultiTerminalLoginCheckReq) (*pbmsggateway.MultiTerminalLoginCheckResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MultiTerminalLoginCheck not implemented")
}
