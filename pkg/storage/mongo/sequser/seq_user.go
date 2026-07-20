package sequser

import (
	"context"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SeqUserModel interface {
	BatchGetUserReadSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]int64, error)
	BatchGetUserSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]*model.SeqUser, error)
	GetUserSeq(ctx context.Context, userID string, conversationID string) (*model.SeqUser, error)
	SetUserMaxSeq(ctx context.Context, userID string, conversationID string, maxSeq int64) error
	SetUserMinSeq(ctx context.Context, userID string, conversationID string, minSeq int64) error
	SetUserReadSeq(ctx context.Context, userID string, conversationID string, readSeq int64) error
	UpsertUserSeq(ctx context.Context, userID string, conversationID string, minSeq int64, maxSeq int64) error
}

type defaultSeqUserModel struct {
	mod *mon.Model
}

func NewSeqUserModel(mod *mon.Model) SeqUserModel {
	return &defaultSeqUserModel{
		mod: mod,
	}
}

func (s *defaultSeqUserModel) UpsertUserSeq(ctx context.Context, userID, conversationID string, minSeq, maxSeq int64) error {
	update := bson.M{
		"$set": bson.M{
			"max_seq":    maxSeq,
			"updated_at": timex.Now(),
		},
		"$min": bson.M{"min_seq": minSeq},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
			"min_seq":         minSeq,
			"read_seq":        0,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, update)
	return err
}

func (s *defaultSeqUserModel) SetUserReadSeq(ctx context.Context, userID, conversationID string, readSeq int64) error {
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, bson.M{
		"$set": bson.M{
			"read_seq":   readSeq,
			"updated_at": timex.Now(),
		},
	})
	return err
}

func (s *defaultSeqUserModel) SetUserMaxSeq(ctx context.Context, userID, conversationID string, maxSeq int64) error {
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, bson.M{
		"$set": bson.M{
			"max_seq":    maxSeq,
			"updated_at": timex.Now(),
		},
	})
	return err
}

func (s *defaultSeqUserModel) SetUserMinSeq(ctx context.Context, userID, conversationID string, minSeq int64) error {
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, bson.M{
		"$set": bson.M{
			"min_seq":    minSeq,
			"updated_at": timex.Now(),
		},
	})
	return err
}

func (s *defaultSeqUserModel) GetUserSeq(ctx context.Context, userID, conversationID string) (*model.SeqUser, error) {
	var seq model.SeqUser
	err := s.mod.FindOne(ctx, &seq, bson.M{"user_id": userID, "conversation_id": conversationID})
	if err != nil {
		return nil, err
	}
	return &seq, nil
}

func (s *defaultSeqUserModel) BatchGetUserSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]*model.SeqUser, error) {
	var seqs []model.SeqUser
	err := s.mod.Find(ctx, &seqs, bson.M{"user_id": userID, "conversation_id": bson.M{"$in": conversationIDs}})
	if err != nil {
		return nil, err
	}
	result := make(map[string]*model.SeqUser)
	for _, seq := range seqs {
		result[seq.ConversationID] = &seq
	}
	return result, nil
}

func (s *defaultSeqUserModel) BatchGetUserReadSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]int64, error) {
	seqs, err := s.BatchGetUserSeqs(ctx, userID, conversationIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for convID, seq := range seqs {
		result[convID] = seq.ReadSeq
	}
	return result, nil
}
