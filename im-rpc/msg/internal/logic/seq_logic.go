package logic

import (
	"context"

	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/mongo/sequser"
)

func (l *Logic) GetMaxSeq(ctx context.Context, req *sdkws.GetMaxSeqReq) (*sdkws.GetMaxSeqResp, error) {
	seqs, err := l.svcCtx.SeqUserModel.FindAllUserSeqs(ctx, req.UserID)
	if err != nil {
		l.Errorf("get max seq failed, userID: %s, err: %v", req.UserID, err)
		return nil, err
	}
	maxSeqs := make(map[string]int64)
	minSeqs := make(map[string]int64)
	for _, seq := range seqs {
		maxSeqs[seq.ConversationID] = seq.MaxSeq
		minSeqs[seq.ConversationID] = seq.MinSeq
	}
	return &sdkws.GetMaxSeqResp{
		MaxSeqs: maxSeqs,
		MinSeqs: minSeqs,
	}, nil
}

func (l *Logic) GetMaxSeqs(ctx context.Context, req *pbmsg.GetMaxSeqsReq) (*pbmsg.SeqsInfoResp, error) {
	if l.svcCtx.SeqAllocator == nil {
		convsMap, err := l.svcCtx.SeqConversationModel.BatchGetConversationSeqs(ctx, req.ConversationIDs)
		if err != nil {
			l.Errorf("get max seqs via model failed, convCount: %d, err: %v", len(req.ConversationIDs), err)
			return nil, err
		}
		maxSeqs := make(map[string]int64)
		for convID, conv := range convsMap {
			maxSeqs[convID] = conv.MaxSeq
		}
		return &pbmsg.SeqsInfoResp{
			MaxSeqs: maxSeqs,
		}, nil
	}
	maxSeqs := make(map[string]int64)
	for _, convID := range req.ConversationIDs {
		maxSeq, err := l.svcCtx.SeqAllocator.GetCurrent(ctx, convID)
		if err != nil {
			l.Errorf("get conversation max seq failed: %v, convID: %s", err, convID)
			continue
		}
		maxSeqs[convID] = maxSeq
	}
	return &pbmsg.SeqsInfoResp{
		MaxSeqs: maxSeqs,
	}, nil
}

func (l *Logic) GetHasReadSeqs(ctx context.Context, req *pbmsg.GetHasReadSeqsReq) (*pbmsg.SeqsInfoResp, error) {
	readSeqs, err := l.svcCtx.SeqUserModel.BatchGetUserReadSeqs(ctx, req.UserID, req.ConversationIDs)
	if err != nil {
		l.Errorf("get has read seqs failed, userID: %s, convCount: %d, err: %v", req.UserID, len(req.ConversationIDs), err)
		return nil, err
	}
	return &pbmsg.SeqsInfoResp{
		MaxSeqs: readSeqs,
	}, nil
}

func (l *Logic) GetConversationMaxSeq(ctx context.Context, req *pbmsg.GetConversationMaxSeqReq) (*pbmsg.GetConversationMaxSeqResp, error) {
	if l.svcCtx.SeqAllocator == nil {
		maxSeq, err := l.svcCtx.SeqConversationModel.GetConversationMaxSeq(ctx, req.ConversationID)
		if err != nil {
			l.Errorf("get conversation max seq failed: %v, convID: %s", err, req.ConversationID)
			return nil, err
		}
		return &pbmsg.GetConversationMaxSeqResp{
			MaxSeq: maxSeq,
		}, nil
	}
	maxSeq, err := l.svcCtx.SeqAllocator.GetCurrent(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get conversation max seq failed: %v, convID: %s", err, req.ConversationID)
		return nil, err
	}
	return &pbmsg.GetConversationMaxSeqResp{
		MaxSeq: maxSeq,
	}, nil
}

func (l *Logic) GetConversationsHasReadAndMaxSeq(ctx context.Context, req *pbmsg.GetConversationsHasReadAndMaxSeqReq) (*pbmsg.GetConversationsHasReadAndMaxSeqResp, error) {
	err := l.requireSelfOrAdmin(req.UserID)
	if err != nil {
		l.Errorf("require self or admin failed, err: %v, userID: %s", err, req.UserID)
		return nil, err
	}

	userID := req.UserID
	conversationIDs := req.ConversationIDs
	if len(conversationIDs) == 0 {
		resp, err := l.svcCtx.ConvService.GetConversationIDs(ctx, &pbconv.GetConversationIDsReq{
			UserID: req.UserID,
		})
		if err != nil {
			l.Errorf("get conversation ids failed, userID: %s, err: %v", userID, err)
			return nil, err
		}
		conversationIDs = resp.ConversationIDs
	}

	seqConvs, err := l.svcCtx.SeqConversationModel.BatchGetConversationSeqs(ctx, conversationIDs)
	if err != nil {
		return nil, err
	}
	userSeqs, err := l.svcCtx.SeqUserModel.BatchGetUserSeqs(ctx, userID, conversationIDs)
	if err != nil {
		l.Errorf("batch get user seqs failed, userID: %s, err: %v", userID, err)
		return nil, err
	}

	var pinnedConvs []string
	if req.GetReturnPinned() {
		resp, err := l.svcCtx.ConvService.GetPinnedConversationIDs(ctx, &pbconv.GetPinnedConversationIDsReq{UserID: userID})
		if err != nil {
			l.Errorf("get pinned conversation ids failed, userID: %s, err: %v", userID, err)
		}
		pinnedConvs = resp.GetConversationIDs()
	}

	seqs := make(map[string]*pbmsg.Seqs)
	for _, convID := range req.ConversationIDs {
		var convMax int64
		var maxSeqTime int64
		if cs, ok := seqConvs[convID]; ok && cs != nil {
			convMax = cs.MaxSeq
			maxSeqTime = cs.UpdatedAt.Unix()
		}
		var userMax, readSeq int64
		if us, ok := userSeqs[convID]; ok && us != nil {
			userMax = us.MaxSeq
			readSeq = us.ReadSeq
		}
		seqs[convID] = &pbmsg.Seqs{
			// 用户被踢/退群后 MaxSeq 冻结，需封顶会话级 maxSeq
			MaxSeq:     sequser.EffectiveMaxSeq(userMax, convMax),
			MaxSeqTime: maxSeqTime,
			HasReadSeq: readSeq,
		}
	}

	return &pbmsg.GetConversationsHasReadAndMaxSeqResp{
		Seqs:                  seqs,
		PinnedConversationIDs: pinnedConvs,
	}, nil
}

func (l *Logic) SetConversationHasReadSeq(ctx context.Context, req *pbmsg.SetConversationHasReadSeqReq) (*pbmsg.SetConversationHasReadSeqResp, error) {
	err := l.requireSelfOrAdmin(req.UserID)
	if err != nil {
		l.Errorf("require self or admin failed, err: %v, userID: %s", err, req.UserID)
		return nil, err
	}

	conversation, err := l.svcCtx.SeqConversationModel.GetSeqConversation(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get seq conversation failed, convID: %s, err: %v", req.ConversationID, err)
		return nil, err
	}
	hasReadSeq := req.HasReadSeq
	if hasReadSeq > conversation.MaxSeq {
		hasReadSeq = conversation.MaxSeq
	}
	err = l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, hasReadSeq)
	if err != nil {
		l.Errorf("set conversation has read seq failed, userID: %s, convID: %s, err: %v", req.UserID, req.ConversationID, err)
		return nil, err
	}
	l.Infof("set conversation has read seq success, userID: %s, convID: %s, seq: %d",
		req.UserID, req.ConversationID, hasReadSeq)
	return &pbmsg.SetConversationHasReadSeqResp{}, nil
}

func (l *Logic) SetUserConversationMaxSeq(ctx context.Context, req *pbmsg.SetUserConversationMaxSeqReq) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}

	conversation, err := l.svcCtx.SeqConversationModel.GetSeqConversation(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get seq conversation failed, convID: %s, err: %v", req.ConversationID, err)
		return nil, err
	}
	maxSeq := sequser.EffectiveMaxSeq(req.MaxSeq, conversation.MaxSeq)
	for _, userID := range req.OwnerUserID {
		if err := l.svcCtx.SeqUserModel.SetUserMaxSeq(ctx, userID, req.ConversationID, maxSeq); err != nil {
			l.Errorf("set user conversation max seq failed, userID: %s, convID: %s, err: %v",
				userID, req.ConversationID, err)
			return nil, err
		}
	}
	l.Infof("set user conversation max seq success, convID: %s, userCount: %d, maxSeq: %d",
		req.ConversationID, len(req.OwnerUserID), maxSeq)
	return &pbmsg.SetUserConversationMaxSeqResp{}, nil
}

func (l *Logic) SetUserConversationMinSeq(ctx context.Context, req *pbmsg.SetUserConversationMinSeqReq) (*pbmsg.SetUserConversationMinSeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}

	conversation, err := l.svcCtx.SeqConversationModel.GetSeqConversation(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get seq conversation failed, convID: %s, err: %v", req.ConversationID, err)
		return nil, err
	}
	for _, userID := range req.OwnerUserID {
		if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, req.ConversationID, sequser.EffectiveMinSeq(req.MinSeq, conversation.MinSeq)); err != nil {
			l.Errorf("set user conversation min seq failed, userID: %s, convID: %s, err: %v",
				userID, req.ConversationID, err)
			return nil, err
		}
	}
	l.Infof("set user conversation min seq success, convID: %s, userCount: %d",
		req.ConversationID, len(req.OwnerUserID))
	return &pbmsg.SetUserConversationMinSeqResp{}, nil
}

func (l *Logic) SetUserConversationsMinSeq(ctx context.Context, req *pbmsg.SetUserConversationsMinSeqReq) (*pbmsg.SetUserConversationsMinSeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}

	conversation, err := l.svcCtx.SeqConversationModel.GetSeqConversation(ctx, req.ConversationID)
	if err != nil {
		l.Errorf("get seq conversation failed, convID: %s, err: %v", req.ConversationID, err)
		return nil, err
	}
	minSeq := sequser.EffectiveMinSeq(req.Seq, conversation.MinSeq)
	for _, userID := range req.UserIDs {
		if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, req.ConversationID, minSeq); err != nil {
			l.Errorf("set user conversations min seq failed, userID: %s, convID: %s, err: %v",
				userID, req.ConversationID, err)
			return nil, err
		}
	}
	l.Infof("set user conversations min seq success, convID: %s, userCount: %d, minSeq: %d",
		req.ConversationID, len(req.UserIDs), minSeq)
	return &pbmsg.SetUserConversationsMinSeqResp{}, nil
}
