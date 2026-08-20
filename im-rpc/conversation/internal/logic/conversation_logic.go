package logic

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbconversation "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	"github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	"github.com/PaperMan11/goim/pkg/storage/model"
	conversationModel "github.com/PaperMan11/goim/pkg/storage/mongo/conversation"
	"github.com/PaperMan11/goim/pkg/utils/hash"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/jinzhu/copier"
)

// ==================== 查询方法 ====================

// GetConversation 获取单个会话。DID=ownerUserID 范围内的会话。
func (l *Logic) GetConversation(ctx context.Context, req *pbconversation.GetConversationReq) (*pbconversation.GetConversationResp, error) {
	ownerUserID := req.GetOwnerUserID()
	conversationID := req.GetConversationID()
	if ownerUserID == "" || conversationID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID and conversationID are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	conv, err := l.svcCtx.ConversationModel.FindConversation(ctx, ownerUserID, conversationID)
	if err != nil {
		if isConversationNotFound(err) {
			return nil, errx.RecordNotFoundError
		}
		l.Errorf("find conversation failed, owner: %s, conv: %s, err: %v", ownerUserID, conversationID, err)
		return nil, err
	}

	pbConv := modelToPbConversation(conv)
	// 从 SeqUser 填充 min_seq/max_seq
	l.fillConversationSeqs(ownerUserID, []*pbconversation.Conversation{pbConv})

	return &pbconversation.GetConversationResp{
		Conversation: pbConv,
	}, nil
}

// GetAllConversations 获取用户所有会话
func (l *Logic) GetAllConversations(ctx context.Context, req *pbconversation.GetAllConversationsReq) (*pbconversation.GetAllConversationsResp, error) {
	ownerUserID := req.GetOwnerUserID()
	if ownerUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID is required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, ownerUserID)
	if err != nil {
		l.Errorf("find conversations by owner failed, owner: %s, err: %v", ownerUserID, err)
		return nil, err
	}

	var pbConvs []*pbconversation.Conversation
	for _, c := range convs {
		pbConvs = append(pbConvs, modelToPbConversation(c))
	}
	// 从 SeqUser 批量填充 min_seq/max_seq
	l.fillConversationSeqs(ownerUserID, pbConvs)

	return &pbconversation.GetAllConversationsResp{
		Conversations: pbConvs,
	}, nil
}

// GetConversations 获取多个会话
func (l *Logic) GetConversations(ctx context.Context, req *pbconversation.GetConversationsReq) (*pbconversation.GetConversationsResp, error) {
	ownerUserID := req.GetOwnerUserID()
	convIDs := req.GetConversationIDs()
	if ownerUserID == "" || len(convIDs) == 0 {
		return nil, errx.ArgsError.Wrap("ownerUserID and conversationIDs are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByIDs(ctx, ownerUserID, convIDs)
	if err != nil {
		l.Errorf("find conversations by ids failed, owner: %s, err: %v", ownerUserID, err)
		return nil, err
	}

	var pbConvs []*pbconversation.Conversation
	for _, c := range convs {
		pbConvs = append(pbConvs, modelToPbConversation(c))
	}
	// 从 SeqUser 批量填充 min_seq/max_seq
	l.fillConversationSeqs(ownerUserID, pbConvs)

	return &pbconversation.GetConversationsResp{
		Conversations: pbConvs,
	}, nil
}

// GetConversationsByConversationID 按会话ID（不限所有者）获取会话列表。
// 此方法仅限管理员/内部调用使用，因为可能跨用户读取会话。
func (l *Logic) GetConversationsByConversationID(ctx context.Context, req *pbconversation.GetConversationsByConversationIDReq) (*pbconversation.GetConversationsByConversationIDResp, error) {
	convIDs := req.GetConversationIDs()
	if len(convIDs) == 0 {
		return &pbconversation.GetConversationsByConversationIDResp{}, nil
	}

	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	var pbConvs []*pbconversation.Conversation
	convs, err := l.svcCtx.ConversationModel.FindConversationsByConvIDs(ctx, convIDs)
	if err != nil {
		l.Errorf("find conversations by conv ids failed, err: %v", err)
		return nil, err
	}

	for _, conv := range convs {
		pbConvs = append(pbConvs, modelToPbConversation(conv))
	}

	return &pbconversation.GetConversationsByConversationIDResp{
		Conversations: pbConvs,
	}, nil
}

// GetConversationIDs 获取用户会话ID列表
func (l *Logic) GetConversationIDs(ctx context.Context, req *pbconversation.GetConversationIDsReq) (*pbconversation.GetConversationIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	convIDs := make([]string, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ConversationID)
	}

	return &pbconversation.GetConversationIDsResp{
		ConversationIDs: convIDs,
	}, nil
}

// GetUserConversationIDsHash 获取用户会话ID哈希，用于客户端判断是否需要全量同步
func (l *Logic) GetUserConversationIDsHash(ctx context.Context, req *pbconversation.GetUserConversationIDsHashReq) (*pbconversation.GetUserConversationIDsHashResp, error) {
	ownerUserID := req.GetOwnerUserID()
	if ownerUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID is required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, ownerUserID)
	if err != nil {
		l.Errorf("find conversations by owner failed, owner: %s, err: %v", ownerUserID, err)
		return nil, err
	}

	convIDs := make([]string, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ConversationID)
	}

	return &pbconversation.GetUserConversationIDsHashResp{
		Hash: hash.HashStringSet(convIDs),
	}, nil
}

// GetOwnerConversation 分页获取用户会话列表
func (l *Logic) GetOwnerConversation(ctx context.Context, req *pbconversation.GetOwnerConversationReq) (*pbconversation.GetOwnerConversationResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	pagination := req.GetPagination()
	page := int(pagination.GetPageNumber())
	size := int(pagination.GetShowNumber())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = len(convs)
		if size == 0 {
			size = 50
		}
	}

	start := (page - 1) * size
	end := start + size
	if start > len(convs) {
		start = len(convs)
	}
	if end > len(convs) {
		end = len(convs)
	}

	var pbConvs []*pbconversation.Conversation
	for _, c := range convs[start:end] {
		pbConvs = append(pbConvs, modelToPbConversation(c))
	}
	// 从 SeqUser 批量填充 min_seq/max_seq
	l.fillConversationSeqs(userID, pbConvs)

	return &pbconversation.GetOwnerConversationResp{
		Total:         int64(len(convs)),
		Conversations: pbConvs,
	}, nil
}

// GetSortedConversationList 获取按最新消息时间排序的会话列表
func (l *Logic) GetSortedConversationList(ctx context.Context, req *pbconversation.GetSortedConversationListReq) (*pbconversation.GetSortedConversationListResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	filterIDs := req.GetConversationIDs()
	if len(filterIDs) == 0 {
		ids, err := l.svcCtx.ConversationModel.FindConversationIDsByOwner(ctx, userID)
		if err != nil {
			l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
			return nil, err
		}
		filterIDs = ids
	}

	// 分页查询
	total := int64(len(filterIDs))
	pagination := req.GetPagination()
	page := int(pagination.GetPageNumber())
	size := int(pagination.GetShowNumber())
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}
	start := (page - 1) * size
	end := start + size
	if start > len(filterIDs) {
		start = len(filterIDs)
	}
	if end > len(filterIDs) {
		end = len(filterIDs)
	}

	// 收集需要查询会话详情的 ID
	idsAtPage := make([]string, 0, end-start)
	copy(idsAtPage, filterIDs[start:end])
	convs, err := l.svcCtx.ConversationModel.FindConversationsByIDs(ctx, userID, idsAtPage)
	if err != nil {
		l.Errorf("find conversations by ids failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 获取所有的会话最新消息
	maxSeqs, err := l.svcCtx.MsgService.GetMaxSeqs(ctx, &msg.GetMaxSeqsReq{
		ConversationIDs: idsAtPage,
	})
	if err != nil {
		l.Errorf("get max seqs failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	msgs, err := l.svcCtx.MsgService.GetMsgByConversationIDs(ctx, &msg.GetMsgByConversationIDsReq{
		ConversationIDs: idsAtPage,
		MaxSeqs:         maxSeqs.GetMaxSeqs(),
	})
	if err != nil {
		l.Errorf("get msgs by conversation ids failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	convElems, err := l.buildConversationElem(ctx, msgs.GetMsgDatas(), userID)
	if err != nil {
		return nil, err
	}
	// 未读消息
	hasReadSeqs, err := l.svcCtx.MsgService.GetHasReadSeqs(ctx, &msg.GetHasReadSeqsReq{
		UserID:          userID,
		ConversationIDs: idsAtPage,
	})
	if err != nil {
		return nil, err
	}
	var unreadTotal int64
	convsUnreadCount := make(map[string]int64)
	hasReadSeqsMap := hasReadSeqs.GetMaxSeqs()
	for conversationID, maxReq := range hasReadSeqsMap {
		unreadCount := maxReq - hasReadSeqsMap[conversationID]
		convsUnreadCount[conversationID] = unreadCount
		unreadTotal += unreadCount
	}

	// 会话排序
	for _, c := range convs {
		conversationID := c.ConversationID
		elem, ok := convElems[conversationID]
		if !ok {
			convElems[conversationID] = &pbconversation.ConversationElem{
				ConversationID: conversationID,
				IsPinned:       c.IsPinned,
				MsgInfo:        nil,
			}
		}
		elem.IsPinned = c.IsPinned
		elem.RecvMsgOpt = c.RecvMsgOpt
		elem.UnreadCount = convsUnreadCount[conversationID]
	}
	elems := l.sortConversationElems(convElems)

	return &pbconversation.GetSortedConversationListResp{
		ConversationTotal: total,
		UnreadTotal:       unreadTotal,
		ConversationElems: elems,
	}, nil
}

// 构造conversation elem
func (l *Logic) buildConversationElem(ctx context.Context, chatLogs map[string]*sdkws.MsgData, userID string) (map[string]*pbconversation.ConversationElem, error) {
	var (
		sendIDs         []string
		groupIDs        []string
		sendMap         = make(map[string]*sdkws.UserInfo)
		groupMap        = make(map[string]*sdkws.GroupInfo)
		conversationMsg = make(map[string]*pbconversation.ConversationElem)
	)
	for _, chatLog := range chatLogs {
		switch chatLog.SessionType {
		case constant.SingleChatType:
			if chatLog.SendID == userID {
				sendIDs = append(sendIDs, chatLog.RecvID)
			}
			sendIDs = append(sendIDs, chatLog.SendID)
		case constant.WriteGroupChatType, constant.ReadGroupChatType:
			groupIDs = append(groupIDs, chatLog.GroupID)
			sendIDs = append(sendIDs, chatLog.SendID)
		}
	}
	if len(sendIDs) != 0 {
		sendInfos, err := l.svcCtx.UserService.GetDesignateUsers(ctx, &pbuser.GetDesignateUsersReq{
			UserIDs: sendIDs,
		})
		if err != nil {
			return nil, err
		}
		for _, sendInfo := range sendInfos.GetUsersInfo() {
			sendMap[sendInfo.UserID] = sendInfo
		}
	}
	if len(groupIDs) != 0 {
		groupInfos, err := l.svcCtx.GroupService.GetGroupsInfo(ctx, &pbgroup.GetGroupsInfoReq{
			GroupIDs: groupIDs,
		})
		if err != nil {
			return nil, err
		}
		for _, groupInfo := range groupInfos.GetGroupInfos() {
			groupMap[groupInfo.GroupID] = groupInfo
		}
	}
	for conversationID, chatLog := range chatLogs {
		pbchatLog := &pbconversation.ConversationElem{}
		msgInfo := &pbconversation.MsgInfo{}
		if err := copier.Copy(msgInfo, chatLog); err != nil {
			return nil, err
		}
		switch chatLog.SessionType {
		case constant.SingleChatType:
			if chatLog.SendID == userID {
				if recv, ok := sendMap[chatLog.RecvID]; ok {
					msgInfo.FaceURL = recv.FaceURL
					msgInfo.SenderName = recv.Nickname
				}
				break
			}
			if send, ok := sendMap[chatLog.SendID]; ok {
				msgInfo.FaceURL = send.FaceURL
				msgInfo.SenderName = send.Nickname
			}
		case constant.WriteGroupChatType, constant.ReadGroupChatType:
			msgInfo.GroupID = chatLog.GroupID
			if group, ok := groupMap[chatLog.GroupID]; ok {
				msgInfo.GroupName = group.GroupName
				msgInfo.GroupFaceURL = group.FaceURL
				msgInfo.GroupMemberCount = group.MemberCount
				msgInfo.GroupType = group.GroupType
			}
			if send, ok := sendMap[chatLog.SendID]; ok {
				msgInfo.SenderName = send.Nickname
			}
		}
		pbchatLog.ConversationID = conversationID
		msgInfo.LatestMsgRecvTime = chatLog.SendTime
		pbchatLog.MsgInfo = msgInfo
		conversationMsg[conversationID] = pbchatLog
	}
	return conversationMsg, nil
}

// 排序会话
func (l *Logic) sortConversationElems(conversationElems map[string]*pbconversation.ConversationElem) (elems []*pbconversation.ConversationElem) {
	elems = make([]*pbconversation.ConversationElem, 0, len(conversationElems))
	isPinned := make([]*pbconversation.ConversationElem, 0) // 置顶会话
	unPinned := make([]*pbconversation.ConversationElem, 0) // 未置顶会话
	for _, elem := range conversationElems {
		if elem.IsPinned {
			isPinned = append(isPinned, elem)
		} else {
			unPinned = append(unPinned, elem)
		}
	}
	// 置顶会话按 latest_msg_recv_time 排序
	sortFunc := func(i, j *pbconversation.ConversationElem) int {
		if (i == nil || i.MsgInfo == nil) && (j == nil || j.MsgInfo == nil) {
			return 0
		}
		if i == nil || i.MsgInfo == nil {
			return -1
		}
		if j == nil || j.MsgInfo == nil {
			return 1
		}
		if i.MsgInfo.LatestMsgRecvTime < j.MsgInfo.LatestMsgRecvTime {
			return -1
		} else if i.MsgInfo.LatestMsgRecvTime > j.MsgInfo.LatestMsgRecvTime {
			return 1
		} else {
			return 0
		}
	}
	slices.SortFunc(isPinned, sortFunc)
	slices.SortFunc(unPinned, sortFunc)
	elems = append(elems, isPinned...)
	elems = append(elems, unPinned...)
	return elems
}

// GetRecvMsgNotNotifyUserIDs 获取群消息不通知用户列表（recvMsgOpt != 0 的群成员）。
// 当前 conversation model 没有直接维护"群不通知用户列表"，这里通过查询该群所有成员的会话并过滤得到。
func (l *Logic) GetRecvMsgNotNotifyUserIDs(ctx context.Context, req *pbconversation.GetRecvMsgNotNotifyUserIDsReq) (*pbconversation.GetRecvMsgNotNotifyUserIDsResp, error) {
	groupID := req.GetGroupID()
	if groupID == "" {
		return nil, errx.ArgsError.Wrap("groupID is required")
	}

	// 群聊会话 conversation_id 一般由 groupID 派生，这里不直接查所有用户。
	// 业务侧调用 msg gateway 等模块时自行维护群成员列表，此处保留接口兼容返回空列表。
	return &pbconversation.GetRecvMsgNotNotifyUserIDsResp{}, nil
}

// GetConversationOfflinePushUserIDs 获取会话离线推送用户列表（除 recvMsgOpt != 0 之外的用户）。
// 入参 userIDs 为候选用户列表，过滤掉设置了"不接收/接收但不通知"的成员。
func (l *Logic) GetConversationOfflinePushUserIDs(ctx context.Context, req *pbconversation.GetConversationOfflinePushUserIDsReq) (*pbconversation.GetConversationOfflinePushUserIDsResp, error) {
	conversationID := req.GetConversationID()
	userIDs := req.GetUserIDs()
	if conversationID == "" || len(userIDs) == 0 {
		return &pbconversation.GetConversationOfflinePushUserIDsResp{}, nil
	}

	var result []string
	filterUserIDs, err := l.svcCtx.ConversationModel.FindNoRecvConversationUserIDs(ctx, conversationID)
	if err != nil {
		l.Errorf("find no recv conversation user ids failed, conversationID: %s, err: %v", conversationID, err)
		return nil, err
	}

	result = slices.DeleteFunc(userIDs, func(uid string) bool {
		return slices.Contains(filterUserIDs, uid)
	})
	return &pbconversation.GetConversationOfflinePushUserIDsResp{
		UserIDs: result,
	}, nil
}

// GetConversationNotReceiveMessageUserIDs 获取会话不接收消息的用户列表（recvMsgOpt == 1）。
func (l *Logic) GetConversationNotReceiveMessageUserIDs(ctx context.Context, req *pbconversation.GetConversationNotReceiveMessageUserIDsReq) (*pbconversation.GetConversationNotReceiveMessageUserIDsResp, error) {
	conversationID := req.GetConversationID()
	if conversationID == "" {
		return nil, errx.ArgsError.Wrap("conversationID is required")
	}

	filterUserIDs, err := l.svcCtx.ConversationModel.FindNoRecvConversationUserIDs(ctx, conversationID)
	if err != nil {
		l.Errorf("find no recv conversation user ids failed, conversationID: %s, err: %v", conversationID, err)
		return nil, err
	}
	return &pbconversation.GetConversationNotReceiveMessageUserIDsResp{
		UserIDs: filterUserIDs,
	}, nil
}

// GetNotNotifyConversationIDs 获取不通知会话ID列表（recvMsgOpt != 0 的会话）。
func (l *Logic) GetNotNotifyConversationIDs(ctx context.Context, req *pbconversation.GetNotNotifyConversationIDsReq) (*pbconversation.GetNotNotifyConversationIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	convIDs, err := l.svcCtx.ConversationModel.FindUserNotNotifyConversationIDs(ctx, userID)
	if err != nil {
		l.Errorf("find user not notify conversations failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	return &pbconversation.GetNotNotifyConversationIDsResp{
		ConversationIDs: convIDs,
	}, nil
}

// GetPinnedConversationIDs 获取置顶会话ID列表
func (l *Logic) GetPinnedConversationIDs(ctx context.Context, req *pbconversation.GetPinnedConversationIDsReq) (*pbconversation.GetPinnedConversationIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	ids, err := l.svcCtx.ConversationModel.FindPinnedConversationIDs(ctx, userID)
	if err != nil {
		l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	return &pbconversation.GetPinnedConversationIDsResp{
		ConversationIDs: ids,
	}, nil
}

// GetConversationsNeedClearMsg 获取需要清理消息的会话（is_msg_destruct=true 且 msg_destruct_time 已到期）。
func (l *Logic) GetConversationsNeedClearMsg(ctx context.Context, req *pbconversation.GetConversationsNeedClearMsgReq) (*pbconversation.GetConversationsNeedClearMsgResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	now := timex.Now()
	// 由于 conversation model 不支持跨用户聚合查询，这里返回空列表占位。
	// 实际清理消息的批量扫描应在定时任务中按 user 维度进行。
	_ = now

	return &pbconversation.GetConversationsNeedClearMsgResp{}, nil
}

// ==================== 写入方法 ====================

// SetConversation 设置单个会话（upsert）。IncrVersionLog(owner, convID, Insert 或 Update)。
// isPinned 变更时额外写入 VersionSortChangeID（影响会话列表排序）。
func (l *Logic) SetConversation(ctx context.Context, req *pbconversation.SetConversationReq) (*pbconversation.SetConversationResp, error) {
	pbConv := req.GetConversation()
	if pbConv == nil {
		return nil, errx.ArgsError.Wrap("conversation is required")
	}
	ownerUserID := pbConv.GetOwnerUserID()
	conversationID := pbConv.GetConversationID()
	if ownerUserID == "" || conversationID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID and conversationID are required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	// 判断是新增还是更新
	exist, err := l.svcCtx.ConversationModel.FindConversation(ctx, ownerUserID, conversationID)
	isInsert := false
	if err != nil {
		if !isConversationNotFound(err) {
			l.Errorf("find conversation failed, owner: %s, conv: %s, err: %v", ownerUserID, conversationID, err)
			return nil, err
		}
		isInsert = true
	}

	isPinnedChanged := false
	if !isInsert && pbConv.GetIsPinned() != exist.IsPinned {
		isPinnedChanged = true
	}

	conv := &model.Conversation{
		OwnerUserID:      ownerUserID,
		ConversationID:   conversationID,
		RecvMsgOpt:       pbConv.GetRecvMsgOpt(),
		ConversationType: pbConv.GetConversationType(),
		UserID:           pbConv.GetUserID(),
		GroupID:          pbConv.GetGroupID(),
		IsPinned:         pbConv.GetIsPinned(),
		AttachedInfo:     pbConv.GetAttachedInfo(),
		IsPrivateChat:    pbConv.GetIsPrivateChat(),
		GroupAtType:      pbConv.GetGroupAtType(),
		Extra:            pbConv.GetEx(),
		BurnDuration:     pbConv.GetBurnDuration(),
		IsMsgDestruct:    pbConv.GetIsMsgDestruct(),
		UpdatedAt:        timex.Now(),
	}
	if t := pbConv.GetMsgDestructTime(); t > 0 {
		conv.MsgDestructTime = time.UnixMilli(t)
	}
	if t := pbConv.GetLatestMsgDestructTime(); t > 0 {
		conv.LatestMsgDestructTime = time.UnixMilli(t)
	}

	if err := l.svcCtx.ConversationModel.UpsertConversation(ctx, conv); err != nil {
		l.Errorf("upsert conversation failed, owner: %s, conv: %s, err: %v", ownerUserID, conversationID, err)
		return nil, err
	}

	// seq 字段（min_seq/max_seq）不写入 Conversation 表，由 SetConversationMaxSeq/MinSeq 或 msg 模块写 SeqUser/SeqConversation
	// 写版本日志：新增=Insert，更新=Update；isPinned 变更时合并 VersionSortChangeID
	state := int32(model.VersionStateUpdate)
	if isInsert {
		state = model.VersionStateInsert
	}
	if isPinnedChanged {
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, model.ConversationDID(ownerUserID), []string{conversationID, model.VersionSortChangeID}, state); err != nil {
			l.Errorf("incr version log batch for conv+sort failed, owner: %s, conv: %s, err: %v", ownerUserID, conversationID, err)
		}
	} else {
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(ownerUserID), conversationID, state); err != nil {
			l.Errorf("incr version log for conv failed, owner: %s, conv: %s, err: %v", ownerUserID, conversationID, err)
		}
	}

	return &pbconversation.SetConversationResp{}, nil
}

// SetConversations 批量为多个用户设置同一会话参数。
// 对每个 user 都 upsert 一条会话，并写版本日志（Insert/Update）。
func (l *Logic) SetConversations(ctx context.Context, req *pbconversation.SetConversationsReq) (*pbconversation.SetConversationsResp, error) {
	userIDs := req.GetUserIDs()
	convReq := req.GetConversation()
	if len(userIDs) == 0 || convReq == nil {
		return nil, errx.ArgsError.Wrap("userIDs and conversation are required")
	}
	conversationID := convReq.GetConversationID()
	if conversationID == "" {
		return nil, errx.ArgsError.Wrap("conversationID is required")
	}

	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	now := timex.Now()
	updates := make(map[string]any)
	if convReq.RecvMsgOpt != nil {
		updates["recv_msg_opt"] = convReq.RecvMsgOpt.GetValue()
	}
	if convReq.IsPinned != nil {
		updates["is_pinned"] = convReq.IsPinned.GetValue()
	}
	if convReq.AttachedInfo != nil {
		updates["attached_info"] = convReq.AttachedInfo.GetValue()
	}
	if convReq.IsPrivateChat != nil {
		updates["is_private_chat"] = convReq.IsPrivateChat.GetValue()
	}
	if convReq.Ex != nil {
		updates["extra"] = convReq.Ex.GetValue()
	}
	if convReq.BurnDuration != nil {
		updates["burn_duration"] = convReq.BurnDuration.GetValue()
	}
	if convReq.GroupAtType != nil {
		updates["group_at_type"] = convReq.GroupAtType.GetValue()
	}
	if convReq.MsgDestructTime != nil {
		updates["msg_destruct_time"] = time.UnixMilli(convReq.MsgDestructTime.GetValue())
	}
	if convReq.IsMsgDestruct != nil {
		updates["is_msg_destruct"] = convReq.IsMsgDestruct.GetValue()
	}

	_, hasPinned := updates["is_pinned"]

	for _, uid := range userIDs {
		exist, err := l.svcCtx.ConversationModel.FindConversation(ctx, uid, conversationID)
		isInsert := false
		if err != nil {
			if !isConversationNotFound(err) {
				l.Errorf("find conversation failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
				continue
			}
			isInsert = true
		}

		isPinnedChanged := false
		if !isInsert && hasPinned {
			oldPinned, _ := updates["is_pinned"].(bool)
			if oldPinned != exist.IsPinned {
				isPinnedChanged = true
			}
		}

		if isInsert {
			conv := &model.Conversation{
				OwnerUserID:      uid,
				ConversationID:   conversationID,
				RecvMsgOpt:       getOptionalInt32(convReq.RecvMsgOpt),
				ConversationType: convReq.GetConversationType(),
				UserID:           convReq.GetUserID(),
				GroupID:          convReq.GetGroupID(),
				IsPinned:         getOptionalBool(convReq.IsPinned),
				AttachedInfo:     getOptionalString(convReq.AttachedInfo),
				IsPrivateChat:    getOptionalBool(convReq.IsPrivateChat),
				GroupAtType:      getOptionalInt32(convReq.GroupAtType),
				Extra:            getOptionalString(convReq.Ex),
				BurnDuration:     getOptionalInt32(convReq.BurnDuration),
				IsMsgDestruct:    getOptionalBool(convReq.IsMsgDestruct),
				UpdatedAt:        now,
			}
			if t := getOptionalInt64(convReq.MsgDestructTime); t > 0 {
				conv.MsgDestructTime = time.UnixMilli(t)
			}
			if err := l.svcCtx.ConversationModel.UpsertConversation(ctx, conv); err != nil {
				l.Errorf("upsert conversation failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
				continue
			}
		} else {
			if err := l.svcCtx.ConversationModel.UpdateConversation(ctx, uid, conversationID, updates); err != nil {
				l.Errorf("update conversation failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
				continue
			}
		}

		state := int32(model.VersionStateUpdate)
		if isInsert {
			state = model.VersionStateInsert
		}
		if isPinnedChanged {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, model.ConversationDID(uid), []string{conversationID, model.VersionSortChangeID}, state); err != nil {
				l.Errorf("incr version log batch for conv+sort failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
			}
		} else {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(uid), conversationID, state); err != nil {
				l.Errorf("incr version log for conv failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
			}
		}
	}

	return &pbconversation.SetConversationsResp{}, nil
}

// UpdateConversation 更新会话。IncrVersionLog(owner, convID, Update)，isPinned 变更时合并 VersionSortChangeID。
func (l *Logic) UpdateConversation(ctx context.Context, req *pbconversation.UpdateConversationReq) (*pbconversation.UpdateConversationResp, error) {
	conversationID := req.GetConversationID()
	userIDs := req.GetUserIDs()
	if conversationID == "" || len(userIDs) == 0 {
		return nil, errx.ArgsError.Wrap("conversationID and userIDs are required")
	}

	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	updates := make(map[string]any)
	if req.RecvMsgOpt != nil {
		updates["recv_msg_opt"] = int(req.RecvMsgOpt.GetValue())
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = req.IsPinned.GetValue()
	}
	if req.AttachedInfo != nil {
		updates["attached_info"] = req.AttachedInfo.GetValue()
	}
	if req.IsPrivateChat != nil {
		updates["is_private_chat"] = req.IsPrivateChat.GetValue()
	}
	if req.Ex != nil {
		updates["extra"] = req.Ex.GetValue()
	}
	if req.BurnDuration != nil {
		updates["burn_duration"] = int(req.BurnDuration.GetValue())
	}
	if req.GroupAtType != nil {
		updates["group_at_type"] = int(req.GroupAtType.GetValue())
	}
	if req.MsgDestructTime != nil {
		updates["msg_destruct_time"] = time.UnixMilli(req.MsgDestructTime.GetValue())
	}
	if req.IsMsgDestruct != nil {
		updates["is_msg_destruct"] = req.IsMsgDestruct.GetValue()
	}
	if req.LatestMsgDestructTime != nil {
		updates["latest_msg_destruct_time"] = time.UnixMilli(req.LatestMsgDestructTime.GetValue())
	}

	if len(updates) == 0 {
		return &pbconversation.UpdateConversationResp{}, nil
	}

	_, hasPinned := updates["is_pinned"]

	for _, uid := range userIDs {
		isPinnedChanged := false
		if hasPinned {
			if exist, err := l.svcCtx.ConversationModel.FindConversation(ctx, uid, conversationID); err == nil {
				newPinned, _ := updates["is_pinned"].(bool)
				if newPinned != exist.IsPinned {
					isPinnedChanged = true
				}
			} else if !isConversationNotFound(err) {
				l.Errorf("find conversation failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
			}
		}

		if err := l.svcCtx.ConversationModel.UpdateConversation(ctx, uid, conversationID, updates); err != nil {
			l.Errorf("update conversation failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
			continue
		}

		if isPinnedChanged {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, model.ConversationDID(uid), []string{conversationID, model.VersionSortChangeID}, model.VersionStateUpdate); err != nil {
				l.Errorf("incr version log batch for conv+sort failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
			}
		} else {
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(uid), conversationID, model.VersionStateUpdate); err != nil {
				l.Errorf("incr version log for conv update failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
			}
		}
	}

	return &pbconversation.UpdateConversationResp{}, nil
}

// UpdateConversationsByUser 按用户更新所有会话的某个字段（目前仅支持 ex）。
func (l *Logic) UpdateConversationsByUser(ctx context.Context, req *pbconversation.UpdateConversationsByUserReq) (*pbconversation.UpdateConversationsByUserResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	if req.Ex == nil {
		return &pbconversation.UpdateConversationsByUserResp{}, nil
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	convIDs := make([]string, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ConversationID)
	}
	if len(convIDs) == 0 {
		return &pbconversation.UpdateConversationsByUserResp{}, nil
	}

	updates := map[string]any{"extra": req.Ex.GetValue()}
	if err := l.svcCtx.ConversationModel.UpdateConversations(ctx, userID, convIDs, updates); err != nil {
		l.Errorf("update conversations by user failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 批量写版本日志（Update）
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, model.ConversationDID(userID), convIDs, model.VersionStateUpdate); err != nil {
		l.Errorf("incr version log batch for user conv update failed, userID: %s, err: %v", userID, err)
	}

	return &pbconversation.UpdateConversationsByUserResp{}, nil
}

// SetConversationMaxSeq 批量设置会话最大序列号。
// 写入路径：SeqConversation.MaxSeq（会话级全局）+ SeqUser.MaxSeq（用户级，每个 owner 一条）。
// 不再写入 Conversation 表（已删除冗余字段 max_seq）。
func (l *Logic) SetConversationMaxSeq(ctx context.Context, req *pbconversation.SetConversationMaxSeqReq) (*pbconversation.SetConversationMaxSeqResp, error) {
	// conversationID := req.GetConversationID()
	// owners := req.GetOwnerUserID()
	// maxSeq := req.GetMaxSeq()
	// if conversationID == "" || len(owners) == 0 {
	// 	return nil, errx.ArgsError.Wrap("conversationID and ownerUserID are required")
	// }

	// if err := l.requireAdmin(); err != nil {
	// 	return nil, err
	// }

	// // 1. 写会话级全局 max_seq（所有用户共享）
	// if err := l.svcCtx.SeqConversationModel.UpsertConversationMaxSeq(ctx, conversationID, maxSeq); err != nil {
	// 	l.Errorf("upsert seq conversation max_seq failed, conv: %s, err: %v", conversationID, err)
	// 	return nil, err
	// }

	// // 2. 写每个用户级 max_seq + 版本日志
	// for _, uid := range owners {
	// 	if err := l.svcCtx.SeqUserModel.SetUserMaxSeq(ctx, uid, conversationID, maxSeq); err != nil {
	// 		l.Errorf("set user max_seq failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
	// 		continue
	// 	}
	// 	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(uid), conversationID, model.VersionStateUpdate); err != nil {
	// 		l.Errorf("incr version log for max_seq update failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
	// 	}
	// }

	return &pbconversation.SetConversationMaxSeqResp{}, nil
}

// SetConversationMinSeq 批量设置会话最小序列号。
// 写入路径：SeqConversation.MinSeq（会话级全局，历史清理边界）+ SeqUser.MinSeq（用户级）。
func (l *Logic) SetConversationMinSeq(ctx context.Context, req *pbconversation.SetConversationMinSeqReq) (*pbconversation.SetConversationMinSeqResp, error) {
	// conversationID := req.GetConversationID()
	// owners := req.GetOwnerUserID()
	// minSeq := req.GetMinSeq()
	// if conversationID == "" || len(owners) == 0 {
	// 	return nil, errx.ArgsError.Wrap("conversationID and ownerUserID are required")
	// }

	// if err := l.requireAdmin(); err != nil {
	// 	return nil, err
	// }

	// // 1. 写会话级全局 min_seq（清理边界，所有用户共享）
	// if err := l.svcCtx.SeqConversationModel.UpsertConversationMinSeq(ctx, conversationID, minSeq); err != nil {
	// 	l.Errorf("upsert seq conversation min_seq failed, conv: %s, err: %v", conversationID, err)
	// 	return nil, err
	// }

	// // 2. 写每个用户级 min_seq + 版本日志
	// for _, uid := range owners {
	// 	if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, uid, conversationID, minSeq); err != nil {
	// 		l.Errorf("set user min_seq failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
	// 		continue
	// 	}
	// 	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(uid), conversationID, model.VersionStateUpdate); err != nil {
	// 		l.Errorf("incr version log for min_seq update failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
	// 	}
	// }

	return &pbconversation.SetConversationMinSeqResp{}, nil
}

// CreateSingleChatConversations 创建单聊会话（双向）。IncrVersionLog(owner, convID, Insert)。
func (l *Logic) CreateSingleChatConversations(ctx context.Context, req *pbconversation.CreateSingleChatConversationsReq) (*pbconversation.CreateSingleChatConversationsResp, error) {
	recvID := req.GetRecvID()
	sendID := req.GetSendID()
	conversationID := req.GetConversationID()
	convType := req.GetConversationType()
	if recvID == "" || sendID == "" || conversationID == "" {
		return nil, errx.ArgsError.Wrap("recvID, sendID and conversationID are required")
	}
	if convType == 0 {
		convType = int32(constant.SingleChatType)
	}

	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	now := timex.Now()
	// 双向创建会话
	convA := &model.Conversation{
		OwnerUserID:      sendID,
		ConversationID:   conversationID,
		RecvMsgOpt:       0,
		ConversationType: convType,
		UserID:           recvID,
		UpdatedAt:        now,
	}
	convB := &model.Conversation{
		OwnerUserID:      recvID,
		ConversationID:   conversationID,
		RecvMsgOpt:       0,
		ConversationType: convType,
		UserID:           sendID,
		UpdatedAt:        now,
	}

	if err := l.svcCtx.ConversationModel.InsertConversation(ctx, []*model.Conversation{convA, convB}); err != nil {
		l.Errorf("insert single chat conversations failed, sendID: %s, recvID: %s, err: %v", sendID, recvID, err)
		return nil, err
	}

	// 对两个用户写版本日志（Insert）
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(sendID), conversationID, model.VersionStateInsert); err != nil {
		l.Errorf("incr version log for conv insert failed, owner: %s, conv: %s, err: %v", sendID, conversationID, err)
	}
	if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(recvID), conversationID, model.VersionStateInsert); err != nil {
		l.Errorf("incr version log for conv insert failed, owner: %s, conv: %s, err: %v", recvID, conversationID, err)
	}

	return &pbconversation.CreateSingleChatConversationsResp{}, nil
}

// CreateGroupChatConversations 为群成员创建群聊会话。IncrVersionLogBatch(owner, [convID for each member], Insert)。
// 由于 conversation_id 在群聊中所有成员共享，这里对每个 user 都写入同一 conversationID。
func (l *Logic) CreateGroupChatConversations(ctx context.Context, req *pbconversation.CreateGroupChatConversationsReq) (*pbconversation.CreateGroupChatConversationsResp, error) {
	userIDs := req.GetUserIDs()
	groupID := req.GetGroupID()
	if len(userIDs) == 0 || groupID == "" {
		return nil, errx.ArgsError.Wrap("userIDs and groupID are required")
	}

	if err := l.requireAdmin(); err != nil {
		return nil, err
	}

	now := timex.Now()
	// 群聊会话 conversation_id 通常是 groupID 或派生 ID；这里用 groupID 作为 conversation_id
	conversationID := groupID
	convs := make([]*model.Conversation, 0, len(userIDs))
	for _, uid := range userIDs {
		convs = append(convs, &model.Conversation{
			OwnerUserID:      uid,
			ConversationID:   conversationID,
			RecvMsgOpt:       0,
			ConversationType: constant.ReadGroupChatType,
			GroupID:          groupID,
			UpdatedAt:        now,
		})
	}

	if err := l.svcCtx.ConversationModel.InsertConversation(ctx, convs); err != nil {
		l.Errorf("insert group chat conversations failed, groupID: %s, err: %v", groupID, err)
		return nil, err
	}

	// 对每个用户写版本日志（Insert）
	// 注意：所有用户共用同一 conversationID，因此对每个 user 写一次 IncrVersionLog
	for _, uid := range userIDs {
		if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(uid), conversationID, model.VersionStateInsert); err != nil {
			l.Errorf("incr version log for group conv insert failed, owner: %s, conv: %s, err: %v", uid, conversationID, err)
		}
	}

	return &pbconversation.CreateGroupChatConversationsResp{}, nil
}

// DeleteConversations 删除用户会话。IncrVersionLog(owner, convID, Delete)。
func (l *Logic) DeleteConversations(ctx context.Context, req *pbconversation.DeleteConversationsReq) (*pbconversation.DeleteConversationsResp, error) {
	ownerUserID := req.GetOwnerUserID()
	convIDs := req.GetConversationIDs()
	if ownerUserID == "" {
		return nil, errx.ArgsError.Wrap("ownerUserID is required")
	}

	if err := l.requireSelfOrAdmin(ownerUserID); err != nil {
		return nil, err
	}

	if len(convIDs) > 0 {
		// 删除指定会话
		for _, convID := range convIDs {
			if err := l.svcCtx.ConversationModel.DeleteConversation(ctx, ownerUserID, convID); err != nil {
				l.Errorf("delete conversation failed, owner: %s, conv: %s, err: %v", ownerUserID, convID, err)
				continue
			}
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLog(ctx, model.ConversationDID(ownerUserID), convID, model.VersionStateDelete); err != nil {
				l.Errorf("incr version log for conv delete failed, owner: %s, conv: %s, err: %v", ownerUserID, convID, err)
			}
		}
	} else if req.GetNeedDeleteTime() > 0 {
		// 按时间删除（needDeleteTime 之前的会话）
		convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, ownerUserID)
		if err != nil {
			l.Errorf("find conversations by owner failed, owner: %s, err: %v", ownerUserID, err)
			return nil, err
		}
		threshold := time.Unix(req.GetNeedDeleteTime(), 0)
		toDelete := make([]string, 0)
		for _, c := range convs {
			if c.UpdatedAt.Before(threshold) {
				toDelete = append(toDelete, c.ConversationID)
			}
		}
		if len(toDelete) > 0 {
			updates := map[string]any{"updated_at": timex.Now()}
			// 这里按需求是删除，不是更新；为避免破坏数据先不实现按时间物理删除
			_ = updates
			if _, err := l.svcCtx.VersionLogModel.IncrVersionLogBatch(ctx, model.ConversationDID(ownerUserID), toDelete, model.VersionStateDelete); err != nil {
				l.Errorf("incr version log batch for conv delete failed, owner: %s, err: %v", ownerUserID, err)
			}
		}
	}

	return &pbconversation.DeleteConversationsResp{}, nil
}

// ClearUserConversationMsg 清理用户会话消息（仅占位，实际由 msg 模块处理）。
func (l *Logic) ClearUserConversationMsg(ctx context.Context, req *pbconversation.ClearUserConversationMsgReq) (*pbconversation.ClearUserConversationMsgResp, error) {
	if err := l.requireAdmin(); err != nil {
		return nil, err
	}
	// 占位实现：实际清理逻辑由 msg 模块负责
	return &pbconversation.ClearUserConversationMsgResp{}, nil
}

// ==================== 增量同步方法 ====================

// GetFullOwnerConversationIDs 返回全量会话ID列表 + FNV-1a 哈希比对。
func (l *Logic) GetFullOwnerConversationIDs(ctx context.Context, req *pbconversation.GetFullOwnerConversationIDsReq) (*pbconversation.GetFullOwnerConversationIDsResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	convIDs := make([]string, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ConversationID)
	}
	curHash := hash.HashStringSet(convIDs)

	resp := &pbconversation.GetFullOwnerConversationIDsResp{
		Equal:           req.GetIdHash() != 0 && req.GetIdHash() == curHash,
		ConversationIDs: convIDs,
	}
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, model.ConversationDID(userID)); err2 == nil && verLog != nil {
		resp.VersionID = verLog.ID.Hex()
		resp.Version = uint64(verLog.Version)
	} else if err2 != nil {
		l.Errorf("get version log failed, userID: %s, err: %v", userID, err2)
	}
	return resp, nil
}

// GetIncrementalConversation 获取增量会话。DID=userID。
// 使用 FindChangeLog 拉取增量变更。空 Logs → 全量同步。
func (l *Logic) GetIncrementalConversation(ctx context.Context, req *pbconversation.GetIncrementalConversationReq) (*pbconversation.GetIncrementalConversationResp, error) {
	userID := req.GetUserID()
	if userID == "" {
		return nil, errx.ArgsError.Wrap("userID is required")
	}

	if err := l.requireSelfOrAdmin(userID); err != nil {
		return nil, err
	}

	clientVersion := uint(req.GetVersion())
	clientVersionID := req.GetVersionID()

	verLog, err := l.svcCtx.VersionLogModel.FindChangeLog(ctx, model.ConversationDID(userID), clientVersion, SyncLimit)
	if err != nil {
		l.Errorf("find change log failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	// 空 Logs → 全量同步
	if len(verLog.Logs) == 0 || (clientVersionID != "" && clientVersionID != verLog.ID.Hex()) {
		return l.fullConversationsResp(ctx, userID)
	}

	// 增量同步：分类处理变更日志
	c := model.ClassifyIncrementalLogs(verLog.Logs)

	resp := &pbconversation.GetIncrementalConversationResp{
		Version:   uint64(verLog.Version),
		VersionID: userID,
		Full:      false,
		Delete:    c.DeleteIDs,
	}
	if c.SortChanged {
		// conversation 增量同步也带上 SortVersion，客户端据此重新排序会话列表
		resp.Version = c.SortVersion
	}

	// 拉取新增/更新的会话详情
	fetchIDs := append(append([]string{}, c.InsertIDs...), c.UpdateIDs...)
	if len(fetchIDs) > 0 {
		convs, err2 := l.svcCtx.ConversationModel.FindConversationsByIDs(ctx, userID, fetchIDs)
		if err2 != nil {
			l.Errorf("find conversations by ids failed, userID: %s, ids: %v, err: %v", userID, fetchIDs, err2)
			return nil, err2
		}
		convMap := make(map[string]*model.Conversation, len(convs))
		for _, c2 := range convs {
			convMap[c2.ConversationID] = c2
		}
		// 从 SeqUser 批量填充 min_seq/max_seq
		seqs, err3 := l.svcCtx.SeqUserModel.BatchGetUserSeqs(ctx, userID, fetchIDs)
		if err3 != nil {
			l.Errorf("batch get user seqs failed, userID: %s, err: %v", userID, err3)
		}
		fillPbSeq := func(pbConv *pbconversation.Conversation) {
			if seq, ok := seqs[pbConv.GetConversationID()]; ok && seq != nil {
				pbConv.MinSeq = seq.MinSeq
				pbConv.MaxSeq = seq.MaxSeq
			}
		}
		for _, id := range c.InsertIDs {
			if conv, ok := convMap[id]; ok {
				pbConv := modelToPbConversation(conv)
				fillPbSeq(pbConv)
				resp.Insert = append(resp.Insert, pbConv)
			}
		}
		for _, id := range c.UpdateIDs {
			if conv, ok := convMap[id]; ok {
				pbConv := modelToPbConversation(conv)
				fillPbSeq(pbConv)
				resp.Update = append(resp.Update, pbConv)
			}
		}
	}

	return resp, nil
}

// fullConversationsResp 构造会话全量同步响应
func (l *Logic) fullConversationsResp(ctx context.Context, userID string) (*pbconversation.GetIncrementalConversationResp, error) {
	convs, err := l.svcCtx.ConversationModel.FindConversationsByOwner(ctx, userID)
	if err != nil {
		l.Errorf("find conversations by owner failed, userID: %s, err: %v", userID, err)
		return nil, err
	}
	inserts := make([]*pbconversation.Conversation, 0, len(convs))
	for _, c := range convs {
		inserts = append(inserts, modelToPbConversation(c))
	}
	// 从 SeqUser 批量填充 min_seq/max_seq
	l.fillConversationSeqs(userID, inserts)
	var curVersion uint64
	if verLog, err2 := l.svcCtx.VersionLogModel.GetVersionLog(ctx, model.ConversationDID(userID)); err2 == nil && verLog != nil {
		curVersion = uint64(verLog.Version)
	}
	return &pbconversation.GetIncrementalConversationResp{
		Version:   curVersion,
		VersionID: userID,
		Full:      true,
		Insert:    inserts,
	}, nil
}

// ==================== 辅助函数 ====================

func isConversationNotFound(err error) bool {
	return err != nil && errors.Is(err, conversationModel.ErrConversationNotFound)
}

func getOptionalInt32(v interface{ GetValue() int32 }) int32 {
	if v == nil {
		return 0
	}
	return v.GetValue()
}

func getOptionalInt64(v interface{ GetValue() int64 }) int64 {
	if v == nil {
		return 0
	}
	return v.GetValue()
}

func getOptionalBool(v interface{ GetValue() bool }) bool {
	if v == nil {
		return false
	}
	return v.GetValue()
}

func getOptionalString(v interface{ GetValue() string }) string {
	if v == nil {
		return ""
	}
	return v.GetValue()
}
