package logic

import (
	"context"

	pbmsg "github.com/PaperMan11/goim/pkg/protocol/msg"
	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
)

func (l *Logic) GetMaxSeq(ctx context.Context, req *sdkws.GetMaxSeqReq) (*sdkws.GetMaxSeqResp, error) {
	return &sdkws.GetMaxSeqResp{
		MaxSeqs: make(map[string]int64),
		MinSeqs: make(map[string]int64),
	}, nil
}

func (l *Logic) GetMaxSeqs(ctx context.Context, req *pbmsg.GetMaxSeqsReq) (*pbmsg.SeqsInfoResp, error) {
	maxSeqs, err := l.svcCtx.SeqConversationModel.BatchGetConversationMaxSeqs(ctx, req.ConversationIDs)
	if err != nil {
		return nil, err
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
	maxSeq, err := l.svcCtx.SeqConversationModel.GetConversationMaxSeq(ctx, req.ConversationID)
	if err != nil {
		return nil, err
	}
	return &pbmsg.GetConversationMaxSeqResp{
		MaxSeq: maxSeq,
	}, nil
}

func (l *Logic) GetConversationsHasReadAndMaxSeq(ctx context.Context, req *pbmsg.GetConversationsHasReadAndMaxSeqReq) (*pbmsg.GetConversationsHasReadAndMaxSeqResp, error) {
	maxSeqs, err := l.svcCtx.SeqConversationModel.BatchGetConversationMaxSeqs(ctx, req.ConversationIDs)
	if err != nil {
		return nil, err
	}
	readSeqs, err := l.svcCtx.SeqUserModel.BatchGetUserReadSeqs(ctx, req.UserID, req.ConversationIDs)
	if err != nil {
		return nil, err
	}

	seqs := make(map[string]*pbmsg.Seqs)
	for _, convID := range req.ConversationIDs {
		seqs[convID] = &pbmsg.Seqs{
			MaxSeq:     maxSeqs[convID],
			HasReadSeq: readSeqs[convID],
		}
	}

	return &pbmsg.GetConversationsHasReadAndMaxSeqResp{
		Seqs: seqs,
	}, nil
}

func (l *Logic) SetConversationHasReadSeq(ctx context.Context, req *pbmsg.SetConversationHasReadSeqReq) (*pbmsg.SetConversationHasReadSeqResp, error) {
	err := l.svcCtx.SeqUserModel.SetUserReadSeq(ctx, req.UserID, req.ConversationID, req.HasReadSeq)
	if err != nil {
		return nil, err
	}
	return &pbmsg.SetConversationHasReadSeqResp{}, nil
}

func (l *Logic) SetUserConversationMaxSeq(ctx context.Context, req *pbmsg.SetUserConversationMaxSeqReq) (*pbmsg.SetUserConversationMaxSeqResp, error) {
	for _, userID := range req.OwnerUserID {
		if err := l.svcCtx.SeqUserModel.SetUserMaxSeq(ctx, userID, req.ConversationID, req.MaxSeq); err != nil {
			return nil, err
		}
	}
	return &pbmsg.SetUserConversationMaxSeqResp{}, nil
}

func (l *Logic) SetUserConversationMinSeq(ctx context.Context, req *pbmsg.SetUserConversationMinSeqReq) (*pbmsg.SetUserConversationMinSeqResp, error) {
	for _, userID := range req.OwnerUserID {
		if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, req.ConversationID, req.MinSeq); err != nil {
			return nil, err
		}
	}
	return &pbmsg.SetUserConversationMinSeqResp{}, nil
}

func (l *Logic) SetUserConversationsMinSeq(ctx context.Context, req *pbmsg.SetUserConversationsMinSeqReq) (*pbmsg.SetUserConversationsMinSeqResp, error) {
	for _, userID := range req.UserIDs {
		if err := l.svcCtx.SeqUserModel.SetUserMinSeq(ctx, userID, req.ConversationID, req.Seq); err != nil {
			return nil, err
		}
	}
	return &pbmsg.SetUserConversationsMinSeqResp{}, nil
}
