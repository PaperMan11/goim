package msg

import (
	"context"
	"time"

	"github.com/PaperMan11/goim/pkg/protocol/sdkws"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MsgModel interface {
	BatchInsert(ctx context.Context, msgs []*model.Message) error
	DeleteBySeq(ctx context.Context, conversationID string, seqs []int64) error
	DeleteByTimestamp(ctx context.Context, conversationIDs []string, timestamp int64) error
	FindByConversationID(ctx context.Context, conversationID string, seqStart int64, seqEnd int64, limit int) ([]*model.Message, error)
	FindBySeq(ctx context.Context, conversationID string, seq int64) (*model.Message, error)
	FindLatestMsg(ctx context.Context, conversationID string) (*model.Message, error)
	GetMaxSeq(ctx context.Context, conversationID string) (int64, error)
	GetMinSeq(ctx context.Context, conversationID string) (int64, error)
	Insert(ctx context.Context, msg *model.Message) error
	ToSDKMsg(msg *model.Message) *sdkws.MsgData
	UpdateRevoke(ctx context.Context, conversationID string, seq int64, revokedContent *model.MessageRevokedContent) error
}

type defaultMsgModel struct {
	mod *mon.Model
}

func NewMsgModel(mod *mon.Model) MsgModel {
	return &defaultMsgModel{
		mod: mod,
	}
}

func (m *defaultMsgModel) Insert(ctx context.Context, msg *model.Message) error {
	_, err := m.mod.InsertOne(ctx, msg)
	return err
}

func (m *defaultMsgModel) BatchInsert(ctx context.Context, msgs []*model.Message) error {
	var docs []interface{}
	for _, msg := range msgs {
		docs = append(docs, msg)
	}
	_, err := m.mod.InsertMany(ctx, docs)
	return err
}

func (m *defaultMsgModel) FindByConversationID(ctx context.Context, conversationID string, seqStart, seqEnd int64, limit int) ([]*model.Message, error) {
	var msgs []*model.Message
	filter := bson.M{
		"conversation_id": conversationID,
		"seq":             bson.M{"$gte": seqStart, "$lte": seqEnd},
	}
	opts := options.Find().SetSort(bson.M{"seq": 1}).SetLimit(int64(limit))
	err := m.mod.Find(ctx, &msgs, filter, opts)
	return msgs, err
}

func (m *defaultMsgModel) FindBySeq(ctx context.Context, conversationID string, seq int64) (*model.Message, error) {
	var msg model.Message
	err := m.mod.FindOne(ctx, &msg, bson.M{
		"conversation_id": conversationID,
		"seq":             seq,
	})
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *defaultMsgModel) FindLatestMsg(ctx context.Context, conversationID string) (*model.Message, error) {
	var msg model.Message
	opts := options.FindOne().SetSort(bson.M{"seq": -1})
	err := m.mod.FindOne(ctx, &msg, bson.M{"conversation_id": conversationID}, opts)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *defaultMsgModel) UpdateRevoke(ctx context.Context, conversationID string, seq int64, revokedContent *model.MessageRevokedContent) error {
	update := bson.M{
		"$set": bson.M{
			"is_revoked":      true,
			"revoked_content": revokedContent,
			"updated_at":      timex.Now(),
		},
	}
	_, err := m.mod.UpdateOne(ctx, bson.M{
		"conversation_id": conversationID,
		"seq":             seq,
	}, update)
	return err
}

func (m *defaultMsgModel) DeleteBySeq(ctx context.Context, conversationID string, seqs []int64) error {
	_, err := m.mod.DeleteMany(ctx, bson.M{
		"conversation_id": conversationID,
		"seq":             bson.M{"$in": seqs},
	})
	return err
}

func (m *defaultMsgModel) DeleteByTimestamp(ctx context.Context, conversationIDs []string, timestamp int64) error {
	_, err := m.mod.DeleteMany(ctx, bson.M{
		"conversation_id": bson.M{"$in": conversationIDs},
		"send_time":       bson.M{"$lte": time.Unix(timestamp/1000, 0)},
	})
	return err
}

func (m *defaultMsgModel) GetMaxSeq(ctx context.Context, conversationID string) (int64, error) {
	var result struct {
		MaxSeq int64 `bson:"max_seq"`
	}
	opts := options.Aggregate().SetAllowDiskUse(true)
	pipeline := []bson.M{
		{"$match": bson.M{"conversation_id": conversationID}},
		{"$group": bson.M{"max_seq": bson.M{"$max": "$seq"}}},
	}
	cursor, err := m.mod.Collection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.MaxSeq, nil
	}
	return 0, nil
}

func (m *defaultMsgModel) GetMinSeq(ctx context.Context, conversationID string) (int64, error) {
	var result struct {
		MinSeq int64 `bson:"min_seq"`
	}
	opts := options.Aggregate().SetAllowDiskUse(true)
	pipeline := []bson.M{
		{"$match": bson.M{"conversation_id": conversationID}},
		{"$group": bson.M{"min_seq": bson.M{"$min": "$seq"}}},
	}
	cursor, err := m.mod.Collection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.MinSeq, nil
	}
	return 0, nil
}

func (m *defaultMsgModel) ToSDKMsg(msg *model.Message) *sdkws.MsgData {
	return &sdkws.MsgData{
		SendID:           msg.SendID,
		RecvID:           msg.RecvID,
		GroupID:          msg.GroupID,
		ClientMsgID:      msg.ClientMsgID,
		ServerMsgID:      msg.ServerMsgID,
		SenderPlatformID: int32(msg.SenderPlatformID),
		SenderNickname:   msg.SenderNickname,
		SenderFaceURL:    msg.SenderFaceURL,
		SessionType:      int32(msg.SessionType),
		MsgFrom:          int32(msg.MsgFrom),
		ContentType:      int32(msg.ContentType),
		Content:          msg.Content,
		Seq:              msg.Seq,
		SendTime:         msg.SendTime.UnixMilli(),
		CreateTime:       msg.CreateTime.UnixMilli(),
		Status:           int32(msg.Status),
		IsRead:           msg.IsRead,
		Options:          msg.Options,
		AtUserIDList:     msg.AtUserIDList,
		AttachedInfo:     msg.AttachedInfo,
		Ex:               msg.Extra,
	}
}
