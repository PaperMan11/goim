package msgprocessor

import (
	"strings"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/gogo/protobuf/proto"
)

func IsGroupConversationID(conversationID string) bool {
	return strings.HasPrefix(conversationID, "g_") || strings.HasPrefix(conversationID, "sg_")
}

func buildSortedConversationID(prefix string, id1, id2 string) string {
	// if id1 < id2 {
	// 	return prefix + id1 + "_" + id2
	// }
	// return prefix + id2 + "_" + id1
	return prefix + id1 + "_" + id2
}

func GetNotificationConversationIDByMsg(msg *sdkws.MsgData) string {
	switch msg.SessionType {
	case constant.SingleChatType:
		return buildSortedConversationID("n_", msg.SendID, msg.RecvID)
	case constant.WriteGroupChatType:
		return "n_" + msg.GroupID
	case constant.ReadGroupChatType:
		return "n_" + msg.GroupID
	case constant.NotificationChatType:
		return buildSortedConversationID("n_", msg.SendID, msg.RecvID)
	}
	return ""
}

func GetChatConversationIDByMsg(msg *sdkws.MsgData) string {
	switch msg.SessionType {
	case constant.SingleChatType:
		return buildSortedConversationID("si_", msg.SendID, msg.RecvID)
	case constant.WriteGroupChatType:
		return "g_" + msg.GroupID
	case constant.ReadGroupChatType:
		return "sg_" + msg.GroupID
	case constant.NotificationChatType:
		return buildSortedConversationID("sn_", msg.SendID, msg.RecvID)
	}
	return ""
}

func GetConversationIDByMsg(msg *sdkws.MsgData) string {
	options := Options(msg.Options)
	switch msg.SessionType {
	case constant.SingleChatType:
		if !options.IsNotNotification() {
			return buildSortedConversationID("n_", msg.SendID, msg.RecvID)
		}
		return buildSortedConversationID("si_", msg.SendID, msg.RecvID)
	case constant.WriteGroupChatType:
		if !options.IsNotNotification() {
			return "n_" + msg.GroupID
		}
		return "g_" + msg.GroupID
	case constant.ReadGroupChatType:
		if !options.IsNotNotification() {
			return "n_" + msg.GroupID
		}
		return "sg_" + msg.GroupID
	case constant.NotificationChatType:
		if !options.IsNotNotification() {
			return buildSortedConversationID("n_", msg.SendID, msg.RecvID)
		}
		return buildSortedConversationID("sn_", msg.SendID, msg.RecvID)
	}
	return ""
}

func GetConversationIDBySessionType(sessionType int, ids ...string) string {
	if len(ids) > 2 || len(ids) < 1 {
		return ""
	}
	switch sessionType {
	case constant.SingleChatType:
		return buildSortedConversationID("si_", ids[0], ids[1])
	case constant.WriteGroupChatType:
		return "g_" + ids[0]
	case constant.ReadGroupChatType:
		return "sg_" + ids[0]
	case constant.NotificationChatType:
		return buildSortedConversationID("sn_", ids[0], ids[1])
	}
	return ""
}

func IsNotification(conversationID string) bool {
	return strings.HasPrefix(conversationID, "n_")
}

func IsNotificationByMsg(msg *sdkws.MsgData) bool {
	return !Options(msg.Options).IsNotNotification()
}

type MsgBySeq []*sdkws.MsgData

func (s MsgBySeq) Len() int {
	return len(s)
}

func (s MsgBySeq) Less(i, j int) bool {
	return s[i].Seq < s[j].Seq
}

func (s MsgBySeq) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

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
