package mconvert

import (
	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	"github.com/PaperMan11/goim/pkg/storage/model"
)

// ModelToPbConversation 将 model.Conversation 转换为 pb Conversation。
// 注意：model.Conversation 不再含 min_seq/max_seq/unread_count 字段，
// 这些 seq 信息由 fillConversationSeqs 从 SeqUser 表读取后填充。
func ModelToPbConversation(c *model.Conversation) *pbconv.Conversation {
	if c == nil {
		return nil
	}
	return &pbconv.Conversation{
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

// modelLatestToPbMsgInfo 将 model.ConversationLatestMsg 转换为 pb MsgInfo
func ModelLatestToPbMsgInfo(m *model.ConversationLatestMsg) *pbconv.MsgInfo {
	if m == nil {
		return nil
	}
	return &pbconv.MsgInfo{
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
