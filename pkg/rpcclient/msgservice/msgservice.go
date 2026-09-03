package msgservice

import (
	"context"

	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type MsgService interface {
	// 获取最大序列号(包含用户和指定群组)
	GetMaxSeq(ctx context.Context, in *sdkws.GetMaxSeqReq, opts ...grpc.CallOption) (*sdkws.GetMaxSeqResp, error)
	// 获取会话列表的最大序列号
	GetMaxSeqs(ctx context.Context, in *pbmsg.GetMaxSeqsReq, opts ...grpc.CallOption) (*pbmsg.SeqsInfoResp, error)
	// 获取会话列表的已读序列号
	GetHasReadSeqs(ctx context.Context, in *pbmsg.GetHasReadSeqsReq, opts ...grpc.CallOption) (*pbmsg.SeqsInfoResp, error)
	// 获取最新消息
	GetMsgByConversationIDs(ctx context.Context, in *pbmsg.GetMsgByConversationIDsReq, opts ...grpc.CallOption) (*pbmsg.GetMsgByConversationIDsResp, error)
	// 获取会话最大序列号
	GetConversationMaxSeq(ctx context.Context, in *pbmsg.GetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationMaxSeqResp, error)
	// 拉取历史消息(包含用户和指定群组)
	PullMessageBySeqs(ctx context.Context, in *sdkws.PullMessageBySeqsReq, opts ...grpc.CallOption) (*sdkws.PullMessageBySeqsResp, error)
	// 获取序列号消息
	GetSeqMessage(ctx context.Context, in *pbmsg.GetSeqMessageReq, opts ...grpc.CallOption) (*pbmsg.GetSeqMessageResp, error)
	// 搜索消息
	SearchMessage(ctx context.Context, in *pbmsg.SearchMessageReq, opts ...grpc.CallOption) (*pbmsg.SearchMessageResp, error)
	// 发送消息
	SendMsg(ctx context.Context, in *pbmsg.SendMsgReq, opts ...grpc.CallOption) (*pbmsg.SendMsgResp, error)
	// 发送简单消息
	SendSimpleMsg(ctx context.Context, in *pbmsg.SendSimpleMsgReq, opts ...grpc.CallOption) (*pbmsg.SendSimpleMsgResp, error)
	// 设置用户会话最小序列号
	SetUserConversationsMinSeq(ctx context.Context, in *pbmsg.SetUserConversationsMinSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationsMinSeqResp, error)
	// 清空指定会话的所有消息并重置最小序列号
	ClearConversationsMsg(ctx context.Context, in *pbmsg.ClearConversationsMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearConversationsMsgResp, error)
	// 用户清空所有消息并重置最小序列号
	UserClearAllMsg(ctx context.Context, in *pbmsg.UserClearAllMsgReq, opts ...grpc.CallOption) (*pbmsg.UserClearAllMsgResp, error)
	// 按序列号标记消息删除
	DeleteMsgs(ctx context.Context, in *pbmsg.DeleteMsgsReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgsResp, error)
	// 按序列号物理删除消息
	DeleteMsgPhysicalBySeq(ctx context.Context, in *pbmsg.DeleteMsgPhysicalBySeqReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgPhysicalBySeqResp, error)
	// 按时间戳物理删除消息
	DeleteMsgPhysical(ctx context.Context, in *pbmsg.DeleteMsgPhysicalReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgPhysicalResp, error)
	// 设置API发送消息的送达状态
	SetSendMsgStatus(ctx context.Context, in *pbmsg.SetSendMsgStatusReq, opts ...grpc.CallOption) (*pbmsg.SetSendMsgStatusResp, error)
	// 获取消息送达状态
	GetSendMsgStatus(ctx context.Context, in *pbmsg.GetSendMsgStatusReq, opts ...grpc.CallOption) (*pbmsg.GetSendMsgStatusResp, error)
	// 撤回消息
	RevokeMsg(ctx context.Context, in *pbmsg.RevokeMsgReq, opts ...grpc.CallOption) (*pbmsg.RevokeMsgResp, error)
	// 标记消息为已读
	MarkMsgsAsRead(ctx context.Context, in *pbmsg.MarkMsgsAsReadReq, opts ...grpc.CallOption) (*pbmsg.MarkMsgsAsReadResp, error)
	// 标记会话为已读
	MarkConversationAsRead(ctx context.Context, in *pbmsg.MarkConversationAsReadReq, opts ...grpc.CallOption) (*pbmsg.MarkConversationAsReadResp, error)
	// 设置会话已读序列号
	SetConversationHasReadSeq(ctx context.Context, in *pbmsg.SetConversationHasReadSeqReq, opts ...grpc.CallOption) (*pbmsg.SetConversationHasReadSeqResp, error)
	// 获取会话已读和最大序列号
	GetConversationsHasReadAndMaxSeq(ctx context.Context, in *pbmsg.GetConversationsHasReadAndMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationsHasReadAndMaxSeqResp, error)
	// 获取活跃用户
	GetActiveUser(ctx context.Context, in *pbmsg.GetActiveUserReq, opts ...grpc.CallOption) (*pbmsg.GetActiveUserResp, error)
	// 获取活跃群组
	GetActiveGroup(ctx context.Context, in *pbmsg.GetActiveGroupReq, opts ...grpc.CallOption) (*pbmsg.GetActiveGroupResp, error)
	// 获取服务器时间
	GetServerTime(ctx context.Context, in *pbmsg.GetServerTimeReq, opts ...grpc.CallOption) (*pbmsg.GetServerTimeResp, error)
	// 清理消息
	ClearMsg(ctx context.Context, in *pbmsg.ClearMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearMsgResp, error)
	// 销毁消息
	DestructMsgs(ctx context.Context, in *pbmsg.DestructMsgsReq, opts ...grpc.CallOption) (*pbmsg.DestructMsgsResp, error)
	// 获取活跃会话
	GetActiveConversation(ctx context.Context, in *pbmsg.GetActiveConversationReq, opts ...grpc.CallOption) (*pbmsg.GetActiveConversationResp, error)
	// 设置用户会话最大序列号
	SetUserConversationMaxSeq(ctx context.Context, in *pbmsg.SetUserConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMaxSeqResp, error)
	// 设置用户会话最小序列号
	SetUserConversationMinSeq(ctx context.Context, in *pbmsg.SetUserConversationMinSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMinSeqResp, error)
	// 获取指定时间的最后消息序列号
	GetLastMessageSeqByTime(ctx context.Context, in *pbmsg.GetLastMessageSeqByTimeReq, opts ...grpc.CallOption) (*pbmsg.GetLastMessageSeqByTimeResp, error)
	// 获取最后消息
	GetLastMessage(ctx context.Context, in *pbmsg.GetLastMessageReq, opts ...grpc.CallOption) (*pbmsg.GetLastMessageResp, error)
	// 新增消息
	AddMsg(ctx context.Context, in *pbmsg.AddMsgReq, opts ...grpc.CallOption) (*pbmsg.AddMsgResp, error)
	// 批量新增消息
	AddMsgs(ctx context.Context, in *pbmsg.AddMsgsReq, opts ...grpc.CallOption) (*pbmsg.AddMsgsResp, error)
}

type defaultMsgService struct {
	cli zrpc.Client
}

func NewMsgService(cli zrpc.Client) MsgService {
	return &defaultMsgService{cli: cli}
}

// 获取最大序列号(包含用户和指定群组)
func (s *defaultMsgService) GetMaxSeq(ctx context.Context, in *sdkws.GetMaxSeqReq, opts ...grpc.CallOption) (*sdkws.GetMaxSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetMaxSeq(ctx, in, opts...)
}

// 获取会话列表的最大序列号
func (s *defaultMsgService) GetMaxSeqs(ctx context.Context, in *pbmsg.GetMaxSeqsReq, opts ...grpc.CallOption) (*pbmsg.SeqsInfoResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetMaxSeqs(ctx, in, opts...)
}

// 获取会话列表的已读序列号
func (s *defaultMsgService) GetHasReadSeqs(ctx context.Context, in *pbmsg.GetHasReadSeqsReq, opts ...grpc.CallOption) (*pbmsg.SeqsInfoResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetHasReadSeqs(ctx, in, opts...)
}

// 获取最新消息
func (s *defaultMsgService) GetMsgByConversationIDs(ctx context.Context, in *pbmsg.GetMsgByConversationIDsReq, opts ...grpc.CallOption) (*pbmsg.GetMsgByConversationIDsResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetMsgByConversationIDs(ctx, in, opts...)
}

// 获取会话最大序列号
func (s *defaultMsgService) GetConversationMaxSeq(ctx context.Context, in *pbmsg.GetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationMaxSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetConversationMaxSeq(ctx, in, opts...)
}

// 拉取历史消息(包含用户和指定群组)
func (s *defaultMsgService) PullMessageBySeqs(ctx context.Context, in *sdkws.PullMessageBySeqsReq, opts ...grpc.CallOption) (*sdkws.PullMessageBySeqsResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.PullMessageBySeqs(ctx, in, opts...)
}

// 获取序列号消息
func (s *defaultMsgService) GetSeqMessage(ctx context.Context, in *pbmsg.GetSeqMessageReq, opts ...grpc.CallOption) (*pbmsg.GetSeqMessageResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetSeqMessage(ctx, in, opts...)
}

// 搜索消息
func (s *defaultMsgService) SearchMessage(ctx context.Context, in *pbmsg.SearchMessageReq, opts ...grpc.CallOption) (*pbmsg.SearchMessageResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SearchMessage(ctx, in, opts...)
}

// 发送消息
func (s *defaultMsgService) SendMsg(ctx context.Context, in *pbmsg.SendMsgReq, opts ...grpc.CallOption) (*pbmsg.SendMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SendMsg(ctx, in, opts...)
}

// 发送简单消息
func (s *defaultMsgService) SendSimpleMsg(ctx context.Context, in *pbmsg.SendSimpleMsgReq, opts ...grpc.CallOption) (*pbmsg.SendSimpleMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SendSimpleMsg(ctx, in, opts...)
}

// 设置用户会话最小序列号
func (s *defaultMsgService) SetUserConversationsMinSeq(ctx context.Context, in *pbmsg.SetUserConversationsMinSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationsMinSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SetUserConversationsMinSeq(ctx, in, opts...)
}

// 清空指定会话的所有消息并重置最小序列号
func (s *defaultMsgService) ClearConversationsMsg(ctx context.Context, in *pbmsg.ClearConversationsMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearConversationsMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.ClearConversationsMsg(ctx, in, opts...)
}

// 用户清空所有消息并重置最小序列号
func (s *defaultMsgService) UserClearAllMsg(ctx context.Context, in *pbmsg.UserClearAllMsgReq, opts ...grpc.CallOption) (*pbmsg.UserClearAllMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.UserClearAllMsg(ctx, in, opts...)
}

// 按序列号标记消息删除
func (s *defaultMsgService) DeleteMsgs(ctx context.Context, in *pbmsg.DeleteMsgsReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgsResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.DeleteMsgs(ctx, in, opts...)
}

// 按序列号物理删除消息
func (s *defaultMsgService) DeleteMsgPhysicalBySeq(ctx context.Context, in *pbmsg.DeleteMsgPhysicalBySeqReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgPhysicalBySeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.DeleteMsgPhysicalBySeq(ctx, in, opts...)
}

// 按时间戳物理删除消息
func (s *defaultMsgService) DeleteMsgPhysical(ctx context.Context, in *pbmsg.DeleteMsgPhysicalReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgPhysicalResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.DeleteMsgPhysical(ctx, in, opts...)
}

// 设置API发送消息的送达状态
func (s *defaultMsgService) SetSendMsgStatus(ctx context.Context, in *pbmsg.SetSendMsgStatusReq, opts ...grpc.CallOption) (*pbmsg.SetSendMsgStatusResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SetSendMsgStatus(ctx, in, opts...)
}

// 获取消息送达状态
func (s *defaultMsgService) GetSendMsgStatus(ctx context.Context, in *pbmsg.GetSendMsgStatusReq, opts ...grpc.CallOption) (*pbmsg.GetSendMsgStatusResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetSendMsgStatus(ctx, in, opts...)
}

// 撤回消息
func (s *defaultMsgService) RevokeMsg(ctx context.Context, in *pbmsg.RevokeMsgReq, opts ...grpc.CallOption) (*pbmsg.RevokeMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.RevokeMsg(ctx, in, opts...)
}

// 标记消息为已读
func (s *defaultMsgService) MarkMsgsAsRead(ctx context.Context, in *pbmsg.MarkMsgsAsReadReq, opts ...grpc.CallOption) (*pbmsg.MarkMsgsAsReadResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.MarkMsgsAsRead(ctx, in, opts...)
}

// 标记会话为已读
func (s *defaultMsgService) MarkConversationAsRead(ctx context.Context, in *pbmsg.MarkConversationAsReadReq, opts ...grpc.CallOption) (*pbmsg.MarkConversationAsReadResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.MarkConversationAsRead(ctx, in, opts...)
}

// 设置会话已读序列号
func (s *defaultMsgService) SetConversationHasReadSeq(ctx context.Context, in *pbmsg.SetConversationHasReadSeqReq, opts ...grpc.CallOption) (*pbmsg.SetConversationHasReadSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SetConversationHasReadSeq(ctx, in, opts...)
}

// 获取会话已读和最大序列号
func (s *defaultMsgService) GetConversationsHasReadAndMaxSeq(ctx context.Context, in *pbmsg.GetConversationsHasReadAndMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationsHasReadAndMaxSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetConversationsHasReadAndMaxSeq(ctx, in, opts...)
}

// 获取活跃用户
func (s *defaultMsgService) GetActiveUser(ctx context.Context, in *pbmsg.GetActiveUserReq, opts ...grpc.CallOption) (*pbmsg.GetActiveUserResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetActiveUser(ctx, in, opts...)
}

// 获取活跃群组
func (s *defaultMsgService) GetActiveGroup(ctx context.Context, in *pbmsg.GetActiveGroupReq, opts ...grpc.CallOption) (*pbmsg.GetActiveGroupResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetActiveGroup(ctx, in, opts...)
}

// 获取服务器时间
func (s *defaultMsgService) GetServerTime(ctx context.Context, in *pbmsg.GetServerTimeReq, opts ...grpc.CallOption) (*pbmsg.GetServerTimeResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetServerTime(ctx, in, opts...)
}

// 清理消息
func (s *defaultMsgService) ClearMsg(ctx context.Context, in *pbmsg.ClearMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.ClearMsg(ctx, in, opts...)
}

// 销毁消息
func (s *defaultMsgService) DestructMsgs(ctx context.Context, in *pbmsg.DestructMsgsReq, opts ...grpc.CallOption) (*pbmsg.DestructMsgsResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.DestructMsgs(ctx, in, opts...)
}

// 将GetActiveConversation方法的接收者从d改为s，并添加authClient调用逻辑
func (s *defaultMsgService) GetActiveConversation(ctx context.Context, in *pbmsg.GetActiveConversationReq, opts ...grpc.CallOption) (*pbmsg.GetActiveConversationResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetActiveConversation(ctx, in, opts...)
}

// 将SetUserConversationMaxSeq方法的接收者从d改为s，并添加authClient调用逻辑
func (s *defaultMsgService) SetUserConversationMaxSeq(ctx context.Context, in *pbmsg.SetUserConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SetUserConversationMaxSeq(ctx, in, opts...)
}

// 设置用户会话最小序列号
func (s *defaultMsgService) SetUserConversationMinSeq(ctx context.Context, in *pbmsg.SetUserConversationMinSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMinSeqResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.SetUserConversationMinSeq(ctx, in, opts...)
}

// 获取指定时间的最后消息序列号
func (s *defaultMsgService) GetLastMessageSeqByTime(ctx context.Context, in *pbmsg.GetLastMessageSeqByTimeReq, opts ...grpc.CallOption) (*pbmsg.GetLastMessageSeqByTimeResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetLastMessageSeqByTime(ctx, in, opts...)
}

// 获取最后消息
func (s *defaultMsgService) GetLastMessage(ctx context.Context, in *pbmsg.GetLastMessageReq, opts ...grpc.CallOption) (*pbmsg.GetLastMessageResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.GetLastMessage(ctx, in, opts...)
}

// 新增消息
func (s *defaultMsgService) AddMsg(ctx context.Context, in *pbmsg.AddMsgReq, opts ...grpc.CallOption) (*pbmsg.AddMsgResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.AddMsg(ctx, in, opts...)
}

// 批量新增消息
func (s *defaultMsgService) AddMsgs(ctx context.Context, in *pbmsg.AddMsgsReq, opts ...grpc.CallOption) (*pbmsg.AddMsgsResp, error) {
	authClient := pbmsg.NewMsgClient(s.cli.Conn())
	return authClient.AddMsgs(ctx, in, opts...)
}

// stub
type stubMsgService struct {
}

func NewStubMsgService() MsgService {
	return &stubMsgService{}
}

// 获取最大序列号(包含用户和指定群组)
func (s *stubMsgService) GetMaxSeq(ctx context.Context, in *sdkws.GetMaxSeqReq, opts ...grpc.CallOption) (*sdkws.GetMaxSeqResp, error) {
	return &sdkws.GetMaxSeqResp{}, nil
}

// 获取会话列表的最大序列号
func (s *stubMsgService) GetMaxSeqs(ctx context.Context, in *pbmsg.GetMaxSeqsReq, opts ...grpc.CallOption) (*pbmsg.SeqsInfoResp, error) {
	return &pbmsg.SeqsInfoResp{}, nil
}

// 获取会话列表的已读序列号
func (s *stubMsgService) GetHasReadSeqs(ctx context.Context, in *pbmsg.GetHasReadSeqsReq, opts ...grpc.CallOption) (*pbmsg.SeqsInfoResp, error) {
	return &pbmsg.SeqsInfoResp{}, nil
}

// 获取最新消息
func (s *stubMsgService) GetMsgByConversationIDs(ctx context.Context, in *pbmsg.GetMsgByConversationIDsReq, opts ...grpc.CallOption) (*pbmsg.GetMsgByConversationIDsResp, error) {
	return &pbmsg.GetMsgByConversationIDsResp{}, nil
}

// 获取会话最大序列号
func (s *stubMsgService) GetConversationMaxSeq(ctx context.Context, in *pbmsg.GetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationMaxSeqResp, error) {
	return &pbmsg.GetConversationMaxSeqResp{}, nil
}

// 拉取历史消息(包含用户和指定群组)
func (s *stubMsgService) PullMessageBySeqs(ctx context.Context, in *sdkws.PullMessageBySeqsReq, opts ...grpc.CallOption) (*sdkws.PullMessageBySeqsResp, error) {
	return &sdkws.PullMessageBySeqsResp{}, nil
}

// 获取序列号消息
func (s *stubMsgService) GetSeqMessage(ctx context.Context, in *pbmsg.GetSeqMessageReq, opts ...grpc.CallOption) (*pbmsg.GetSeqMessageResp, error) {
	return &pbmsg.GetSeqMessageResp{}, nil
}

// 搜索消息
func (s *stubMsgService) SearchMessage(ctx context.Context, in *pbmsg.SearchMessageReq, opts ...grpc.CallOption) (*pbmsg.SearchMessageResp, error) {
	return &pbmsg.SearchMessageResp{}, nil
}

// 发送消息
func (s *stubMsgService) SendMsg(ctx context.Context, in *pbmsg.SendMsgReq, opts ...grpc.CallOption) (*pbmsg.SendMsgResp, error) {
	return &pbmsg.SendMsgResp{}, nil
}

// 发送简单消息
func (s *stubMsgService) SendSimpleMsg(ctx context.Context, in *pbmsg.SendSimpleMsgReq, opts ...grpc.CallOption) (*pbmsg.SendSimpleMsgResp, error) {
	return &pbmsg.SendSimpleMsgResp{}, nil
}

// 设置用户会话最小序列号
func (s *stubMsgService) SetUserConversationsMinSeq(ctx context.Context, in *pbmsg.SetUserConversationsMinSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationsMinSeqResp, error) {
	return &pbmsg.SetUserConversationsMinSeqResp{}, nil
}

// 清空指定会话的所有消息并重置最小序列号
func (s *stubMsgService) ClearConversationsMsg(ctx context.Context, in *pbmsg.ClearConversationsMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearConversationsMsgResp, error) {
	return &pbmsg.ClearConversationsMsgResp{}, nil
}

// 用户清空所有消息并重置最小序列号
func (s *stubMsgService) UserClearAllMsg(ctx context.Context, in *pbmsg.UserClearAllMsgReq, opts ...grpc.CallOption) (*pbmsg.UserClearAllMsgResp, error) {
	return &pbmsg.UserClearAllMsgResp{}, nil
}

// 按序列号标记消息删除
func (s *stubMsgService) DeleteMsgs(ctx context.Context, in *pbmsg.DeleteMsgsReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgsResp, error) {
	return &pbmsg.DeleteMsgsResp{}, nil
}

// 按序列号物理删除消息
func (s *stubMsgService) DeleteMsgPhysicalBySeq(ctx context.Context, in *pbmsg.DeleteMsgPhysicalBySeqReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgPhysicalBySeqResp, error) {
	return &pbmsg.DeleteMsgPhysicalBySeqResp{}, nil
}

// 按时间戳物理删除消息
func (s *stubMsgService) DeleteMsgPhysical(ctx context.Context, in *pbmsg.DeleteMsgPhysicalReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgPhysicalResp, error) {
	return &pbmsg.DeleteMsgPhysicalResp{}, nil
}

// 设置API发送消息的送达状态
func (s *stubMsgService) SetSendMsgStatus(ctx context.Context, in *pbmsg.SetSendMsgStatusReq, opts ...grpc.CallOption) (*pbmsg.SetSendMsgStatusResp, error) {
	return &pbmsg.SetSendMsgStatusResp{}, nil
}

// 获取消息送达状态
func (s *stubMsgService) GetSendMsgStatus(ctx context.Context, in *pbmsg.GetSendMsgStatusReq, opts ...grpc.CallOption) (*pbmsg.GetSendMsgStatusResp, error) {
	return &pbmsg.GetSendMsgStatusResp{}, nil
}

// 撤回消息
func (s *stubMsgService) RevokeMsg(ctx context.Context, in *pbmsg.RevokeMsgReq, opts ...grpc.CallOption) (*pbmsg.RevokeMsgResp, error) {
	return &pbmsg.RevokeMsgResp{}, nil
}

// 标记消息为已读
func (s *stubMsgService) MarkMsgsAsRead(ctx context.Context, in *pbmsg.MarkMsgsAsReadReq, opts ...grpc.CallOption) (*pbmsg.MarkMsgsAsReadResp, error) {
	return &pbmsg.MarkMsgsAsReadResp{}, nil
}

// 标记会话为已读
func (s *stubMsgService) MarkConversationAsRead(ctx context.Context, in *pbmsg.MarkConversationAsReadReq, opts ...grpc.CallOption) (*pbmsg.MarkConversationAsReadResp, error) {
	return &pbmsg.MarkConversationAsReadResp{}, nil
}

// 设置会话已读序列号
func (s *stubMsgService) SetConversationHasReadSeq(ctx context.Context, in *pbmsg.SetConversationHasReadSeqReq, opts ...grpc.CallOption) (*pbmsg.SetConversationHasReadSeqResp, error) {
	return &pbmsg.SetConversationHasReadSeqResp{}, nil
}

// 获取会话已读和最大序列号
func (s *stubMsgService) GetConversationsHasReadAndMaxSeq(ctx context.Context, in *pbmsg.GetConversationsHasReadAndMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationsHasReadAndMaxSeqResp, error) {
	return &pbmsg.GetConversationsHasReadAndMaxSeqResp{}, nil
}

// 获取活跃用户
func (s *stubMsgService) GetActiveUser(ctx context.Context, in *pbmsg.GetActiveUserReq, opts ...grpc.CallOption) (*pbmsg.GetActiveUserResp, error) {
	return &pbmsg.GetActiveUserResp{}, nil
}

// 获取活跃群组
func (s *stubMsgService) GetActiveGroup(ctx context.Context, in *pbmsg.GetActiveGroupReq, opts ...grpc.CallOption) (*pbmsg.GetActiveGroupResp, error) {
	return &pbmsg.GetActiveGroupResp{}, nil
}

// 获取服务器时间
func (s *stubMsgService) GetServerTime(ctx context.Context, in *pbmsg.GetServerTimeReq, opts ...grpc.CallOption) (*pbmsg.GetServerTimeResp, error) {
	return &pbmsg.GetServerTimeResp{}, nil
}

// 清理消息
func (s *stubMsgService) ClearMsg(ctx context.Context, in *pbmsg.ClearMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearMsgResp, error) {
	return &pbmsg.ClearMsgResp{}, nil
}

// 销毁消息
func (s *stubMsgService) DestructMsgs(ctx context.Context, in *pbmsg.DestructMsgsReq, opts ...grpc.CallOption) (*pbmsg.DestructMsgsResp, error) {
	return &pbmsg.DestructMsgsResp{}, nil
}

// 将GetActiveConversation方法的接收者从d改为s，并添加authClient调用逻辑
func (s *stubMsgService) GetActiveConversation(ctx context.Context, in *pbmsg.GetActiveConversationReq, opts ...grpc.CallOption) (*pbmsg.GetActiveConversationResp, error) {
	return &pbmsg.GetActiveConversationResp{}, nil
}

// 将SetUserConversationMaxSeq方法的接收者从d改为s，并添加authClient调用逻辑
func (s *stubMsgService) SetUserConversationMaxSeq(ctx context.Context, in *pbmsg.SetUserConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	return &pbmsg.SetUserConversationMaxSeqResp{}, nil
}

// 设置用户会话最小序列号
func (s *stubMsgService) SetUserConversationMinSeq(ctx context.Context, in *pbmsg.SetUserConversationMinSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMinSeqResp, error) {
	return &pbmsg.SetUserConversationMinSeqResp{}, nil
}

// 获取指定时间的最后消息序列号
func (s *stubMsgService) GetLastMessageSeqByTime(ctx context.Context, in *pbmsg.GetLastMessageSeqByTimeReq, opts ...grpc.CallOption) (*pbmsg.GetLastMessageSeqByTimeResp, error) {
	return &pbmsg.GetLastMessageSeqByTimeResp{}, nil
}

// 获取最后消息
func (s *stubMsgService) GetLastMessage(ctx context.Context, in *pbmsg.GetLastMessageReq, opts ...grpc.CallOption) (*pbmsg.GetLastMessageResp, error) {
	return &pbmsg.GetLastMessageResp{}, nil
}

// 新增消息
func (s *stubMsgService) AddMsg(ctx context.Context, in *pbmsg.AddMsgReq, opts ...grpc.CallOption) (*pbmsg.AddMsgResp, error) {
	return &pbmsg.AddMsgResp{}, nil
}

// 批量新增消息
func (s *stubMsgService) AddMsgs(ctx context.Context, in *pbmsg.AddMsgsReq, opts ...grpc.CallOption) (*pbmsg.AddMsgsResp, error) {
	return &pbmsg.AddMsgsResp{}, nil
}
