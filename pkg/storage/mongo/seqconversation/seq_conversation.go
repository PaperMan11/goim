package seqconversation

import (
	"context"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SeqConversationModel interface {
	BatchGetConversationMaxSeqs(ctx context.Context, conversationIDs []string) (map[string]int64, error)
	GetConversationMaxSeq(ctx context.Context, conversationID string) (int64, error)
	GetConversationMinSeq(ctx context.Context, conversationID string) (int64, error)
	UpsertConversationMaxSeq(ctx context.Context, conversationID string, maxSeq int64) error
	UpsertConversationMinSeq(ctx context.Context, conversationID string, minSeq int64) error
}

type defaultSeqConversationModel struct {
	mod *mon.Model
}

func NewSeqConversationModel(mod *mon.Model) SeqConversationModel {
	return &defaultSeqConversationModel{
		mod: mod,
	}
}

func (s *defaultSeqConversationModel) UpsertConversationMaxSeq(ctx context.Context, conversationID string, maxSeq int64) error {
	update := bson.M{
		"$set": bson.M{
			"max_seq":    maxSeq,
			"updated_at": time.Now(),
		},
		"$setOnInsert": bson.M{
			"conversation_id": conversationID,
			"min_seq":         maxSeq,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"conversation_id": conversationID}, update)
	return err
}

func (s *defaultSeqConversationModel) UpsertConversationMinSeq(ctx context.Context, conversationID string, minSeq int64) error {
	_, err := s.mod.UpdateOne(ctx, bson.M{"conversation_id": conversationID}, bson.M{
		"$set": bson.M{
			"min_seq":    minSeq,
			"updated_at": time.Now(),
		},
	})
	return err
}

func (s *defaultSeqConversationModel) GetConversationMaxSeq(ctx context.Context, conversationID string) (int64, error) {
	var seq model.SeqConversation
	err := s.mod.FindOne(ctx, &seq, bson.M{"conversation_id": conversationID})
	if err != nil {
		return 0, err
	}
	return seq.MaxSeq, nil
}

func (s *defaultSeqConversationModel) GetConversationMinSeq(ctx context.Context, conversationID string) (int64, error) {
	var seq model.SeqConversation
	err := s.mod.FindOne(ctx, &seq, bson.M{"conversation_id": conversationID})
	if err != nil {
		return 0, err
	}
	return seq.MinSeq, nil
}

func (s *defaultSeqConversationModel) BatchGetConversationMaxSeqs(ctx context.Context, conversationIDs []string) (map[string]int64, error) {
	var seqs []model.SeqConversation
	err := s.mod.Find(ctx, &seqs, bson.M{"conversation_id": bson.M{"$in": conversationIDs}})
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for _, seq := range seqs {
		result[seq.ConversationID] = seq.MaxSeq
	}
	return result, nil
}
