package seqconversation

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	ErrSeqConversationNotFound = errors.New("seq conversation not found")
)

type SeqConversationModel interface {
	BatchGetConversationSeqs(ctx context.Context, conversationIDs []string) (map[string]*model.SeqConversation, error)
	GetSeqConversation(ctx context.Context, conversationID string) (*model.SeqConversation, error)
	GetConversationMaxSeq(ctx context.Context, conversationID string) (int64, error)
	GetConversationMinSeq(ctx context.Context, conversationID string) (int64, error)
	SetConversationMaxSeq(ctx context.Context, conversationID string, maxSeq int64) error
	SetConversationMinSeq(ctx context.Context, conversationID string, minSeq int64) error
}

type defaultSeqConversationModel struct {
	mod *mon.Model
}

func NewSeqConversationModel(mod *mon.Model) SeqConversationModel {
	return &defaultSeqConversationModel{
		mod: mod,
	}
}

func (s *defaultSeqConversationModel) SetConversationMaxSeq(ctx context.Context, conversationID string, maxSeq int64) error {
	_, err := s.mod.UpdateOne(ctx, bson.M{"conversation_id": conversationID}, bson.M{
		"$set": bson.M{
			"max_seq":    maxSeq,
			"updated_at": timex.Now(),
		},
	})
	return err
}

func (s *defaultSeqConversationModel) SetConversationMinSeq(ctx context.Context, conversationID string, minSeq int64) error {
	_, err := s.mod.UpdateOne(ctx, bson.M{"conversation_id": conversationID}, bson.M{
		"$set": bson.M{
			"min_seq":    minSeq,
			"updated_at": timex.Now(),
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

func (s *defaultSeqConversationModel) BatchGetConversationSeqs(ctx context.Context, conversationIDs []string) (map[string]*model.SeqConversation, error) {
	var seqs []*model.SeqConversation
	err := s.mod.Find(ctx, &seqs, bson.M{"conversation_id": bson.M{"$in": conversationIDs}})
	if err != nil {
		return nil, err
	}
	result := make(map[string]*model.SeqConversation)
	for _, seq := range seqs {
		result[seq.ConversationID] = seq
	}
	return result, nil
}

func (s *defaultSeqConversationModel) GetSeqConversation(ctx context.Context, conversationID string) (*model.SeqConversation, error) {
	var seq model.SeqConversation
	err := s.mod.FindOne(ctx, &seq, bson.M{"conversation_id": conversationID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrSeqConversationNotFound
		}
		return nil, err
	}
	return &seq, nil
}
