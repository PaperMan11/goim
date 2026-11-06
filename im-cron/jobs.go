package imcron

import (
	"context"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/mcontext"
	"github.com/PaperMan11/goim/pkg/protocol/constant"
	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	pbuser "github.com/PaperMan11/goim/pkg/protocol/user"
	usermodel "github.com/PaperMan11/goim/pkg/storage/mongo/user"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	deleteCount          = 10000
	deleteLimit          = 50
	onlineReconcileBatch = 200
)

func (s *CronServer) DeleteMessage() {
	// 删除过期的消息
	now := timex.Now()
	delTime := now.Add(-time.Hour * 24 * time.Duration(s.cfg.CronTask.MsgRetentionDays))
	logx.Infof("DeleteMessage start, delTime: %v, deleteCount: %d, deleteLimit: %d", delTime, deleteCount, deleteLimit)

	count := 0
	operationID := mcontext.NewOperationID("cron_delete_message")
	for i := range deleteCount {
		ctx := mcontext.SetOperationIDInContext(context.Background(), fmt.Sprintf("%s:%d", operationID, i))
		resp, err := s.msgService.DestructMsgs(ctx, &pbmsg.DestructMsgsReq{
			Timestamp: delTime.UnixMilli(),
			Limit:     deleteLimit,
		})
		if err != nil {
			logx.Errorf("DeleteMessage, err: %v", err)
			break
		}
		count += int(resp.GetCount())
		if resp.GetCount() < deleteLimit {
			break
		}
	}
	logx.Infof("DeleteMessage end, count: %d, cost: %v", count, timex.Since(now))
}

func (s *CronServer) ClearUserMessage() {
	// 清除过期的用户消息
	now := timex.Now()
	delTime := now.Add(-time.Hour * 24 * time.Duration(s.cfg.CronTask.ConvRetentionDays))
	logx.Infof("ClearUserMessage start, delTime: %v, deleteCount: %d, deleteLimit: %d", delTime, deleteCount, deleteLimit)

	count := 0
	operationID := mcontext.NewOperationID("cron_clear_user_message")
	for i := range deleteCount {
		ctx := mcontext.SetOperationIDInContext(context.Background(), fmt.Sprintf("%s:%d", operationID, i))
		resp, err := s.convService.ClearUserConversationMsg(ctx, &pbconv.ClearUserConversationMsgReq{
			Timestamp: delTime.UnixMilli(),
			Limit:     deleteLimit,
		})
		if err != nil {
			logx.Errorf("ClearUserMessage, err: %v", err)
			break
		}
		count += int(resp.GetCount())
		if resp.GetCount() < deleteLimit {
			break
		}
	}
	logx.Infof("ClearUserMessage end, count: %d, cost: %v", count, timex.Since(now))
}

// ClearDestructMsg 清理已到期的自毁会话消息。以当前时间为截止点，
// 由 conversation 服务找出 msg_destruct_time <= now 的会话并调用 msg 服务物理删除。
func (s *CronServer) ClearDestructMsg() {
	now := timex.Now()
	logx.Infof("ClearDestructMsg start, timestamp: %v, deleteCount: %d, deleteLimit: %d", now, deleteCount, deleteLimit)

	count := 0
	operationID := mcontext.NewOperationID("cron_destruct_message")
	for i := range deleteCount {
		ctx := mcontext.SetOperationIDInContext(context.Background(), fmt.Sprintf("%s:%d", operationID, i))
		resp, err := s.convService.ClearUserConversationMsg(ctx, &pbconv.ClearUserConversationMsgReq{
			Timestamp: now.UnixMilli(),
			Limit:     deleteLimit,
		})
		if err != nil {
			logx.Errorf("ClearDestructMsg, err: %v", err)
			break
		}
		count += int(resp.GetCount())
		if resp.GetCount() < deleteLimit {
			break
		}
	}
	logx.Infof("ClearDestructMsg end, count: %d, cost: %v", count, timex.Since(now))
}

// ReconcileOnlineUsers 对账全局在线用户集合，清理僵尸成员：
// 从 Redis 集合取出所有成员，通过 user RPC 校验在线状态，离线成员从集合中移除。
func (s *CronServer) ReconcileOnlineUsers() {
	now := timex.Now()
	ctx := context.Background()
	setKey := usermodel.GetUserOnlineSetKey()

	members, err := s.redisClient.SMembers(ctx, setKey).Result()
	if err != nil {
		logx.Errorf("ReconcileOnlineUsers get online set failed, err: %v", err)
		return
	}
	if len(members) == 0 {
		logx.Infof("ReconcileOnlineUsers end, online set is empty, cost: %v", timex.Since(now))
		return
	}

	removed := 0
	for start := 0; start < len(members); start += onlineReconcileBatch {
		end := start + onlineReconcileBatch
		if end > len(members) {
			end = len(members)
		}
		batch := members[start:end]

		resp, err := s.userService.GetUserStatus(ctx, &pbuser.GetUserStatusReq{UserIDs: batch})
		if err != nil {
			logx.Errorf("ReconcileOnlineUsers get user status failed, err: %v", err)
			continue
		}
		online := make(map[string]struct{}, len(resp.GetStatusList()))
		for _, status := range resp.GetStatusList() {
			if status.GetStatus() == constant.Online {
				online[status.GetUserID()] = struct{}{}
			}
		}

		stale := make([]string, 0, len(batch))
		for _, userID := range batch {
			if _, ok := online[userID]; !ok {
				stale = append(stale, userID)
			}
		}
		if len(stale) == 0 {
			continue
		}
		args := make([]interface{}, 0, len(stale))
		for _, userID := range stale {
			args = append(args, userID)
		}
		if err := s.redisClient.SRem(ctx, setKey, args...).Err(); err != nil {
			logx.Errorf("ReconcileOnlineUsers remove stale members failed, err: %v", err)
			continue
		}
		removed += len(stale)
	}

	logx.Infof("ReconcileOnlineUsers end, total: %d, removed: %d, cost: %v", len(members), removed, timex.Since(now))
}
