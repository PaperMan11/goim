package sequser

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/syncx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrSeqUserNotFound = errors.New("seq user not found")
)

// EffectiveMaxSeq 统一 "0 = 无限制" 语义：用户未设置 max_seq（默认 0）时，
// 可见消息上界为会话级 convMaxSeq；被踢出/退群冻结时返回冻结值。
// 所有计算用户可见上界/未读数的读路径必须使用此函数，禁止直接比较 SeqUser.MaxSeq。
func EffectiveMaxSeq(userMaxSeq, convMaxSeq int64) int64 {
	if userMaxSeq > 0 && userMaxSeq < convMaxSeq {
		return userMaxSeq
	}
	return convMaxSeq
}

// EffectiveMinSeq 统一 min_seq 语义：用户级 min_seq（入群边界/清空记录）与会话级取较大者。
func EffectiveMinSeq(userMinSeq, convMinSeq int64) int64 {
	if userMinSeq > convMinSeq {
		return userMinSeq
	}
	return convMinSeq
}

type SeqUserModel interface {
	BatchGetUserReadSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]int64, error)
	BatchGetUserSeqs(ctx context.Context, userID string, conversationIDs []string) (map[string]*model.SeqUser, error)
	FindAllUserSeqs(ctx context.Context, userID string) ([]*model.SeqUser, error)
	GetUserSeq(ctx context.Context, userID string, conversationID string) (*model.SeqUser, error)
	SetUserMaxSeq(ctx context.Context, userID string, conversationID string, maxSeq int64) error
	SetUserMinSeq(ctx context.Context, userID string, conversationID string, minSeq int64) error
	SetUserReadSeq(ctx context.Context, userID string, conversationID string, readSeq int64) error
}

type defaultSeqUserModel struct {
	mod *mon.Model
}

func NewSeqUserModel(mod *mon.Model, barrier syncx.SingleFlight, rdb goredis.UniversalClient, enableCache bool) SeqUserModel {
	m := &defaultSeqUserModel{
		mod: mod,
	}
	if !enableCache || rdb == nil {
		return m
	}
	return NewCachedSeqUserModel(m, rdb, barrier)
}

func (s *defaultSeqUserModel) SetUserReadSeq(ctx context.Context, userID, conversationID string, readSeq int64) error {
	update := bson.M{
		// read_seq 单调递增：用 $max 防止 Kafka 乱序/重试时旧值覆盖新值
		"$max": bson.M{
			"read_seq": readSeq,
		},
		"$set": bson.M{
			"updated_at": timex.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
			"min_seq":         0,
			"max_seq":         0,
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
			"read_seq":        0,
		},
	}
	_, err := s.mod.UpdateOne(ctx, bson.M{"user_id": userID, "conversation_id": conversationID}, update, options.UpdateOne().SetUpsert(true))
	return err
}

func (s *defaultSeqUserModel) SetUserMinSeq(ctx context.Context, userID, conversationID string, minSeq int64) error {
	update := bson.M{
		// min_seq 单调递增（清空历史后不可恢复可见）：用 $max 防并发/重试回退
		"$max": bson.M{
			"min_seq": minSeq,
		},
		"$set": bson.M{
			"updated_at": timex.Now(),
		},
		"$setOnInsert": bson.M{
			"user_id":         userID,
			"conversation_id": conversationID,
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
