package logic

import (
	"context"

	"github.com/PaperMan11/goim/im-rpc/conversation/internal/svc"
	"github.com/PaperMan11/goim/pkg/apiresp/errx"
	"github.com/PaperMan11/goim/pkg/mcontext"
	pbreconversation "github.com/PaperMan11/goim/pkg/protocol/conversation"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// SyncLimit 增量同步时返回的最大变更条数，超过则要求客户端走全量同步。
	SyncLimit = 200
)

// Logic 封装 conversation rpc 业务逻辑，所有 RPC 方法都通过 NewLogic 创建实例调用。
type Logic struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logx.Logger
}

func NewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Logic {
	return &Logic{
		svcCtx: svcCtx,
		ctx:    ctx,
		Logger: logx.WithContext(ctx),
	}
}

// requireSelfOrAdmin 校验操作者是否为本人或管理员
func (l *Logic) requireSelfOrAdmin(targetUserID string) error {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	ok, err := l.svcCtx.AuthVerifier.CheckAccess(l.ctx, targetUserID)
	if err != nil {
		l.Errorf("check access failed, opUserID=%s targetUserID=%s err=%v", opUserID, targetUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("access denied, opUserID=%s targetUserID=%s", opUserID, targetUserID)
		return errx.NoPermissionError
	}
	return nil
}

// requireAdmin 校验操作者是否为管理员
func (l *Logic) requireAdmin() error {
	opUserID := mcontext.GetOpUserIDFromContext(l.ctx)
	ok, err := l.svcCtx.AuthVerifier.IsIMAdmin(l.ctx, opUserID)
	if err != nil {
		l.Errorf("check admin failed, opUserID=%s err=%v", opUserID, err)
		return errx.InternalError.WrapWithError(err)
	}
	if !ok {
		l.Errorf("not admin, opUserID=%s", opUserID)
		return errx.NoPermissionError
	}
	return nil
}

// modelToPbConversation 将 model.Conversation 转换为 pb Conversation。
// 注意：model.Conversation 不再含 min_seq/max_seq/unread_count 字段，
// 这些 seq 信息由 fillConversationSeqs 从 SeqUser 表读取后填充。
func modelToPbConversation(c *model.Conversation) *pbreconversation.Conversation {
	if c == nil {
		return nil
	}
	return &pbreconversation.Conversation{
		OwnerUserID:           c.OwnerUserID,
		ConversationID:        c.ConversationID,
		RecvMsgOpt:            int32(c.RecvMsgOpt),
		ConversationType:      int32(c.ConversationType),
		UserID:                c.UserID,
		GroupID:               c.GroupID,
		IsPinned:              c.IsPinned,
		AttachedInfo:          c.AttachedInfo,
		IsPrivateChat:         c.IsPrivateChat,
		GroupAtType:           int32(c.GroupAtType),
		Ex:                    c.Extra,
		BurnDuration:          int32(c.BurnDuration),
		MsgDestructTime:       c.MsgDestructTime.UnixMilli(),
		LatestMsgDestructTime: c.LatestMsgDestructTime.UnixMilli(),
		IsMsgDestruct:         c.IsMsgDestruct,
	}
}

// fillConversationSeqs 批量从 SeqUser 表读取用户在各会话中的 min/max/read seq，
// 填充到 pb Conversation 列表。未读数 = max_seq - read_seq。
func (l *Logic) fillConversationSeqs(ownerUserID string, pbConvs []*pbreconversation.Conversation) {
	if len(pbConvs) == 0 {
		return
	}
	convIDs := make([]string, 0, len(pbConvs))
	for _, c := range pbConvs {
		convIDs = append(convIDs, c.GetConversationID())
	}
	seqs, err := l.svcCtx.SeqUserModel.BatchGetUserSeqs(l.ctx, ownerUserID, convIDs)
	if err != nil {
		l.Errorf("batch get user seqs failed, owner: %s, err: %v", ownerUserID, err)
		return
	}
	for _, c := range pbConvs {
		if seq, ok := seqs[c.GetConversationID()]; ok && seq != nil {
			c.MinSeq = seq.MinSeq
			c.MaxSeq = seq.MaxSeq
		}
	}
}

// getUserUnreadCount 从 SeqUser 表读取用户在某会话的未读数（max_seq - read_seq）。
func (l *Logic) getUserUnreadCount(ownerUserID, conversationID string) int64 {
	seq, err := l.svcCtx.SeqUserModel.GetUserSeq(l.ctx, ownerUserID, conversationID)
	if err != nil || seq == nil {
		return 0
	}
	unread := seq.MaxSeq - seq.ReadSeq
	if unread < 0 {
		return 0
	}
	return unread
}

// batchGetUserUnreadCounts 批量获取用户在多个会话的未读数。
func (l *Logic) batchGetUserUnreadCounts(ownerUserID string, convIDs []string) map[string]int64 {
	result := make(map[string]int64, len(convIDs))
	if len(convIDs) == 0 {
		return result
	}
	seqs, err := l.svcCtx.SeqUserModel.BatchGetUserSeqs(l.ctx, ownerUserID, convIDs)
	if err != nil {
		l.Errorf("batch get user seqs for unread failed, owner: %s, err: %v", ownerUserID, err)
		return result
	}
	for convID, seq := range seqs {
		if seq == nil {
			continue
		}
		unread := seq.MaxSeq - seq.ReadSeq
		if unread < 0 {
			unread = 0
		}
		result[convID] = unread
	}
	return result
}

// modelLatestToPbMsgInfo 将 model.ConversationLatestMsg 转换为 pb MsgInfo
func modelLatestToPbMsgInfo(m *model.ConversationLatestMsg) *pbreconversation.MsgInfo {
	if m == nil {
		return nil
	}
	return &pbreconversation.MsgInfo{
		ServerMsgID:       m.ServerMsgID,
		ClientMsgID:       m.ClientMsgID,
		SessionType:       int32(m.SessionType),
		SendID:            m.SendID,
		RecvID:            m.RecvID,
		SenderName:        m.SenderName,
		FaceURL:           m.FaceURL,
		GroupID:           m.GroupID,
		GroupName:         m.GroupName,
		GroupFaceURL:      m.GroupFaceURL,
		GroupType:         int32(m.GroupType),
		GroupMemberCount:  uint32(m.GroupMemberCount),
		LatestMsgRecvTime: m.LatestMsgRecvTime.UnixMilli(),
		MsgFrom:           int32(m.MsgFrom),
		ContentType:       int32(m.ContentType),
		Content:           m.Content,
		Ex:                m.Extra,
	}
}
