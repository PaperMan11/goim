package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/msg/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/msg/internal/svc"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
)

type MsgServer struct {
	svcCtx *svc.ServiceContext
	pbmsg.UnimplementedMsgServer
}

func NewMsgServer(svcCtx *svc.ServiceContext) *MsgServer {
	return &MsgServer{svcCtx: svcCtx}
}

func (s *MsgServer) GetMaxSeq(ctx context.Context, req *sdkws.GetMaxSeqReq) (*sdkws.GetMaxSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetMaxSeq(ctx, req)
}

func (s *MsgServer) GetMaxSeqs(ctx context.Context, req *pbmsg.GetMaxSeqsReq) (*pbmsg.SeqsInfoResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetMaxSeqs(ctx, req)
}

func (s *MsgServer) GetHasReadSeqs(ctx context.Context, req *pbmsg.GetHasReadSeqsReq) (*pbmsg.SeqsInfoResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetHasReadSeqs(ctx, req)
}

func (s *MsgServer) GetMsgByConversationIDs(ctx context.Context, req *pbmsg.GetMsgByConversationIDsReq) (*pbmsg.GetMsgByConversationIDsResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetMsgByConversationIDs(ctx, req)
}

func (s *MsgServer) GetConversationMaxSeq(ctx context.Context, req *pbmsg.GetConversationMaxSeqReq) (*pbmsg.GetConversationMaxSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetConversationMaxSeq(ctx, req)
}

func (s *MsgServer) PullMessageBySeqs(ctx context.Context, req *sdkws.PullMessageBySeqsReq) (*sdkws.PullMessageBySeqsResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.PullMessageBySeqs(ctx, req)
}

func (s *MsgServer) GetSeqMessage(ctx context.Context, req *pbmsg.GetSeqMessageReq) (*pbmsg.GetSeqMessageResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetSeqMessage(ctx, req)
}

func (s *MsgServer) SearchMessage(ctx context.Context, req *pbmsg.SearchMessageReq) (*pbmsg.SearchMessageResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SearchMessage(ctx, req)
}

func (s *MsgServer) SendMsg(ctx context.Context, req *pbmsg.SendMsgReq) (*pbmsg.SendMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SendMsg(ctx, req)
}

func (s *MsgServer) SendSimpleMsg(ctx context.Context, req *pbmsg.SendSimpleMsgReq) (*pbmsg.SendSimpleMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SendSimpleMsg(ctx, req)
}

func (s *MsgServer) SetUserConversationsMinSeq(ctx context.Context, req *pbmsg.SetUserConversationsMinSeqReq) (*pbmsg.SetUserConversationsMinSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SetUserConversationsMinSeq(ctx, req)
}

func (s *MsgServer) ClearConversationsMsg(ctx context.Context, req *pbmsg.ClearConversationsMsgReq) (*pbmsg.ClearConversationsMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.ClearConversationsMsg(ctx, req)
}

func (s *MsgServer) UserClearAllMsg(ctx context.Context, req *pbmsg.UserClearAllMsgReq) (*pbmsg.UserClearAllMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.UserClearAllMsg(ctx, req)
}

func (s *MsgServer) DeleteMsgs(ctx context.Context, req *pbmsg.DeleteMsgsReq) (*pbmsg.DeleteMsgsResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.DeleteMsgs(ctx, req)
}

func (s *MsgServer) DeleteMsgPhysicalBySeq(ctx context.Context, req *pbmsg.DeleteMsgPhysicalBySeqReq) (*pbmsg.DeleteMsgPhysicalBySeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.DeleteMsgPhysicalBySeq(ctx, req)
}

func (s *MsgServer) DeleteMsgPhysical(ctx context.Context, req *pbmsg.DeleteMsgPhysicalReq) (*pbmsg.DeleteMsgPhysicalResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.DeleteMsgPhysical(ctx, req)
}

func (s *MsgServer) SetSendMsgStatus(ctx context.Context, req *pbmsg.SetSendMsgStatusReq) (*pbmsg.SetSendMsgStatusResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SetSendMsgStatus(ctx, req)
}

func (s *MsgServer) GetSendMsgStatus(ctx context.Context, req *pbmsg.GetSendMsgStatusReq) (*pbmsg.GetSendMsgStatusResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetSendMsgStatus(ctx, req)
}

func (s *MsgServer) RevokeMsg(ctx context.Context, req *pbmsg.RevokeMsgReq) (*pbmsg.RevokeMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.RevokeMsg(ctx, req)
}

func (s *MsgServer) MarkMsgsAsRead(ctx context.Context, req *pbmsg.MarkMsgsAsReadReq) (*pbmsg.MarkMsgsAsReadResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.MarkMsgsAsRead(ctx, req)
}

func (s *MsgServer) MarkConversationAsRead(ctx context.Context, req *pbmsg.MarkConversationAsReadReq) (*pbmsg.MarkConversationAsReadResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.MarkConversationAsRead(ctx, req)
}

func (s *MsgServer) SetConversationHasReadSeq(ctx context.Context, req *pbmsg.SetConversationHasReadSeqReq) (*pbmsg.SetConversationHasReadSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SetConversationHasReadSeq(ctx, req)
}

func (s *MsgServer) GetConversationsHasReadAndMaxSeq(ctx context.Context, req *pbmsg.GetConversationsHasReadAndMaxSeqReq) (*pbmsg.GetConversationsHasReadAndMaxSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetConversationsHasReadAndMaxSeq(ctx, req)
}

func (s *MsgServer) GetActiveUser(ctx context.Context, req *pbmsg.GetActiveUserReq) (*pbmsg.GetActiveUserResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetActiveUser(ctx, req)
}

func (s *MsgServer) GetActiveGroup(ctx context.Context, req *pbmsg.GetActiveGroupReq) (*pbmsg.GetActiveGroupResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetActiveGroup(ctx, req)
}

func (s *MsgServer) GetServerTime(ctx context.Context, req *pbmsg.GetServerTimeReq) (*pbmsg.GetServerTimeResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetServerTime(ctx, req)
}

func (s *MsgServer) ClearMsg(ctx context.Context, req *pbmsg.ClearMsgReq) (*pbmsg.ClearMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.ClearMsg(ctx, req)
}

func (s *MsgServer) DestructMsgs(ctx context.Context, req *pbmsg.DestructMsgsReq) (*pbmsg.DestructMsgsResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.DestructMsgs(ctx, req)
}

func (s *MsgServer) GetActiveConversation(ctx context.Context, req *pbmsg.GetActiveConversationReq) (*pbmsg.GetActiveConversationResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetActiveConversation(ctx, req)
}

func (s *MsgServer) SetUserConversationMaxSeq(ctx context.Context, req *pbmsg.SetUserConversationMaxSeqReq) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SetUserConversationMaxSeq(ctx, req)
}

func (s *MsgServer) SetUserConversationMinSeq(ctx context.Context, req *pbmsg.SetUserConversationMinSeqReq) (*pbmsg.SetUserConversationMinSeqResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.SetUserConversationMinSeq(ctx, req)
}

func (s *MsgServer) GetLastMessageSeqByTime(ctx context.Context, req *pbmsg.GetLastMessageSeqByTimeReq) (*pbmsg.GetLastMessageSeqByTimeResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetLastMessageSeqByTime(ctx, req)
}

func (s *MsgServer) GetLastMessage(ctx context.Context, req *pbmsg.GetLastMessageReq) (*pbmsg.GetLastMessageResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.GetLastMessage(ctx, req)
}

func (s *MsgServer) AddMsg(ctx context.Context, req *pbmsg.AddMsgReq) (*pbmsg.AddMsgResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.AddMsg(ctx, req)
}

func (s *MsgServer) AddMsgs(ctx context.Context, req *pbmsg.AddMsgsReq) (*pbmsg.AddMsgsResp, error) {
	l := logic.NewLogic(ctx, s.svcCtx)
	return l.AddMsgs(ctx, req)
}
