package server

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/conversation/internal/logic"
	"github.com/PaperMan11/goim/im-rpc/conversation/internal/svc"
	pbreconversation "github.com/PaperMan11/goim/pkg/protocol/conversation"
)

type ConversationServer struct {
	svcCtx *svc.ServiceContext
	pbreconversation.UnimplementedConversationServer
}

func NewConversationServer(svcCtx *svc.ServiceContext) *ConversationServer {
	return &ConversationServer{svcCtx: svcCtx}
}

func (s *ConversationServer) GetConversation(ctx context.Context, req *pbreconversation.GetConversationReq) (*pbreconversation.GetConversationResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversation(ctx, req)
}

func (s *ConversationServer) GetSortedConversationList(ctx context.Context, req *pbreconversation.GetSortedConversationListReq) (*pbreconversation.GetSortedConversationListResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetSortedConversationList(ctx, req)
}

func (s *ConversationServer) GetAllConversations(ctx context.Context, req *pbreconversation.GetAllConversationsReq) (*pbreconversation.GetAllConversationsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetAllConversations(ctx, req)
}

func (s *ConversationServer) GetConversations(ctx context.Context, req *pbreconversation.GetConversationsReq) (*pbreconversation.GetConversationsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversations(ctx, req)
}

func (s *ConversationServer) SetConversation(ctx context.Context, req *pbreconversation.SetConversationReq) (*pbreconversation.SetConversationResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetConversation(ctx, req)
}

func (s *ConversationServer) GetRecvMsgNotNotifyUserIDs(ctx context.Context, req *pbreconversation.GetRecvMsgNotNotifyUserIDsReq) (*pbreconversation.GetRecvMsgNotNotifyUserIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetRecvMsgNotNotifyUserIDs(ctx, req)
}

func (s *ConversationServer) CreateSingleChatConversations(ctx context.Context, req *pbreconversation.CreateSingleChatConversationsReq) (*pbreconversation.CreateSingleChatConversationsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).CreateSingleChatConversations(ctx, req)
}

func (s *ConversationServer) CreateGroupChatConversations(ctx context.Context, req *pbreconversation.CreateGroupChatConversationsReq) (*pbreconversation.CreateGroupChatConversationsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).CreateGroupChatConversations(ctx, req)
}

func (s *ConversationServer) SetConversationMaxSeq(ctx context.Context, req *pbreconversation.SetConversationMaxSeqReq) (*pbreconversation.SetConversationMaxSeqResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetConversationMaxSeq(ctx, req)
}

func (s *ConversationServer) SetConversationMinSeq(ctx context.Context, req *pbreconversation.SetConversationMinSeqReq) (*pbreconversation.SetConversationMinSeqResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetConversationMinSeq(ctx, req)
}

func (s *ConversationServer) GetConversationIDs(ctx context.Context, req *pbreconversation.GetConversationIDsReq) (*pbreconversation.GetConversationIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversationIDs(ctx, req)
}

func (s *ConversationServer) SetConversations(ctx context.Context, req *pbreconversation.SetConversationsReq) (*pbreconversation.SetConversationsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).SetConversations(ctx, req)
}

func (s *ConversationServer) GetUserConversationIDsHash(ctx context.Context, req *pbreconversation.GetUserConversationIDsHashReq) (*pbreconversation.GetUserConversationIDsHashResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetUserConversationIDsHash(ctx, req)
}

func (s *ConversationServer) GetConversationsByConversationID(ctx context.Context, req *pbreconversation.GetConversationsByConversationIDReq) (*pbreconversation.GetConversationsByConversationIDResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversationsByConversationID(ctx, req)
}

func (s *ConversationServer) GetConversationOfflinePushUserIDs(ctx context.Context, req *pbreconversation.GetConversationOfflinePushUserIDsReq) (*pbreconversation.GetConversationOfflinePushUserIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversationOfflinePushUserIDs(ctx, req)
}

func (s *ConversationServer) GetConversationNotReceiveMessageUserIDs(ctx context.Context, req *pbreconversation.GetConversationNotReceiveMessageUserIDsReq) (*pbreconversation.GetConversationNotReceiveMessageUserIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversationNotReceiveMessageUserIDs(ctx, req)
}

func (s *ConversationServer) UpdateConversation(ctx context.Context, req *pbreconversation.UpdateConversationReq) (*pbreconversation.UpdateConversationResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UpdateConversation(ctx, req)
}

func (s *ConversationServer) GetFullOwnerConversationIDs(ctx context.Context, req *pbreconversation.GetFullOwnerConversationIDsReq) (*pbreconversation.GetFullOwnerConversationIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetFullOwnerConversationIDs(ctx, req)
}

func (s *ConversationServer) GetIncrementalConversation(ctx context.Context, req *pbreconversation.GetIncrementalConversationReq) (*pbreconversation.GetIncrementalConversationResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetIncrementalConversation(ctx, req)
}

func (s *ConversationServer) GetOwnerConversation(ctx context.Context, req *pbreconversation.GetOwnerConversationReq) (*pbreconversation.GetOwnerConversationResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetOwnerConversation(ctx, req)
}

func (s *ConversationServer) GetConversationsNeedClearMsg(ctx context.Context, req *pbreconversation.GetConversationsNeedClearMsgReq) (*pbreconversation.GetConversationsNeedClearMsgResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetConversationsNeedClearMsg(ctx, req)
}

func (s *ConversationServer) GetNotNotifyConversationIDs(ctx context.Context, req *pbreconversation.GetNotNotifyConversationIDsReq) (*pbreconversation.GetNotNotifyConversationIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetNotNotifyConversationIDs(ctx, req)
}

func (s *ConversationServer) GetPinnedConversationIDs(ctx context.Context, req *pbreconversation.GetPinnedConversationIDsReq) (*pbreconversation.GetPinnedConversationIDsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).GetPinnedConversationIDs(ctx, req)
}

func (s *ConversationServer) ClearUserConversationMsg(ctx context.Context, req *pbreconversation.ClearUserConversationMsgReq) (*pbreconversation.ClearUserConversationMsgResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).ClearUserConversationMsg(ctx, req)
}

func (s *ConversationServer) UpdateConversationsByUser(ctx context.Context, req *pbreconversation.UpdateConversationsByUserReq) (*pbreconversation.UpdateConversationsByUserResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).UpdateConversationsByUser(ctx, req)
}

func (s *ConversationServer) DeleteConversations(ctx context.Context, req *pbreconversation.DeleteConversationsReq) (*pbreconversation.DeleteConversationsResp, error) {
	return logic.NewLogic(ctx, s.svcCtx).DeleteConversations(ctx, req)
}
