package conversationservice

import (
	"context"

	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ConversationService interface {
	// 获取单个会话
	GetConversation(ctx context.Context, in *pbconv.GetConversationReq, opts ...grpc.CallOption) (*pbconv.GetConversationResp, error)
	// 获取排序会话列表
	GetSortedConversationList(ctx context.Context, in *pbconv.GetSortedConversationListReq, opts ...grpc.CallOption) (*pbconv.GetSortedConversationListResp, error)
	// 获取所有会话
	GetAllConversations(ctx context.Context, in *pbconv.GetAllConversationsReq, opts ...grpc.CallOption) (*pbconv.GetAllConversationsResp, error)
	// 获取多个会话
	GetConversations(ctx context.Context, in *pbconv.GetConversationsReq, opts ...grpc.CallOption) (*pbconv.GetConversationsResp, error)
	// 设置会话
	SetConversation(ctx context.Context, in *pbconv.SetConversationReq, opts ...grpc.CallOption) (*pbconv.SetConversationResp, error)
	// 获取群消息不通知用户列表
	GetRecvMsgNotNotifyUserIDs(ctx context.Context, in *pbconv.GetRecvMsgNotNotifyUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetRecvMsgNotNotifyUserIDsResp, error)
	// 创建单聊会话
	CreateSingleChatConversations(ctx context.Context, in *pbconv.CreateSingleChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateSingleChatConversationsResp, error)
	// 创建群聊会话
	CreateGroupChatConversations(ctx context.Context, in *pbconv.CreateGroupChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateGroupChatConversationsResp, error)
	// 设置会话最大序列号
	SetConversationMaxSeq(ctx context.Context, in *pbconv.SetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMaxSeqResp, error)
	// 设置会话最小序列号
	SetConversationMinSeq(ctx context.Context, in *pbconv.SetConversationMinSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMinSeqResp, error)
	// 获取会话ID列表
	GetConversationIDs(ctx context.Context, in *pbconv.GetConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationIDsResp, error)
	// 批量设置会话
	SetConversations(ctx context.Context, in *pbconv.SetConversationsReq, opts ...grpc.CallOption) (*pbconv.SetConversationsResp, error)
	// 获取用户会话ID哈希
	GetUserConversationIDsHash(ctx context.Context, in *pbconv.GetUserConversationIDsHashReq, opts ...grpc.CallOption) (*pbconv.GetUserConversationIDsHashResp, error)
	// 按会话ID获取会话列表
	GetConversationsByConversationID(ctx context.Context, in *pbconv.GetConversationsByConversationIDReq, opts ...grpc.CallOption) (*pbconv.GetConversationsByConversationIDResp, error)
	// 获取会话离线推送用户列表
	GetConversationOfflinePushUserIDs(ctx context.Context, in *pbconv.GetConversationOfflinePushUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationOfflinePushUserIDsResp, error)
	// 获取会话不接收消息用户列表
	GetConversationNotReceiveMessageUserIDs(ctx context.Context, in *pbconv.GetConversationNotReceiveMessageUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationNotReceiveMessageUserIDsResp, error)
	// 更新会话
	UpdateConversation(ctx context.Context, in *pbconv.UpdateConversationReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationResp, error)
	// 获取完整所有者会话ID列表
	GetFullOwnerConversationIDs(ctx context.Context, in *pbconv.GetFullOwnerConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetFullOwnerConversationIDsResp, error)
	// 获取增量会话
	GetIncrementalConversation(ctx context.Context, in *pbconv.GetIncrementalConversationReq, opts ...grpc.CallOption) (*pbconv.GetIncrementalConversationResp, error)
	// 获取所有者会话列表
	GetOwnerConversation(ctx context.Context, in *pbconv.GetOwnerConversationReq, opts ...grpc.CallOption) (*pbconv.GetOwnerConversationResp, error)
	// 获取需要清理消息的会话
	GetConversationsNeedClearMsg(ctx context.Context, in *pbconv.GetConversationsNeedClearMsgReq, opts ...grpc.CallOption) (*pbconv.GetConversationsNeedClearMsgResp, error)
	// 获取不通知会话ID列表
	GetNotNotifyConversationIDs(ctx context.Context, in *pbconv.GetNotNotifyConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetNotNotifyConversationIDsResp, error)
	// 获取置顶会话ID列表
	GetPinnedConversationIDs(ctx context.Context, in *pbconv.GetPinnedConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetPinnedConversationIDsResp, error)
	// 清理用户会话消息
	ClearUserConversationMsg(ctx context.Context, in *pbconv.ClearUserConversationMsgReq, opts ...grpc.CallOption) (*pbconv.ClearUserConversationMsgResp, error)
	// 按用户更新会话
	UpdateConversationsByUser(ctx context.Context, in *pbconv.UpdateConversationsByUserReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationsByUserResp, error)
	// 删除会话
	DeleteConversations(ctx context.Context, in *pbconv.DeleteConversationsReq, opts ...grpc.CallOption) (*pbconv.DeleteConversationsResp, error)
}

type defaultConversationService struct {
	cli zrpc.Client
}

func NewConversationService(cli zrpc.Client) ConversationService {
	return &defaultConversationService{cli: cli}
}

func (s *defaultConversationService) GetConversation(ctx context.Context, in *pbconv.GetConversationReq, opts ...grpc.CallOption) (*pbconv.GetConversationResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversation(ctx, in, opts...)
}

func (s *defaultConversationService) GetSortedConversationList(ctx context.Context, in *pbconv.GetSortedConversationListReq, opts ...grpc.CallOption) (*pbconv.GetSortedConversationListResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetSortedConversationList(ctx, in, opts...)
}

func (s *defaultConversationService) GetAllConversations(ctx context.Context, in *pbconv.GetAllConversationsReq, opts ...grpc.CallOption) (*pbconv.GetAllConversationsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetAllConversations(ctx, in, opts...)
}

func (s *defaultConversationService) GetConversations(ctx context.Context, in *pbconv.GetConversationsReq, opts ...grpc.CallOption) (*pbconv.GetConversationsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversations(ctx, in, opts...)
}

func (s *defaultConversationService) SetConversation(ctx context.Context, in *pbconv.SetConversationReq, opts ...grpc.CallOption) (*pbconv.SetConversationResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.SetConversation(ctx, in, opts...)
}

func (s *defaultConversationService) GetRecvMsgNotNotifyUserIDs(ctx context.Context, in *pbconv.GetRecvMsgNotNotifyUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetRecvMsgNotNotifyUserIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetRecvMsgNotNotifyUserIDs(ctx, in, opts...)
}

func (s *defaultConversationService) CreateSingleChatConversations(ctx context.Context, in *pbconv.CreateSingleChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateSingleChatConversationsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.CreateSingleChatConversations(ctx, in, opts...)
}

func (s *defaultConversationService) CreateGroupChatConversations(ctx context.Context, in *pbconv.CreateGroupChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateGroupChatConversationsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.CreateGroupChatConversations(ctx, in, opts...)
}

func (s *defaultConversationService) SetConversationMaxSeq(ctx context.Context, in *pbconv.SetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMaxSeqResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.SetConversationMaxSeq(ctx, in, opts...)
}

func (s *defaultConversationService) SetConversationMinSeq(ctx context.Context, in *pbconv.SetConversationMinSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMinSeqResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.SetConversationMinSeq(ctx, in, opts...)
}

func (s *defaultConversationService) GetConversationIDs(ctx context.Context, in *pbconv.GetConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversationIDs(ctx, in, opts...)
}

func (s *defaultConversationService) SetConversations(ctx context.Context, in *pbconv.SetConversationsReq, opts ...grpc.CallOption) (*pbconv.SetConversationsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.SetConversations(ctx, in, opts...)
}

func (s *defaultConversationService) GetUserConversationIDsHash(ctx context.Context, in *pbconv.GetUserConversationIDsHashReq, opts ...grpc.CallOption) (*pbconv.GetUserConversationIDsHashResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetUserConversationIDsHash(ctx, in, opts...)
}

func (s *defaultConversationService) GetConversationsByConversationID(ctx context.Context, in *pbconv.GetConversationsByConversationIDReq, opts ...grpc.CallOption) (*pbconv.GetConversationsByConversationIDResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversationsByConversationID(ctx, in, opts...)
}

func (s *defaultConversationService) GetConversationOfflinePushUserIDs(ctx context.Context, in *pbconv.GetConversationOfflinePushUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationOfflinePushUserIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversationOfflinePushUserIDs(ctx, in, opts...)
}

func (s *defaultConversationService) GetConversationNotReceiveMessageUserIDs(ctx context.Context, in *pbconv.GetConversationNotReceiveMessageUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationNotReceiveMessageUserIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversationNotReceiveMessageUserIDs(ctx, in, opts...)
}

func (s *defaultConversationService) UpdateConversation(ctx context.Context, in *pbconv.UpdateConversationReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.UpdateConversation(ctx, in, opts...)
}

func (s *defaultConversationService) GetFullOwnerConversationIDs(ctx context.Context, in *pbconv.GetFullOwnerConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetFullOwnerConversationIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetFullOwnerConversationIDs(ctx, in, opts...)
}

func (s *defaultConversationService) GetIncrementalConversation(ctx context.Context, in *pbconv.GetIncrementalConversationReq, opts ...grpc.CallOption) (*pbconv.GetIncrementalConversationResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetIncrementalConversation(ctx, in, opts...)
}

func (s *defaultConversationService) GetOwnerConversation(ctx context.Context, in *pbconv.GetOwnerConversationReq, opts ...grpc.CallOption) (*pbconv.GetOwnerConversationResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetOwnerConversation(ctx, in, opts...)
}

func (s *defaultConversationService) GetConversationsNeedClearMsg(ctx context.Context, in *pbconv.GetConversationsNeedClearMsgReq, opts ...grpc.CallOption) (*pbconv.GetConversationsNeedClearMsgResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetConversationsNeedClearMsg(ctx, in, opts...)
}

func (s *defaultConversationService) GetNotNotifyConversationIDs(ctx context.Context, in *pbconv.GetNotNotifyConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetNotNotifyConversationIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetNotNotifyConversationIDs(ctx, in, opts...)
}

func (s *defaultConversationService) GetPinnedConversationIDs(ctx context.Context, in *pbconv.GetPinnedConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetPinnedConversationIDsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.GetPinnedConversationIDs(ctx, in, opts...)
}

func (s *defaultConversationService) ClearUserConversationMsg(ctx context.Context, in *pbconv.ClearUserConversationMsgReq, opts ...grpc.CallOption) (*pbconv.ClearUserConversationMsgResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.ClearUserConversationMsg(ctx, in, opts...)
}

func (s *defaultConversationService) UpdateConversationsByUser(ctx context.Context, in *pbconv.UpdateConversationsByUserReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationsByUserResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.UpdateConversationsByUser(ctx, in, opts...)
}

func (s *defaultConversationService) DeleteConversations(ctx context.Context, in *pbconv.DeleteConversationsReq, opts ...grpc.CallOption) (*pbconv.DeleteConversationsResp, error) {
	convClient := pbconv.NewConversationClient(s.cli.Conn())
	return convClient.DeleteConversations(ctx, in, opts...)
}

type stubConversationService struct {
}

func NewStubConversationService() ConversationService {
	return &stubConversationService{}
}

func (s *stubConversationService) GetConversation(ctx context.Context, in *pbconv.GetConversationReq, opts ...grpc.CallOption) (*pbconv.GetConversationResp, error) {
	return &pbconv.GetConversationResp{}, nil
}

func (s *stubConversationService) GetSortedConversationList(ctx context.Context, in *pbconv.GetSortedConversationListReq, opts ...grpc.CallOption) (*pbconv.GetSortedConversationListResp, error) {
	return &pbconv.GetSortedConversationListResp{}, nil
}

func (s *stubConversationService) GetAllConversations(ctx context.Context, in *pbconv.GetAllConversationsReq, opts ...grpc.CallOption) (*pbconv.GetAllConversationsResp, error) {
	return &pbconv.GetAllConversationsResp{}, nil
}

func (s *stubConversationService) GetConversations(ctx context.Context, in *pbconv.GetConversationsReq, opts ...grpc.CallOption) (*pbconv.GetConversationsResp, error) {
	return &pbconv.GetConversationsResp{}, nil
}

func (s *stubConversationService) SetConversation(ctx context.Context, in *pbconv.SetConversationReq, opts ...grpc.CallOption) (*pbconv.SetConversationResp, error) {
	return &pbconv.SetConversationResp{}, nil
}

func (s *stubConversationService) GetRecvMsgNotNotifyUserIDs(ctx context.Context, in *pbconv.GetRecvMsgNotNotifyUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetRecvMsgNotNotifyUserIDsResp, error) {
	return &pbconv.GetRecvMsgNotNotifyUserIDsResp{}, nil
}

func (s *stubConversationService) CreateSingleChatConversations(ctx context.Context, in *pbconv.CreateSingleChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateSingleChatConversationsResp, error) {
	return &pbconv.CreateSingleChatConversationsResp{}, nil
}

func (s *stubConversationService) CreateGroupChatConversations(ctx context.Context, in *pbconv.CreateGroupChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateGroupChatConversationsResp, error) {
	return &pbconv.CreateGroupChatConversationsResp{}, nil
}

func (s *stubConversationService) SetConversationMaxSeq(ctx context.Context, in *pbconv.SetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMaxSeqResp, error) {
	return &pbconv.SetConversationMaxSeqResp{}, nil
}

func (s *stubConversationService) SetConversationMinSeq(ctx context.Context, in *pbconv.SetConversationMinSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMinSeqResp, error) {
	return &pbconv.SetConversationMinSeqResp{}, nil
}

func (s *stubConversationService) GetConversationIDs(ctx context.Context, in *pbconv.GetConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationIDsResp, error) {
	return &pbconv.GetConversationIDsResp{}, nil
}

func (s *stubConversationService) SetConversations(ctx context.Context, in *pbconv.SetConversationsReq, opts ...grpc.CallOption) (*pbconv.SetConversationsResp, error) {
	return &pbconv.SetConversationsResp{}, nil
}

func (s *stubConversationService) GetUserConversationIDsHash(ctx context.Context, in *pbconv.GetUserConversationIDsHashReq, opts ...grpc.CallOption) (*pbconv.GetUserConversationIDsHashResp, error) {
	return &pbconv.GetUserConversationIDsHashResp{}, nil
}

func (s *stubConversationService) GetConversationsByConversationID(ctx context.Context, in *pbconv.GetConversationsByConversationIDReq, opts ...grpc.CallOption) (*pbconv.GetConversationsByConversationIDResp, error) {
	return &pbconv.GetConversationsByConversationIDResp{}, nil
}

func (s *stubConversationService) GetConversationOfflinePushUserIDs(ctx context.Context, in *pbconv.GetConversationOfflinePushUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationOfflinePushUserIDsResp, error) {
	return &pbconv.GetConversationOfflinePushUserIDsResp{}, nil
}

func (s *stubConversationService) GetConversationNotReceiveMessageUserIDs(ctx context.Context, in *pbconv.GetConversationNotReceiveMessageUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationNotReceiveMessageUserIDsResp, error) {
	return &pbconv.GetConversationNotReceiveMessageUserIDsResp{}, nil
}

func (s *stubConversationService) UpdateConversation(ctx context.Context, in *pbconv.UpdateConversationReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationResp, error) {
	return &pbconv.UpdateConversationResp{}, nil
}

func (s *stubConversationService) GetFullOwnerConversationIDs(ctx context.Context, in *pbconv.GetFullOwnerConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetFullOwnerConversationIDsResp, error) {
	return &pbconv.GetFullOwnerConversationIDsResp{}, nil
}

func (s *stubConversationService) GetIncrementalConversation(ctx context.Context, in *pbconv.GetIncrementalConversationReq, opts ...grpc.CallOption) (*pbconv.GetIncrementalConversationResp, error) {
	return &pbconv.GetIncrementalConversationResp{}, nil
}

func (s *stubConversationService) GetOwnerConversation(ctx context.Context, in *pbconv.GetOwnerConversationReq, opts ...grpc.CallOption) (*pbconv.GetOwnerConversationResp, error) {
	return &pbconv.GetOwnerConversationResp{}, nil
}

func (s *stubConversationService) GetConversationsNeedClearMsg(ctx context.Context, in *pbconv.GetConversationsNeedClearMsgReq, opts ...grpc.CallOption) (*pbconv.GetConversationsNeedClearMsgResp, error) {
	return &pbconv.GetConversationsNeedClearMsgResp{}, nil
}

func (s *stubConversationService) GetNotNotifyConversationIDs(ctx context.Context, in *pbconv.GetNotNotifyConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetNotNotifyConversationIDsResp, error) {
	return &pbconv.GetNotNotifyConversationIDsResp{}, nil
}

func (s *stubConversationService) GetPinnedConversationIDs(ctx context.Context, in *pbconv.GetPinnedConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetPinnedConversationIDsResp, error) {
	return &pbconv.GetPinnedConversationIDsResp{}, nil
}

func (s *stubConversationService) ClearUserConversationMsg(ctx context.Context, in *pbconv.ClearUserConversationMsgReq, opts ...grpc.CallOption) (*pbconv.ClearUserConversationMsgResp, error) {
	return &pbconv.ClearUserConversationMsgResp{}, nil
}

func (s *stubConversationService) UpdateConversationsByUser(ctx context.Context, in *pbconv.UpdateConversationsByUserReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationsByUserResp, error) {
	return &pbconv.UpdateConversationsByUserResp{}, nil
}

func (s *stubConversationService) DeleteConversations(ctx context.Context, in *pbconv.DeleteConversationsReq, opts ...grpc.CallOption) (*pbconv.DeleteConversationsResp, error) {
	return &pbconv.DeleteConversationsResp{}, nil
}
