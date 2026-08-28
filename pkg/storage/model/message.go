package model

import (
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/utils/convert"
)

const (
	SingleGocMsgNum     = 100  // 每个文档存储的最大消息数
	SingleGocMsgNum5000 = 5000 // 大桶模式（批量场景）
	OldestList          = 0
	NewestList          = -1
)

// MsgDocModel 消息文档模型（分桶设计）。
// 每个文档存储一个会话中最多 SingleGocMsgNum 条消息，
// DocID 格式为 "conversationID:seqSuffix"，seqSuffix = (seq-1)/SingleGocMsgNum。
// 分桶设计避免单集合文档数膨胀，历史消息清理按文档删除，查询按文档批量拉取。
type MsgDocModel struct {
	DocID string          `bson:"doc_id"` // 文档ID：conversationID:seqSuffix
	Msgs  []*MsgInfoModel `bson:"msgs"`   // 消息槽位数组，长度 <= SingleGocMsgNum
}

func (*MsgDocModel) CollectionName() string {
	return CollectionMessage
}

// IsFull 判断文档是否已满（最后一个槽位已被占用）
func (m *MsgDocModel) IsFull() bool {
	return len(m.Msgs) > 0 && m.Msgs[len(m.Msgs)-1] != nil && m.Msgs[len(m.Msgs)-1].Msg != nil
}

// GetDocIndex 根据 seq 计算文档索引
func (*MsgDocModel) GetDocIndex(seq int64) int64 {
	return (seq - 1) / SingleGocMsgNum
}

// GetDocID 根据 conversationID 和 seq 生成文档ID
func (m *MsgDocModel) GetDocID(conversationID string, seq int64) string {
	seqSuffix := (seq - 1) / SingleGocMsgNum
	return indexGen(conversationID, seqSuffix)
}

// GetMsgIndex 根据 seq 计算在文档数组中的索引
func (*MsgDocModel) GetMsgIndex(seq int64) int64 {
	return (seq - 1) % SingleGocMsgNum
}

// GetMinSeq 根据文档索引计算该文档的最小 seq
func (*MsgDocModel) GetMinSeq(index int) int64 {
	return int64(index*SingleGocMsgNum) + 1
}

func indexGen(conversationID string, seqSuffix int64) string {
	return conversationID + ":" + convert.ToString(seqSuffix)
}

// MsgInfoModel 消息槽位模型，对应文档中 msgs 数组的一个元素。
// Msg 为 nil 表示空槽位（稀疏存储），支持 seq 跳跃场景。
type MsgInfoModel struct {
	Msg     *MsgDataModel `bson:"msg"`      // 消息数据，nil 表示空槽位
	Revoke  *RevokeModel  `bson:"revoke"`   // 撤回信息，nil 表示未撤回（独立于 Msg，不修改原始数据）
	DelList []string      `bson:"del_list"` // 逻辑删除该消息的用户ID列表（群聊场景各用户独立删除）
	IsRead  bool          `bson:"is_read"`  // 文档级已读标记
}

// RevokeModel 撤回信息，独立存储在 MsgInfoModel.Revoke，不修改原始 MsgDataModel
type RevokeModel struct {
	Role     int32  `bson:"role"`     // 撤回者角色
	UserID   string `bson:"user_id"`  // 撤回者ID
	Nickname string `bson:"nickname"` // 撤回者昵称
	Time     int64  `bson:"time"`     // 撤回时间（毫秒时间戳）
}

// OfflinePushModel 离线推送信息，用于 APNS/FCM 等推送服务
type OfflinePushModel struct {
	Title         string `bson:"title"`           // 推送标题
	Desc          string `bson:"desc"`            // 推送描述
	Ex            string `bson:"ex"`              // 扩展字段（JSON格式）
	IOSPushSound  string `bson:"ios_push_sound"`  // iOS推送声音
	IOSBadgeCount bool   `bson:"ios_badge_count"` // iOS角标计数
}

// MsgDataModel 消息数据模型，存储消息的完整内容和元信息。
// 注意：ConversationID 不在此结构中，由 MsgDocModel.DocID 承载。
type MsgDataModel struct {
	SendID           string            `bson:"send_id"`            // 发送者ID
	RecvID           string            `bson:"recv_id"`            // 接收者ID
	GroupID          string            `bson:"group_id"`           // 群组ID（群消息时使用）
	ClientMsgID      string            `bson:"client_msg_id"`      // 客户端消息ID
	ServerMsgID      string            `bson:"server_msg_id"`      // 服务端消息ID
	SenderPlatformID int32             `bson:"sender_platform_id"` // 发送者平台ID
	SenderNickname   string            `bson:"sender_nickname"`    // 发送者昵称
	SenderFaceURL    string            `bson:"sender_face_url"`    // 发送者头像URL
	SessionType      int32             `bson:"session_type"`       // 会话类型（1-单聊，2-群聊）
	MsgFrom          int32             `bson:"msg_from"`           // 消息来源（100-用户消息，200-系统消息）
	ContentType      int32             `bson:"content_type"`       // 消息内容类型
	Content          string            `bson:"content"`            // 消息内容
	Seq              int64             `bson:"seq"`                // 消息序列号
	SendTime         int64             `bson:"send_time"`          // 发送时间（毫秒时间戳）
	CreateTime       int64             `bson:"create_time"`        // 创建时间（毫秒时间戳）
	Status           int32             `bson:"status"`             // 消息状态
	IsRead           bool              `bson:"is_read"`            // 是否已读
	Options          map[string]bool   `bson:"options"`            // 消息选项
	OfflinePush      *OfflinePushModel `bson:"offline_push"`       // 离线推送信息
	AtUserIDList     []string          `bson:"at_user_id_list"`    // @用户ID列表
	AttachedInfo     string            `bson:"attached_info"`      // 附加信息
	Ex               string            `bson:"ex"`                 // 扩展字段（JSON格式）
}

// GenExceptionMessageBySeqs 生成空壳消息，用于客户端拉取的 seq 在 DB 中不存在时的容错返回
func GenExceptionMessageBySeqs(seqs []int64) (exceptionMsg []*sdkws.MsgData) {
	for _, v := range seqs {
		msgModel := new(sdkws.MsgData)
		msgModel.Seq = v
		exceptionMsg = append(exceptionMsg, msgModel)
	}
	return exceptionMsg
}

type SearchMessageReq struct {
	SendID      string
	RecvID      string
	SessionType int32
	ContentType int32
	SendTime    int64
	Pagination  Pagination
}
