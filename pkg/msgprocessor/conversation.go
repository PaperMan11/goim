package msgprocessor

import (
	"slices"
	"strings"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"google.golang.org/protobuf/proto"
)

// ========== 会话 ID 前缀约定 ==========
// 单聊/通知会话由参与双方 userID 字典序拼接，避免同一对用户产生两个会话 ID。
// 群聊会话直接用 groupID，读写扩散加区分前缀。
//
// 前缀   场景
// si_    单聊（Single, Individual）
// g_     写扩散群聊（WriteGroup）
// sg_    读扩散群聊（ReadGroup, SingleGroup 取单成员收件箱）
// sn_    通知会话，类型为 NotificationChatType 的系统通知（撤回/删除/入群）
// n_     聊天会话对应的通知视角；Options.IsNotNotification 为 false 时，
//        聊天消息（si_/g_/sg_）会同步写入 n_ 会话供客户端离线补拉

const (
	prefixSingleChat   = "si_"
	prefixWriteGroup   = "g_"
	prefixReadGroup    = "sg_"
	prefixNotification = "sn_"
	prefixNotifyView   = "n_"
)

// conversationTarget 指定"这次要哪种会话视角"
type conversationTarget int

const (
	targetChat conversationTarget = iota
	targetNotify
)

// isSortedSingleIDPrefix 单聊类前缀需要 id1、id2 字典序排列
var isSortedSingleIDPrefix = map[string]bool{
	prefixSingleChat:   true,
	prefixNotification: true,
	prefixNotifyView:   true,
}

func buildConversation(prefix string, ids ...string) string {
	if n := len(ids); n < 1 || n > 2 {
		return ""
	}
	if isSortedSingleIDPrefix[prefix] {
		arr := []string{ids[0], ids[1]}
		slices.Sort(arr)
		return prefix + strings.Join(arr, "_")
	}
	return prefix + ids[0]
}

// prefixBySessionType 直接返回该 SessionType 对应的"聊天视角"前缀。
func prefixBySessionType(sessionType int) string {
	switch sessionType {
	case constant.SingleChatType:
		return prefixSingleChat
	case constant.WriteGroupChatType:
		return prefixWriteGroup
	case constant.ReadGroupChatType:
		return prefixReadGroup
	case constant.NotificationChatType:
		return prefixNotification
	}
	return ""
}

// chatConvByMsg 返回消息的聊天视角会话 ID（忽略 Options）。
func chatConvByMsg(msg *sdkws.MsgData) string {
	prefix := prefixBySessionType(int(msg.SessionType))
	if prefix == "" {
		return ""
	}
	switch int(msg.SessionType) {
	case constant.SingleChatType, constant.NotificationChatType:
		return buildConversation(prefix, msg.SendID, msg.RecvID)
	case constant.WriteGroupChatType, constant.ReadGroupChatType:
		return buildConversation(prefix, msg.GroupID)
	}
	return ""
}

// notifyConvByMsg 返回消息的通知视角会话 ID（忽略 Options）。
// 所有 SessionType 的通知视角都统一落到 n_ 前缀。
func notifyConvByMsg(msg *sdkws.MsgData) string {
	switch int(msg.SessionType) {
	case constant.SingleChatType, constant.NotificationChatType:
		return buildConversation(prefixNotifyView, msg.SendID, msg.RecvID)
	case constant.WriteGroupChatType, constant.ReadGroupChatType:
		return buildConversation(prefixNotifyView, msg.GroupID)
	}
	return ""
}

// buildConversationBySessionType 根据 SessionType 和目标 ids 生成会话 ID。
// 走"聊天视角"——与 GetChatConversationIDByMsg 输出一致，只是不传 msg。
func buildConversationBySessionType(sessionType int, ids ...string) string {
	prefix := prefixBySessionType(sessionType)
	if prefix == "" {
		return ""
	}
	return buildConversation(prefix, ids...)
}

// ========== 对外 API ==========

// GetNotificationConversationIDByMsg 返回消息的通知视角会话 ID：
// 不关心 Options，强制 n_ 前缀。用于撤回/删除等"系统通知必须落在 n_ 独立会话"的场景。
func GetNotificationConversationIDByMsg(msg *sdkws.MsgData) string {
	return notifyConvByMsg(msg)
}

// GetChatConversationIDByMsg 返回消息的聊天视角会话 ID：
// 不关心 Options，强制聊天前缀（si_/g_/sg_/sn_）。
// 用于推送目标定位、消息落库时确定"这条消息属于哪个聊天会话"。
func GetChatConversationIDByMsg(msg *sdkws.MsgData) string {
	return chatConvByMsg(msg)
}

// GetConversationIDByMsg 根据消息 Options 决定会话视角：
// Options.IsNotNotification=false 视为通知视角（走 n_），否则走聊天视角。
// 用于 MsgData.ConversationID 的填充及业务逻辑判断。
func GetConversationIDByMsg(msg *sdkws.MsgData) string {
	if !Options(msg.Options).IsNotNotification() {
		return notifyConvByMsg(msg)
	}
	return chatConvByMsg(msg)
}

// GetConversationIDBySessionType 从 SessionType + ids 生成聊天视角会话 ID。
// 是 GetChatConversationIDByMsg 的"无 msg 版本"，外部需显式传入 ids。
// 单聊需传 (sendID, recvID)，群聊传 (groupID)。
func GetConversationIDBySessionType(sessionType int, ids ...string) string {
	return buildConversationBySessionType(sessionType, ids...)
}

// IsGroupConversationID 判断会话 ID 是否为群聊（g_ / sg_ 前缀）。
func IsGroupConversationID(conversationID string) bool {
	return strings.HasPrefix(conversationID, prefixWriteGroup) ||
		strings.HasPrefix(conversationID, prefixReadGroup)
}

// IsNotification 判断会话 ID 是否为通知视角（n_ 前缀）。
func IsNotification(conversationID string) bool {
	return strings.HasPrefix(conversationID, prefixNotifyView)
}

// IsNotificationByMsg 根据消息 Options 判断"这条消息在当前上下文里是不是通知视角"。
func IsNotificationByMsg(msg *sdkws.MsgData) bool {
	return !Options(msg.Options).IsNotNotification()
}

// GetNotificationConversationID 把聊天视角会话 ID 映射到通知视角：
// si_a_b -> n_a_b，g_x / sg_x / sn_x -> n_x。已是 n_ 的原样返回。
// 非聊天会话返回空。
func GetNotificationConversationID(conversationID string) string {
	switch {
	case strings.HasPrefix(conversationID, prefixSingleChat),
		strings.HasPrefix(conversationID, prefixWriteGroup),
		strings.HasPrefix(conversationID, prefixReadGroup),
		strings.HasPrefix(conversationID, prefixNotification):
		return prefixNotifyView + strings.TrimPrefix(conversationID,
			strings.SplitN(conversationID, "_", 2)[0]+"_")
	case strings.HasPrefix(conversationID, prefixNotifyView):
		return conversationID
	default:
		return ""
	}
}

// GetSelfNotificationConversationID 返回"仅同步给自己"的通知会话 ID（n_<uid>_<uid>）。
// 仅对自己可见的删除等操作以 sendID == recvID 发出，通知不属于任何聊天会话，
// 也无法从用户会话列表反推，同步时必须显式补上，否则离线期间的通知无法补拉。
func GetSelfNotificationConversationID(userID string) string {
	if userID == "" {
		return ""
	}
	return buildConversation(prefixNotifyView, userID, userID)
}

// IsMsgStateNotification 判断是否为消息状态通知（撤回/删除）。
// 这类通知必须独立落库在 n_ 会话，不占用聊天会话 seq。
func IsMsgStateNotification(contentType int32) bool {
	return contentType == constant.MsgRevokeNotification || contentType == constant.DeleteMsgsNotification
}

// GetRoutingConversationID 返回消息落库与推送使用的会话 ID：
// 撤回/删除通知强制走 n_ 通知会话，其余消息走聊天视角。
func GetRoutingConversationID(msg *sdkws.MsgData) string {
	if IsMsgStateNotification(msg.ContentType) {
		return notifyConvByMsg(msg)
	}
	return chatConvByMsg(msg)
}

// ========== 协议辅助 ==========

type MsgBySeq []*sdkws.MsgData

func (s MsgBySeq) Len() int           { return len(s) }
func (s MsgBySeq) Less(i, j int) bool { return s[i].Seq < s[j].Seq }
func (s MsgBySeq) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func Pb2String(pb proto.Message) (string, error) {
	s, err := proto.Marshal(pb)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func String2Pb(s string, pb proto.Message) error {
	return proto.Unmarshal([]byte(s), pb)
}
