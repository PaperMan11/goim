package internal

import (
	"context"
	"runtime"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbmsggateway "github.com/PaperMan11/goim/pkg/protocol/msggateway"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/utils/workerpool"
	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/PaperMan11/goim/pkg/authverify"
)

type HubServer struct {
	pbmsggateway.UnimplementedMsgGatewayServer
	hubServerConf *HubServerConf
	wsServer      WsServer
	authVerifier  authverify.AuthVerifyService
	pushPool      *workerpool.WorkerPool[*pushTask]
}

type pushTask struct {
	ctx     context.Context
	respMsg *Response
	userID  string
	result  chan<- *pushResult
}

type pushResult struct {
	userID string
	result *pbmsggateway.SingleMsgToUserResults
}

func NewHubServer(wsServer WsServer, authVerifier authverify.AuthVerifyService, hubServerConf *HubServerConf) *HubServer {
	h := &HubServer{
		hubServerConf: hubServerConf,
		wsServer:      wsServer,
		authVerifier:  authVerifier,
	}
	h.pushPool = workerpool.New(h.processPushTask, runtime.NumCPU(), 1024)
	return h
}

func (h *HubServer) Start() error {
	logx.Infof("HubServer Started")
	h.pushPool.Start()
	return nil
}

func (h *HubServer) Stop() error {
	h.pushPool.Stop()
	logx.Infof("HubServer Stopped")
	return nil
}

func (h *HubServer) processPushTask(task *pushTask) {
	userConns := h.wsServer.GetAllUserClients(task.userID)
	if len(userConns) == 0 {
		task.result <- &pushResult{userID: task.userID, result: nil}
		return
	}
	result := &pbmsggateway.SingleMsgToUserResults{
		UserID: task.userID,
		Resp:   make([]*pbmsggateway.SingleMsgToUserPlatform, 0, len(userConns)),
	}
	for _, conn := range userConns {
		clientInfo := conn.Context().HandshakeInfo()
		sendRes := &pbmsggateway.SingleMsgToUserPlatform{
			ResultCode:     0,
			RecvID:         task.userID,
			RecvPlatFormID: clientInfo.GetPlatformID(),
		}
		if err := conn.SendResponse(task.respMsg); err != nil {
			logc.Errorf(task.ctx, "failed to send message to conn %s: %v", conn.ID(), err)
			sendRes.ResultCode = 1
		}
		result.Resp = append(result.Resp, sendRes)
	}
	task.result <- &pushResult{userID: task.userID, result: result}
}

func (h *HubServer) OnlinePushMsg(context.Context, *pbmsggateway.OnlinePushMsgReq) (*pbmsggateway.OnlinePushMsgResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnlinePushMsg not implemented")
}

func (h *HubServer) GetUsersOnlineStatus(ctx context.Context, req *pbmsggateway.GetUsersOnlineStatusReq) (*pbmsggateway.GetUsersOnlineStatusResp, error) {
	opUserID := mcontext.GetOpUserIDFromContext(ctx)
	// 校验用户是否是IM管理员
	isIMAdmin, err := h.authVerifier.IsIMAdmin(ctx, opUserID)
	if err != nil {
		logc.Errorf(ctx, "failed to check if user is IM admin: %v", err)
		return nil, errx.InternalError.WrapWithError(err)
	}
	if !isIMAdmin {
		return nil, errx.NoPermissionError.Wrap("user is not IM admin")
	}

	resp := &pbmsggateway.GetUsersOnlineStatusResp{
		SuccessResult: make([]*pbmsggateway.GetUsersOnlineStatusResp_SuccessResult, 0),
		FailedResult:  make([]*pbmsggateway.GetUsersOnlineStatusResp_FailedDetail, 0),
	}
	for _, userID := range req.UserIDs {
		userConns := h.wsServer.GetAllUserClients(userID)
		successResult := &pbmsggateway.GetUsersOnlineStatusResp_SuccessResult{
			UserID:               userID,
			Status:               constant.Offline,
			DetailPlatformStatus: make([]*pbmsggateway.GetUsersOnlineStatusResp_SuccessDetail, 0),
		}
		for _, conn := range userConns {
			clientInfo := conn.Context().HandshakeInfo()
			successResult.Status = constant.Online
			successResult.DetailPlatformStatus = append(successResult.DetailPlatformStatus, &pbmsggateway.GetUsersOnlineStatusResp_SuccessDetail{
				PlatformID:   clientInfo.GetPlatformID(),
				ConnID:       conn.ID(),
				IsBackground: clientInfo.GetIsBackground(),
				Token:        clientInfo.GetToken(),
			})
		}
		resp.SuccessResult = append(resp.SuccessResult, successResult)
	}
	return resp, nil
}

func (h *HubServer) OnlineBatchPushOneMsg(context.Context, *pbmsggateway.OnlineBatchPushOneMsgReq) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OnlineBatchPushOneMsg not implemented")
}

func (h *HubServer) SuperGroupOnlineBatchPushOneMsg(ctx context.Context, req *pbmsggateway.OnlineBatchPushOneMsgReq) (*pbmsggateway.OnlineBatchPushOneMsgResp, error) {
	userIDs := req.GetPushToUserIDs()
	resp := &pbmsggateway.OnlineBatchPushOneMsgResp{
		SinglePushResult: make([]*pbmsggateway.SingleMsgToUserResults, 0, len(userIDs)),
	}

	if len(userIDs) == 0 {
		return resp, nil
	}

	var pushMsg sdkws.PushMessages
	conversationID := msgprocessor.GetConversationIDByMsg(req.MsgData)
	m := map[string]*sdkws.PullMsgs{conversationID: {Msgs: []*sdkws.MsgData{req.MsgData}}}
	if msgprocessor.IsNotification(conversationID) {
		pushMsg.NotificationMsgs = m
	} else {
		pushMsg.Msgs = m
	}
	marshaledMsg, err := proto.Marshal(&pushMsg)
	if err != nil {
		logc.Errorf(ctx, "failed to marshal push message: %v", err)
		return nil, errx.DataError.WrapWithError(err)
	}
	respMsg := &Response{
		ReqIdentifier: WSPushMsg,
		OperationID:   mcontext.GetOperationIDFromContext(ctx),
		Data:          marshaledMsg,
	}

	resultCh := make(chan *pushResult, len(userIDs))
	failedCount := 0

	for _, userID := range userIDs {
		task := &pushTask{
			ctx:     ctx,
			respMsg: respMsg,
			userID:  userID,
			result:  resultCh,
		}
		if !h.pushPool.TrySubmit(task) {
			logx.Infof("push task channel is full, userID: %s will be processed synchronously", userID)
			go h.processPushTask(task)
			failedCount++
		}
	}

	for i := 0; i < len(userIDs); i++ {
		result := <-resultCh
		if result.result != nil {
			resp.SinglePushResult = append(resp.SinglePushResult, result.result)
		}
	}

	if failedCount > 0 {
		logx.Infof("SuperGroupOnlineBatchPushOneMsg: %d tasks fell back to synchronous processing due to channel full", failedCount)
	}

	return resp, nil
}

func (h *HubServer) KickUserOffline(ctx context.Context, req *pbmsggateway.KickUserOfflineReq) (*pbmsggateway.KickUserOfflineResp, error) {
	for _, userID := range req.KickUserIDList {
		userConns := h.wsServer.GetAllUserClients(userID)
		if req.PlatformID == 0 {
			for _, conn := range userConns {
				h.wsServer.KickClient(conn)
			}
		} else {
			for _, conn := range userConns {
				if conn.Context().HandshakeInfo().GetPlatformID() == req.PlatformID {
					h.wsServer.KickClient(conn)
				}
			}
		}
	}

	return &pbmsggateway.KickUserOfflineResp{}, nil
}

func (h *HubServer) MultiTerminalLoginCheck(ctx context.Context, req *pbmsggateway.MultiTerminalLoginCheckReq) (*pbmsggateway.MultiTerminalLoginCheckResp, error) {
	clients := h.wsServer.GetAllUserClients(req.UserID)
	if len(clients) == 0 {
		return nil, errx.ConnResetError
	}
	for _, conn := range clients {
		if conn.Context().HandshakeInfo().GetPlatformID() == req.PlatformID {
			err := h.wsServer.MultiTerminalCheckStrategy(conn.ID())
			if err != nil {
				logc.Errorf(ctx, "failed to check multi terminal login: userID %s, platformID %d, err: %v", req.UserID, req.PlatformID, err)
				return nil, errx.InternalError
			}
			break
		}
	}

	return &pbmsggateway.MultiTerminalLoginCheckResp{}, nil
}
