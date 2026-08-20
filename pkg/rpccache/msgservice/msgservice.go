package msgservice

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/localcache"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/rpcclient/msgservice"
	"google.golang.org/grpc"
)

type MsgServiceWrapperCache interface {
	msgservice.MsgService
}

type MsgService struct {
	msgservice.MsgService
	localCache localcache.LocalCache
}

func NewMsgServiceWrapperCache(msgService msgservice.MsgService, cache localcache.LocalCache) MsgServiceWrapperCache {
	return &MsgService{
		MsgService: msgService,
		localCache: cache,
	}
}

// deleteConversationMaxSeqCache 删除指定会话的最大序列号缓存
func (s *MsgService) deleteConversationMaxSeqCache(_ context.Context, conversationIDs []string) {
	if s.localCache == nil {
		return
	}
	keys := make([]string, 0, len(conversationIDs))
	for _, id := range conversationIDs {
		keys = append(keys, GetConversationMaxSeqKey(id))
	}
	s.localCache.PublishDelete(keys)
}

// ==================== 读方法（带缓存） ====================

// GetConversationMaxSeq 获取会话最大序列号（带本地缓存）
func (s *MsgService) GetConversationMaxSeq(ctx context.Context, in *pbmsg.GetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.GetConversationMaxSeqResp, error) {
	if s.localCache == nil {
		return s.MsgService.GetConversationMaxSeq(ctx, in, opts...)
	}
	key := GetConversationMaxSeqKey(in.GetConversationID())
	respI, err := s.localCache.Take(key, func() (any, error) {
		return s.MsgService.GetConversationMaxSeq(ctx, in, opts...)
	})
	if err != nil {
		return nil, err
	}
	if result, ok := respI.(*pbmsg.GetConversationMaxSeqResp); ok {
		return result, nil
	}
	return nil, errors.New("invalid response type")
}

// GetServerTime 获取服务器时间（短 TTL 缓存，避免高频调用）
func (s *MsgService) GetServerTime(ctx context.Context, in *pbmsg.GetServerTimeReq, opts ...grpc.CallOption) (*pbmsg.GetServerTimeResp, error) {
	if s.localCache == nil {
		return s.MsgService.GetServerTime(ctx, in, opts...)
	}
	key := GetServerTimeKey()
	if respI, ok := s.localCache.Get(key); ok {
		if result, ok := respI.(*pbmsg.GetServerTimeResp); ok {
			return result, nil
		}
	}
	resp, err := s.MsgService.GetServerTime(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	s.localCache.SetWithExpire(key, resp, 2*time.Second)
	return resp, nil
}

// ==================== 写方法（删缓存） ====================

// SetUserConversationMaxSeq 设置用户会话最大序列号 → 删会话 max_seq 缓存
func (s *MsgService) SetUserConversationMaxSeq(ctx context.Context, in *pbmsg.SetUserConversationMaxSeqReq, opts ...grpc.CallOption) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	if s.localCache != nil {
		s.deleteConversationMaxSeqCache(ctx, []string{in.GetConversationID()})
	}
	return s.MsgService.SetUserConversationMaxSeq(ctx, in, opts...)
}

// ClearConversationsMsg 清空会话消息 → 删会话 max_seq 缓存
func (s *MsgService) ClearConversationsMsg(ctx context.Context, in *pbmsg.ClearConversationsMsgReq, opts ...grpc.CallOption) (*pbmsg.ClearConversationsMsgResp, error) {
	if s.localCache != nil {
		s.deleteConversationMaxSeqCache(ctx, in.GetConversationIDs())
	}
	return s.MsgService.ClearConversationsMsg(ctx, in, opts...)
}

// DeleteMsgs 按序列号删除消息 → 删会话 max_seq 缓存
func (s *MsgService) DeleteMsgs(ctx context.Context, in *pbmsg.DeleteMsgsReq, opts ...grpc.CallOption) (*pbmsg.DeleteMsgsResp, error) {
	if s.localCache != nil {
		s.deleteConversationMaxSeqCache(ctx, []string{in.GetConversationID()})
	}
	return s.MsgService.DeleteMsgs(ctx, in, opts...)
}

// RevokeMsg 撤回消息 → 删会话 max_seq 缓存
func (s *MsgService) RevokeMsg(ctx context.Context, in *pbmsg.RevokeMsgReq, opts ...grpc.CallOption) (*pbmsg.RevokeMsgResp, error) {
	if s.localCache != nil {
		s.deleteConversationMaxSeqCache(ctx, []string{in.GetConversationID()})
	}
	return s.MsgService.RevokeMsg(ctx, in, opts...)
}
