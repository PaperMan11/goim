package sequser

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrSeqUserNotFound = errors.New("seq user not found")
)

type SeqUserModel interface {
	BatchGetUserReadSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]int64, error)
	BatchGetUserSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]*model.SeqUser, error)
	FindAllUserSeqs(ctx context.Context, userID string) ([]*model.SeqUser, error)
	GetUserSeq(ctx context.Context, userID string, conversationID string) (*model.SeqUser, error)
	SetUserMaxSeq(ctx context.Context, userID string, conversationID string, maxSeq int64) error
	SetUserMinSeq(ctx context.Context, userID string, conversationID string, minSeq int64) error
	SetUserReadSeq(ctx context.Context, userID string, conversationID string, readSeq int64) error
	UpsertUserSeq(ctx context.Context, userID string, conversationID string, minSeq int64, maxSeq, readSeq int64) error
}

type defaultSeqUserModel struct {
	mod *mon.Model
}

func NewSeqUserModel(mod *mon.Model) SeqUserModel {
	return &defaultSeqUserModel{
		mod: mod,
	}
}

func (s *defaultSeqUserModel) UpsertUserSeq(ctx context.Context, userID, conversationID string, minSeq, maxSeq, readSeq int64) error {
	update := bson.M{
		"$set": bson.M{
			"max_seq":    maxSeq,
			"min_seq":    minSeq,
			"read_seq":   readSeq,
			"updated_at": timex.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
			"min_seq":         minSeq,
			"max_seq":         maxSeq,
			"read_seq":        readSeq,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func (s *defaultSeqUserModel) SetUserReadSeq(ctx context.Context, userID, conversationID string, readSeq int64) error {
	update := bson.M{
		"$set": bson.M{
			"read_seq":   readSeq,
			"updated_at": timex.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
			"min_seq":         0,
			"max_seq":         0,
			"read_seq":        0,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func (s *defaultSeqUserModel) SetUserMaxSeq(ctx context.Context, userID, conversationID string, maxSeq int64) error {
	update := bson.M{
		"$set": bson.M{
			"max_seq":    maxSeq,
			"updated_at": timex.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
			"min_seq":         0,
			"max_seq":         0,
			"read_seq":        0,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func (s *defaultSeqUserModel) SetUserMinSeq(ctx context.Context, userID, conversationID string, minSeq int64) error {
	update := bson.M{
		"$set": bson.M{
			"min_seq":    minSeq,
			"updated_at": timex.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
			"min_seq":         0,
			"max_seq":         0,
			"read_seq":        0,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func (s *defaultSeqUserModel) GetUserSeq(ctx context.Context, userID, conversationID string) (*model.SeqUser, error) {
	var seq model.SeqUser
	err := s.mod.FindOne(ctx, &seq, bson.M{"user_id": userID, "conversation_id": conversationID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrSeqUserNotFound
		}
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

func (s *defaultSeqUserModel) FindAllUserSeqs(ctx context.Context, userID string) ([]*model.SeqUser, error) {
	var seqs []*model.SeqUser
	err := s.mod.Find(ctx, &seqs, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	return seqs, nil
}
