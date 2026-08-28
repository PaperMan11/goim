package conversationservice

import (
	"context"

	"github.com/PaperMan11/goim/pkg/localcache"
	"github.com/PaperMan11/goim/pkg/rpcclient/conversationservice"
	"google.golang.org/grpc"

	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
)

type ConversationServiceWrapperCache interface {
	conversationservice.ConversationService
	GetSingleConversationRecvMsgOpt(ctx context.Context, userID, conversationID string) (int32, error)
}

type ConversationService struct {
	conversationservice.ConversationService
	localCache localcache.LocalCache
}

func NewConversationServiceWrapperCache(conversationService conversationservice.ConversationService, cache localcache.LocalCache) ConversationServiceWrapperCache {
	return &ConversationService{
		ConversationService: conversationService,
		localCache:          cache,
	}
}

func (s *ConversationService) GetConversation(ctx context.Context, in *pbconv.GetConversationReq, opts ...grpc.CallOption) (*pbconv.GetConversationResp, error) {
	if s.localCache == nil {
		return s.ConversationService.GetConversation(ctx, in, opts...)
	}

	key := GetUserConversationKey(in.OwnerUserID, in.ConversationID)
	conversation, err := s.localCache.Take(key, func() (any, error) {
		resp, err := s.ConversationService.GetConversation(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		return resp.Conversation, nil
	})
	if err != nil {
		return nil, err
	}
	return &pbconv.GetConversationResp{
		Conversation: conversation.(*pbconv.Conversation),
	}, nil
}

func (s *ConversationService) SetConversation(ctx context.Context, in *pbconv.SetConversationReq, opts ...grpc.CallOption) (*pbconv.SetConversationResp, error) {
	if s.localCache != nil && in.Conversation != nil {
		userConvKey := GetUserConversationKey(in.Conversation.OwnerUserID, in.Conversation.ConversationID)
		userConvIDsKey := GetUserConversationIDsKey(in.Conversation.OwnerUserID)
		userPinnedIDsKey := GetUserPinnedConversationIDsKey(in.Conversation.OwnerUserID)
		s.localCache.PublishDelete([]string{userConvKey, userConvIDsKey, userPinnedIDsKey})
	}
	return s.ConversationService.SetConversation(ctx, in, opts...)
}

func (s *ConversationService) GetRecvMsgNotNotifyUserIDs(ctx context.Context, in *pbconv.GetRecvMsgNotNotifyUserIDsReq, opts ...grpc.CallOption) (*pbconv.GetRecvMsgNotNotifyUserIDsResp, error) {
	if s.localCache == nil {
		return s.ConversationService.GetRecvMsgNotNotifyUserIDs(ctx, in, opts...)
	}

	key := GetRecvMsgNotNotifyUserIDsKey(in.GroupID)
	userIDs, err := s.localCache.Take(key, func() (any, error) {
		resp, err := s.ConversationService.GetRecvMsgNotNotifyUserIDs(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		return resp.UserIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return &pbconv.GetRecvMsgNotNotifyUserIDsResp{
		UserIDs: userIDs.([]string),
	}, nil
}

func (s *ConversationService) CreateGroupChatConversations(ctx context.Context, in *pbconv.CreateGroupChatConversationsReq, opts ...grpc.CallOption) (*pbconv.CreateGroupChatConversationsResp, error) {
	if s.localCache != nil {
		keys := make([]string, 0)
		for _, userID := range in.UserIDs {
			userConvIDsKey := GetUserConversationIDsKey(userID)
			userPinnedIDsKey := GetUserPinnedConversationIDsKey(userID)
			keys = append(keys, userConvIDsKey, userPinnedIDsKey)
		}
		s.localCache.PublishDelete(keys)
	}
	return s.ConversationService.CreateGroupChatConversations(ctx, in, opts...)
}

func (s *ConversationService) SetConversationMaxSeq(ctx context.Context, in *pbconv.SetConversationMaxSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMaxSeqResp, error) {
	if s.localCache != nil {
		keys := make([]string, 0)
		for _, ownerUserID := range in.OwnerUserID {
			key := GetUserConversationKey(ownerUserID, in.ConversationID)
			keys = append(keys, key)
		}
		s.localCache.PublishDelete(keys)
	}
	return s.ConversationService.SetConversationMaxSeq(ctx, in, opts...)
}

func (s *ConversationService) SetConversationMinSeq(ctx context.Context, in *pbconv.SetConversationMinSeqReq, opts ...grpc.CallOption) (*pbconv.SetConversationMinSeqResp, error) {
	if s.localCache != nil {
		keys := make([]string, 0)
		for _, ownerUserID := range in.OwnerUserID {
			key := GetUserConversationKey(ownerUserID, in.ConversationID)
			keys = append(keys, key)
		}
		s.localCache.PublishDelete(keys)
	}
	return s.ConversationService.SetConversationMinSeq(ctx, in, opts...)
}

func (s *ConversationService) GetConversationIDs(ctx context.Context, in *pbconv.GetConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetConversationIDsResp, error) {
	if s.localCache == nil {
		return s.ConversationService.GetConversationIDs(ctx, in, opts...)
	}

	userConvIDsKey := GetUserConversationIDsKey(in.UserID)
	conversationIDs, err := s.localCache.Take(userConvIDsKey, func() (any, error) {
		resp, err := s.ConversationService.GetConversationIDs(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		return resp.ConversationIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return &pbconv.GetConversationIDsResp{
		ConversationIDs: conversationIDs.([]string),
	}, nil
}

func (s *ConversationService) SetConversations(ctx context.Context, in *pbconv.SetConversationsReq, opts ...grpc.CallOption) (*pbconv.SetConversationsResp, error) {
	if s.localCache != nil {
		keys := make([]string, 0)
		for _, userID := range in.UserIDs {
			userConvIDsKey := GetUserConversationIDsKey(userID)
			userPinnedIDsKey := GetUserPinnedConversationIDsKey(userID)
			keys = append(keys, userConvIDsKey, userPinnedIDsKey)
		}
		s.localCache.PublishDelete(keys)
	}
	return s.ConversationService.SetConversations(ctx, in, opts...)
}

func (s *ConversationService) UpdateConversation(ctx context.Context, in *pbconv.UpdateConversationReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationResp, error) {
	if s.localCache != nil && in.ConversationID != "" {
		keys := make([]string, 0)
		for _, userID := range in.UserIDs {
			userConvKey := GetUserConversationKey(userID, in.ConversationID)
			userConvIDs := GetUserConversationIDsKey(userID)
			userPinnedIDsKey := GetUserPinnedConversationIDsKey(userID)
			keys = append(keys, userConvKey, userConvIDs, userPinnedIDsKey)
		}
		s.localCache.PublishDelete(keys)
	}
	return s.ConversationService.UpdateConversation(ctx, in, opts...)
}

func (s *ConversationService) UpdateConversationsByUser(ctx context.Context, in *pbconv.UpdateConversationsByUserReq, opts ...grpc.CallOption) (*pbconv.UpdateConversationsByUserResp, error) {
	if s.localCache != nil && in.UserID != "" {
		key := GetUserConversationIDsKey(in.UserID)
		s.localCache.PublishDelete([]string{key})
	}
	return s.ConversationService.UpdateConversationsByUser(ctx, in, opts...)
}

func (s *ConversationService) DeleteConversations(ctx context.Context, in *pbconv.DeleteConversationsReq, opts ...grpc.CallOption) (*pbconv.DeleteConversationsResp, error) {
	if s.localCache != nil && in.OwnerUserID != "" {
		keys := make([]string, 0)
		for _, convID := range in.ConversationIDs {
			userConvKey := GetUserConversationKey(in.OwnerUserID, convID)
			keys = append(keys, userConvKey)
		}
		userConvIDsKey := GetUserConversationIDsKey(in.OwnerUserID)
		userPinnedIDsKey := GetUserPinnedConversationIDsKey(in.OwnerUserID)
		keys = append(keys, userConvIDsKey, userPinnedIDsKey)
		s.localCache.PublishDelete(keys)
	}
	return s.ConversationService.DeleteConversations(ctx, in, opts...)
}

// 获取置顶会话ID列表
func (s *ConversationService) GetPinnedConversationIDs(ctx context.Context, in *pbconv.GetPinnedConversationIDsReq, opts ...grpc.CallOption) (*pbconv.GetPinnedConversationIDsResp, error) {
	if s.localCache == nil {
		return s.ConversationService.GetPinnedConversationIDs(ctx, in, opts...)
	}
	key := GetUserPinnedConversationIDsKey(in.UserID)
	pinnedIDs, err := s.localCache.Take(key, func() (any, error) {
		resp, err := s.ConversationService.GetPinnedConversationIDs(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		return resp.ConversationIDs, nil
	})
	if err != nil {
		return nil, err
	}
	return &pbconv.GetPinnedConversationIDsResp{
		ConversationIDs: pinnedIDs.([]string),
	}, nil
}

func (s *ConversationService) GetSingleConversationRecvMsgOpt(ctx context.Context, userID, conversationID string) (int32, error) {
	conv, err := s.GetConversation(ctx, &pbconv.GetConversationReq{
		OwnerUserID:    userID,
		ConversationID: conversationID,
	})
	if err != nil {
		return 0, err
	}
	return conv.Conversation.RecvMsgOpt, nil
}
