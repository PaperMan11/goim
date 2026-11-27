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
	// 僵尸在线状态清扫：单轮拉取行数，以及一次任务内最多轮数（避免单次执行过久）。
	staleStatusSweepBatch  = 500
	staleStatusSweepRounds = 10
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
//
// 与 MarkStaleUsersOffline 的分工：
//   - 本任务只修「缓存索引」：仅 SRem 集合成员，不写 DB 状态、不发 1304；
//   - 本任务负责清扫任务覆盖不到的情况——集合里存在成员、但对应状态行已不存在
//     （状态行 10 分钟租约到期被 TTL 删除、手工清理、DB 与 Redis 不一致），
//     此时只有这里能通过 getUserStatus 发现并移除。
//   - 进程异常退出这类"状态行还在"的场景由 MarkStaleUsersOffline 负责，
//     它置离线的顺带效果也会移出集合，因此两者在同一场景下会有交集，但互不可替代。
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

// MarkStaleUsersOffline 清扫僵尸在线状态：把超过 ZombieThresholdMs 未心跳的在线快照行置为离线。
// 置离线会同步刷新 Redis、清理全局在线集合，并向订阅者补发 1304 通知，
// 覆盖进程异常退出（kill -9 / OOM）或网络分区导致 onConnRemove 未执行的情况。
//
// 与 ReconcileOnlineUsers 的分工：
//   - 本任务负责「业务状态修复」：写回 DB 离线状态、清理 Redis、补发 1304；
//   - 只有本任务会向订阅者发状态变更通知，对账任务不会，因此不会重复通知；
//   - 本任务只扫 status=1 的状态行，状态行已被 TTL 删除、仅剩集合成员的场景由
//     ReconcileOnlineUsers 兜底。
func (s *CronServer) MarkStaleUsersOffline() {
	now := timex.Now()
	cutoff := now.Add(-time.Duration(usermodel.ZombieThresholdMs) * time.Millisecond)
	total := 0
	operationID := mcontext.NewOperationID("cron_mark_stale_users_offline")
	for round := 0; round < staleStatusSweepRounds; round++ {
		ctx := mcontext.SetOperationIDInContext(context.Background(), fmt.Sprintf("%s:%d", operationID, round))
		resp, err := s.userService.GetStaleOnlineStatuses(ctx, &pbuser.GetStaleOnlineStatusesReq{
			LastSeenBefore: cutoff.UnixMilli(),
			Limit:          staleStatusSweepBatch,
		})
		if err != nil {
			logx.Errorf("MarkStaleUsersOffline query stale statuses failed, err: %v", err)
			break
		}
		staleList := resp.GetStatus()
		if len(staleList) == 0 {
			break
		}

		statuses := make([]*pbuser.UserOnlineStatus, 0, len(staleList))
		for _, st := range staleList {
			statuses = append(statuses, &pbuser.UserOnlineStatus{
				UserID:   st.GetUserID(),
				ConnID:   st.GetConnID(),
				DeviceID: st.GetDeviceID(),
				Offline:  []int32{st.GetPlatformID()},
			})
		}
		if _, err := s.userService.SetUserOnlineStatus(ctx, &pbuser.SetUserOnlineStatusReq{Status: statuses}); err != nil {
			logx.Errorf("MarkStaleUsersOffline mark offline failed, count: %d, err: %v", len(statuses), err)
			break
		}
		total += len(statuses)
		if len(staleList) < staleStatusSweepBatch {
			break
		}
	}

	logx.Infof("MarkStaleUsersOffline end, total: %d, cost: %v", total, timex.Since(now))
}

// CleanExpiredTokens 清理 token 索引 ZSet 中的僵尸数据：
// 历史遗留的 token:user:* key 没有自身 TTL，token 字符串 key 过期后成员会永久残留。
// 通过 SCAN 遍历这些 key，移除 score 已过期的成员，并在集合为空时删除整个 key。
func (s *CronServer) CleanExpiredTokens() {
	now := timex.Now()
	ctx := context.Background()

	scanned, removed, deleted, err := s.tokenStore.CleanExpiredTokensGlobal(ctx)
	if err != nil {
		logx.Errorf("CleanExpiredTokens failed, err: %v", err)
		return
	}
	logx.Infof("CleanExpiredTokens end, keys: %d, removed: %d, deleted: %d, cost: %v",
		scanned, removed, deleted, timex.Since(now))
}
