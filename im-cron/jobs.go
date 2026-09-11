package imcron

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/PaperMan11/goim/pkg/mcontext"
	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	deleteCount = 10000
	deleteLimit = 50
)

func (s *CronServer) DeleteMessage() {
	// 删除过期的消息
	now := timex.Now()
	delTime := now.Add(-time.Hour * 24 * time.Duration(s.cfg.CronTask.MsgRetentionDays))
	logx.Infof("DeleteMessage start, delTime: %v, deleteCount: %d, deleteLimit: %d", delTime, deleteCount, deleteLimit)

	count := 0
	operationID := fmt.Sprintf("cron_delete_message_%d_%d", os.Getpid(), now.UnixMilli())
	for i := range deleteCount {
		ctx := mcontext.SetOperationIDInContext(context.Background(), fmt.Sprintf("%s_%d", operationID, i))
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
	operationID := fmt.Sprintf("cron_clear_user_message_%d_%d", os.Getpid(), now.UnixMilli())
	for i := range deleteCount {
		ctx := mcontext.SetOperationIDInContext(context.Background(), fmt.Sprintf("%s_%d", operationID, i))
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
