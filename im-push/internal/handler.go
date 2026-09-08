package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PaperMan11/goim/im-push/internal/offlnepush/options"
	"github.com/PaperMan11/goim/pkg/msgprocessor"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbgroup "github.com/PaperMan11/goim/pkg/protocol/group"
	pbmsggateway "github.com/PaperMan11/goim/pkg/protocol/msggateway"
	pbpush "github.com/PaperMan11/goim/pkg/protocol/push"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	queuex "github.com/PaperMan11/goim/pkg/queue"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/gogo/protobuf/proto"
	"github.com/zeromicro/go-zero/core/logc"
)

const (
	// pushTokenKeyPrefix 客户端通过 SetUserClientConfig 注册 push token 时采用的 key 前缀
	// 完整 key 格式：push_token_<platformID>，例如 push_token_1（iOS/APNs）、push_token_2（Android/FCM/JPush/GeTui）
	pushTokenKeyPrefix = "push_token_"
)

func (p *Pusher) consumePushMsg(ctx context.Context, msg queuex.Message) error {
	var pushMsg pbpush.PushMsgReq
	if err := proto.Unmarshal(msg.Value(), &pushMsg); err != nil {
		logc.Errorf(ctx, "push unmarshal msg err, err: %v, msg: %s", err, msg.Value)
		return err
	}

	if timex.Since(timex.ParseUnixMilli(pushMsg.MsgData.SendTime)) > 10*time.Second {
		logc.Infof(ctx, "push msg is expired, send time: %d, now: %d", pushMsg.MsgData.SendTime, timex.Now())
		return nil
	}

	switch pushMsg.MsgData.SessionType {
	case constant.ReadGroupChatType:
		return p.pushToGroup(ctx, pushMsg.MsgData.GroupID, pushMsg.MsgData)
	default:
		pushUserIDs := make([]string, 0)
		isSenderSync := msgprocessor.Options(pushMsg.MsgData.Options).IsSenderSync()
		if !isSenderSync || pushMsg.MsgData.SendID == pushMsg.MsgData.RecvID {
			pushUserIDs = append(pushUserIDs, pushMsg.MsgData.SendID)
		} else {
			pushUserIDs = append(pushUserIDs, pushMsg.MsgData.SendID, pushMsg.MsgData.RecvID)
		}
		return p.pushToUser(ctx, pushUserIDs, pushMsg.MsgData)
	}
}

func (p *Pusher) pushToGroup(ctx context.Context, groupID string, msg *sdkws.MsgData) error {
	logc.Infof(ctx, "push msg to group chat, groupID: %s, msg: %s", groupID, msg.String())

	// 1. 获取群成员列表
	membersResp, err := p.groupService.GetGroupMemberUserIDs(ctx, &pbgroup.GetGroupMemberUserIDsReq{
		GroupID: groupID,
	})
	if err != nil {
		logc.Errorf(ctx, "get group member userIDs err, err: %v, groupID: %s", err, groupID)
		return err
	}
	userIDs := membersResp.GetUserIDs()
	if len(userIDs) == 0 {
		return nil
	}

	// 2. 过滤掉发送者本人（除非是 senderSync 模式）
	isSenderSync := msgprocessor.Options(msg.Options).IsSenderSync()
	if !isSenderSync {
		filtered := make([]string, 0, len(userIDs))
		for _, uid := range userIDs {
			if uid != msg.SendID {
				filtered = append(filtered, uid)
			}
		}
		userIDs = filtered
	}
	if len(userIDs) == 0 {
		return nil
	}

	return p.pushToUser(ctx, userIDs, msg)
}

func (p *Pusher) pushToUser(ctx context.Context, userIDs []string, msg *sdkws.MsgData) error {
	logc.Infof(ctx, "push msg to user, userIDs: %v, msg: %s", userIDs, msg.String())

	userStatusResp, err := p.userService.GetUserStatus(ctx, &pbuser.GetUserStatusReq{
		UserIDs: userIDs,
	})
	if err != nil {
		logc.Errorf(ctx, "get user status err, err: %v, userIDs: %v", err, userIDs)
		return err
	}

	onlineUserIDs := make([]string, 0)
	offlineUserIDs := make([]string, 0)
	for _, status := range userStatusResp.GetStatusList() {
		if status.Status == constant.Online {
			onlineUserIDs = append(onlineUserIDs, status.UserID)
		} else {
			offlineUserIDs = append(offlineUserIDs, status.UserID)
		}
	}

	if len(onlineUserIDs) > 0 {
		err = p.onlinePush(ctx, onlineUserIDs, msg)
		if err != nil {
			logc.Errorf(ctx, "online push err, err: %v, userIDs: %v", err, onlineUserIDs)
			return err
		}
	}

	if !p.shouldOfflinePush(ctx, msg) {
		return nil
	}

	if len(offlineUserIDs) > 0 {
		err = p.toOfflinePushTopic(ctx, offlineUserIDs, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Pusher) shouldOfflinePush(_ context.Context, msg *sdkws.MsgData) bool {
	offlinePush := msgprocessor.Options(msg.Options).IsOfflinePush()
	if !offlinePush {
		return false
	}

	if msg.ContentType == constant.RoomParticipantsConnectedNotification || msg.ContentType == constant.RoomParticipantsDisconnectedNotification {
		return false
	}
	return true
}

func (p *Pusher) onlinePush(ctx context.Context, userIDs []string, msg *sdkws.MsgData) error {
	resp, err := p.msgGatewayService.SuperGroupOnlineBatchPushOneMsg(ctx, &pbmsggateway.OnlineBatchPushOneMsgReq{
		MsgData:       msg,
		PushToUserIDs: userIDs,
	})
	if err != nil {
		return err
	}

	logc.Debugf(ctx, "online push resp: %+v", resp.GetSinglePushResult())
	return nil
}

// offlinePush 给离线用户推送原生系统通知
// 流程：获取用户的 push token → 根据消息构造 title/content → 调用离线推送 SDK（JPush/GeTui/FCM）
func (p *Pusher) offlinePush(ctx context.Context, userIDs []string, msg *sdkws.MsgData) error {
	if p.offlinePusher == nil {
		logc.Errorf(ctx, "offline pusher is not configured, skip offline push, userIDs: %v", userIDs)
		return nil
	}
	if len(userIDs) == 0 {
		return nil
	}

	// 1. 构造推送标题和内容
	title, content := buildPushTitleAndContent(msg)

	// 2. 批量获取所有离线用户的 device token
	//    客户端通过 SetUserClientConfig 注册，key 格式 push_token_<platformID>
	userTokens := p.getUserPushTokens(ctx, userIDs)
	if len(userTokens) == 0 {
		logc.Infof(ctx, "no push tokens found for offline users, skip. userIDs: %v", userIDs)
		return nil
	}

	// 3. 按 token 分组推送（同一 platform 的 token 归为一组，JPush/GeTui/FCM 均支持批量单 API 调用）
	var allTokens []string
	for _, tokens := range userTokens {
		allTokens = append(allTokens, tokens...)
	}
	if len(allTokens) == 0 {
		return nil
	}

	logc.Infof(ctx, "offline push to %d users with %d tokens, title: %s, msgID: %s",
		len(userIDs), len(allTokens), title, msg.ClientMsgID)

	err := p.offlinePusher.Push(ctx, allTokens, title, content,
		options.WithSignal(msg.ClientMsgID),
		options.WithIOSPushSound(p.cfg.OfflinePush.PushSound),
		options.WithIOSBadgeCount(p.cfg.OfflinePush.BadgeCount),
		options.WithEx(string(msg.Content)),
	)
	if err != nil {
		logc.Errorf(ctx, "offline push error, err: %v, userIDs: %v, tokens: %v", err, userIDs, allTokens)
		return err
	}

	logc.Infof(ctx, "offline push success, tokens count: %d", len(allTokens))
	return nil
}

// getUserPushTokens 批量获取用户的所有平台 device token
// 返回 map[userID][]token（去重后）
func (p *Pusher) getUserPushTokens(ctx context.Context, userIDs []string) map[string][]string {
	result := make(map[string][]string)
	for _, userID := range userIDs {
		resp, err := p.userService.GetUserClientConfig(ctx, &pbuser.GetUserClientConfigReq{
			UserID: userID,
		})
		if err != nil {
			logc.Errorf(ctx, "get user client config err, userID: %s, err: %v", userID, err)
			continue
		}
		for k, v := range resp.GetConfigs() {
			if strings.HasPrefix(k, pushTokenKeyPrefix) && v != "" {
				result[userID] = append(result[userID], v)
			}
		}
	}
	return result
}

// buildPushTitleAndContent 根据消息内容构造系统通知的标题和正文
func buildPushTitleAndContent(msg *sdkws.MsgData) (title, content string) {
	title = msg.SenderNickname
	if title == "" {
		title = "您有新的消息"
	}

	// 群聊场景加 [群聊] 前缀/群组名
	if msg.SessionType == constant.ReadGroupChatType && msg.GroupID != "" {
		title = fmt.Sprintf("[群聊] %s", title)
	}

	// 通知类消息（系统通知、已读回执等）不发离线推送
	if msg.ContentType >= constant.NotificationBegin && msg.ContentType <= constant.NotificationEnd {
		return title, ""
	}

	// 根据 contentType 生成简要文本预览
	switch msg.ContentType {
	case constant.Text:
		content = extractTextContent(msg.Content)
	case constant.Picture:
		// content = constant.ContentType2PushContent[int64(msg.ContentType)]
		content = "[图片]"
	case constant.Voice:
		content = "[语音]"
	case constant.Video:
		content = "[视频]"
	case constant.File:
		content = "[文件]"
	case constant.Location:
		content = "[位置]"
	case constant.Merger:
		content = "[聊天记录]"
	case constant.Card:
		content = "[卡片]"
	case constant.Quote:
		content = "[引用消息]"
	case constant.Revoke:
		content = "[撤回了一条消息]"
	case constant.HasReadReceipt:
		// 已读回执不推送
		return "", ""
	default:
		content = "[新消息]"
	}

	// 截断过长内容
	runes := []rune(content)
	if len(runes) > 50 {
		content = string(runes[:50]) + "..."
	}
	return title, content
}

// extractTextContent 从 Text 消息的 JSON content 中提取文本
// 兼容 sdkws 中 TextElem 结构（msg.content = {"text":"Hello world","atUsers":[]}）
func extractTextContent(content []byte) string {
	if len(content) == 0 {
		return ""
	}
	// 简单提取 "text" 字段值，避免引入 sdkws.TextElem 循环依赖
	str := string(content)
	if idx := strings.Index(str, `"text"`); idx >= 0 {
		// 找到冒号后的字符串值
		rest := str[idx+len(`"text"`):]
		if colon := strings.Index(rest, ":"); colon >= 0 {
			val := rest[colon+1:]
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"`)
			if end := strings.Index(val, `"`); end >= 0 {
				val = val[:end]
			}
			if val != "" {
				return val
			}
		}
	}
	// 兜底：直接返回原始 bytes 的可读部分（过滤掉 JSON 引号和括号）
	return strings.TrimSpace(strings.Trim(string(content), `{}[]"`))
}

func (p *Pusher) consumeOfflinePushMsg(ctx context.Context, msg queuex.Message) error {
	// 离线推送队列复用 PushMsgReq 格式，消息已过正常 push 队列消费链路后
	// 投递到离线队列做异步兜底（正常情况下 offlinePush 已经同步处理）
	// 这里保留消费逻辑，支持"延迟重试"或"离线重放"等场景
	var pushMsg pbpush.PushMsgReq
	if err := proto.Unmarshal(msg.Value(), &pushMsg); err != nil {
		logc.Errorf(ctx, "offline push unmarshal msg err, err: %v, msg: %s", err, msg.Value)
		return err
	}

	if timex.Since(timex.ParseUnixMilli(pushMsg.MsgData.SendTime)) > 24*time.Hour {
		logc.Infof(ctx, "offline push msg is expired, send time: %d, now: %d", pushMsg.MsgData.SendTime, timex.Now())
		return nil
	}

	// 如果 PushMsgReq 中指定了 userIDs，直接对这些用户做离线推送
	if len(pushMsg.UserIDs) > 0 {
		return p.offlinePush(ctx, pushMsg.UserIDs, pushMsg.MsgData)
	}

	// 否则按会话类型路由：群聊取成员列表，单聊取 recvID
	switch pushMsg.MsgData.SessionType {
	case constant.ReadGroupChatType:
		membersResp, err := p.groupService.GetGroupMemberUserIDs(ctx, &pbgroup.GetGroupMemberUserIDsReq{
			GroupID: pushMsg.MsgData.GroupID,
		})
		if err != nil {
			logc.Errorf(ctx, "offline push get group member err, err: %v, groupID: %s", err, pushMsg.MsgData.GroupID)
			return err
		}
		return p.offlinePush(ctx, membersResp.GetUserIDs(), pushMsg.MsgData)
	default:
		userIDs := []string{pushMsg.MsgData.RecvID}
		return p.offlinePush(ctx, userIDs, pushMsg.MsgData)
	}
}

func (p *Pusher) toOfflinePushTopic(ctx context.Context, userIDs []string, msg *sdkws.MsgData) error {
	pushMsg := pbpush.PushMsgReq{
		ConversationID: msgprocessor.GetConversationIDByMsg(msg),
		MsgData:        msg,
		UserIDs:        userIDs,
	}
	pushMsgBytes, err := proto.Marshal(&pushMsg)
	if err != nil {
		logc.Errorf(ctx, "offline push marshal msg err, err: %v, msg: %s", err, msg)
		return err
	}
	err = p.offlinePushProducer.Push(ctx, string(pushMsgBytes))
	if err != nil {
		logc.Errorf(ctx, "offline push push msg err, err: %v, msg: %s", err, msg)
		return err
	}
	return nil
}
