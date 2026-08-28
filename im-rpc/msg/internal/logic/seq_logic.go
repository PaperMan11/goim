package logic

import (
	"context"

	pbconv "github.com/PaperMan11/goim/pkg/protocol/conversation"
	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
)

func (l *Logic) GetMaxSeq(ctx context.Context, req *sdkws.GetMaxSeqReq) (*sdkws.GetMaxSeqResp, error) {
	seqs, err := l.svcCtx.SeqUserModel.FindAllUserSeqs(ctx, req.UserID)
	if err != nil {
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
	readSeqs, err := l.svcCtx.SeqUserModel.BatchGetUserReadSeqs(ctx, userID, conversationIDs)
	if err != nil {
		l.Errorf("batch user readed conversation req failed, userID: %s, err: %v", userID, err)
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
		seqs[convID] = &pbmsg.Seqs{
			MaxSeq:     seqConvs[convID].MaxSeq,
			MaxSeqTime: seqConvs[convID].UpdatedAt.Unix(),
			HasReadSeq: readSeqs[convID],
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
		return nil, err
	}
	err = l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, req.HasReadSeq)
	if err != nil {
		return nil, err
	}
	return &pbmsg.SetConversationHasReadSeqResp{}, nil
}

func (l *Logic) SetUserConversationMaxSeq(ctx context.Context, req *pbmsg.SetUserConversationMaxSeqReq) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}
	for _, userID := range req.OwnerUserID {
		if err := l.svcCtx.SeqUserModel.SetUserMaxSeq(ctx, userID, req.ConversationID, req.MaxSeq); err != nil {
			return nil, err
		}
	}
	return &pbmsg.SetUserConversationMaxSeqResp{}, nil
}

func (l *Logic) SetUserConversationMinSeq(ctx context.Context, req *pbmsg.SetUserConversationMinSeqReq) (*pbmsg.SetUserConversationMinSeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}
	for _, userID := range req.OwnerUserID {
		if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, req.ConversationID, req.MinSeq); err != nil {
			return nil, err
		}
	}
	return &pbmsg.SetUserConversationMinSeqResp{}, nil
}

func (l *Logic) SetUserConversationsMinSeq(ctx context.Context, req *pbmsg.SetUserConversationsMinSeqReq) (*pbmsg.SetUserConversationsMinSeqResp, error) {
	err := l.requireAdmin()
	if err != nil {
		return nil, err
	}
	for _, userID := range req.UserIDs {
		if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, req.ConversationID, req.Seq); err != nil {
			return nil, err
		}
	}
	return &pbmsg.SetUserConversationsMinSeqResp{}, nil
}
