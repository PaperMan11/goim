package conversation

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrLatestMsgNotFound    = errors.New("conversation latest msg not found")
)

type ConversationModel interface {
	InsertConversation(ctx context.Context, convs []*model.Conversation) error
	UpsertConversation(ctx context.Context, conv *model.Conversation) error
	FindConversationIDsByOwner(ctx context.Context, ownerUserID string) ([]string, error)
	FindConversationsByConvIDs(ctx context.Context, convIDs []string) ([]*model.Conversation, error)
	FindConversation(ctx context.Context, ownerUserID, conversationID string) (*model.Conversation, error)
	FindConversationsByOwner(ctx context.Context, ownerUserID string) ([]*model.Conversation, error)
	FindPinnedConversationIDs(ctx context.Context, ownerUserID string) ([]string, error)
	FindConversationsByIDs(ctx context.Context, ownerUserID string, convIDs []string) ([]*model.Conversation, error)
	UpdateConversation(ctx context.Context, owner, convID string, updates map[string]any) error
	UpdateConversations(ctx context.Context, ownerUserID string, convIDs []string, updates map[string]any) error
	DeleteConversation(ctx context.Context, owner, convID string) error
	DeleteConversationsByOwner(ctx context.Context, ownerUserID string) error
	// 获取不进行推送的会话用户列表
	FindNoRecvConversationUserIDs(ctx context.Context, conversationID string) ([]string, error)
	FindUserNotNotifyConversationIDs(ctx context.Context, ownerUserID string) ([]string, error)

	UpsertConversationLatestMsg(ctx context.Context, latest *model.ConversationLatestMsg) error
	FindLatestMsg(ctx context.Context, owner, convID string) (*model.ConversationLatestMsg, error)
	FindLatestMsgsByOwner(ctx context.Context, owner string, limit int) ([]*model.ConversationLatestMsg, error)
	DeleteLatestMsg(ctx context.Context, owner, convID string) error
}

type defaultConversationModel struct {
	convMod   *mon.Model
	latestMod *mon.Model
}

func NewConversationModel(convMod, latestMod *mon.Model) ConversationModel {
	return &defaultConversationModel{
		convMod:   convMod,
		latestMod: latestMod,
	}
}

func (m *defaultConversationModel) InsertConversation(ctx context.Context, convs []*model.Conversation) error {
	var docs []any
	for _, conv := range convs {
		docs = append(docs, conv)
	}
	_, err := m.convMod.Collection.InsertMany(ctx, docs)
	return err
}

func (m *defaultConversationModel) UpsertConversation(ctx context.Context, conv *model.Conversation) error {
	filter := bson.M{
		"owner_user_id":   conv.OwnerUserID,
		"conversation_id": conv.ConversationID,
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.convMod.Collection.UpdateOne(ctx, filter, bson.M{"$set": conv}, opts)
	return err
}

func (m *defaultConversationModel) FindConversationIDsByOwner(ctx context.Context, ownerUserID string) ([]string, error) {
	var ids []string
	findOpts := options.Find().SetProjection(bson.M{"conversation_id": 1}).SetSort(bson.M{"_id": 1})
	cursor, err := m.convMod.Collection.Find(ctx, bson.M{"owner_user_id": ownerUserID}, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (m *defaultConversationModel) FindConversationsByConvIDs(ctx context.Context, convIDs []string) ([]*model.Conversation, error) {
	var convs []*model.Conversation
	filter := bson.M{
		"conversation_id": bson.M{"$in": convIDs},
	}
	cursor, err := m.convMod.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &convs); err != nil {
		return nil, err
	}
	return convs, nil
}

func (m *defaultConversationModel) FindConversation(ctx context.Context, ownerUserID, conversationID string) (*model.Conversation, error) {
	var conv model.Conversation
	filter := bson.M{
		"owner_user_id":   ownerUserID,
		"conversation_id": conversationID,
	}
	result, err := m.convMod.Collection.FindOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	if err := result.Decode(&conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

func (m *defaultConversationModel) FindConversationsByOwner(ctx context.Context, ownerUserID string) ([]*model.Conversation, error) {
	var convs []*model.Conversation
	cursor, err := m.convMod.Collection.Find(ctx, bson.M{"owner_user_id": ownerUserID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &convs); err != nil {
		return nil, err
	}
	return convs, nil
}

func (m *defaultConversationModel) FindPinnedConversationIDs(ctx context.Context, ownerUserID string) ([]string, error) {
	var ids []string
	cursor, err := m.convMod.Collection.Find(ctx, bson.M{"owner_user_id": ownerUserID, "is_pinned": true},
		options.Find().SetProjection(bson.M{"_id": 0, "conversation_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (m *defaultConversationModel) FindConversationsByIDs(ctx context.Context, ownerUserID string, convIDs []string) ([]*model.Conversation, error) {
	var convs []*model.Conversation
	filter := bson.M{
		"owner_user_id":   ownerUserID,
		"conversation_id": bson.M{"$in": convIDs},
	}
	cursor, err := m.convMod.Collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &convs); err != nil {
		return nil, err
	}
	return convs, nil
}

func (m *defaultConversationModel) UpdateConversation(ctx context.Context, owner, convID string, updates map[string]any) error {
	filter := bson.M{
		"owner_user_id":   owner,
		"conversation_id": convID,
	}
	_, err := m.convMod.Collection.UpdateOne(ctx, filter, bson.M{"$set": updates})
	return err
}

func (m *defaultConversationModel) UpdateConversations(ctx context.Context, ownerUserID string, convIDs []string, updates map[string]any) error {
	filter := bson.M{
		"owner_user_id":   ownerUserID,
		"conversation_id": bson.M{"$in": convIDs},
	}
	_, err := m.convMod.Collection.UpdateMany(ctx, filter, bson.M{"$set": updates})
	return err
}

func (m *defaultConversationModel) DeleteConversation(ctx context.Context, owner, convID string) error {
	filter := bson.M{
		"owner_user_id":   owner,
		"conversation_id": convID,
	}
	_, err := m.convMod.Collection.DeleteOne(ctx, filter)
	return err
}

func (m *defaultConversationModel) DeleteConversationsByOwner(ctx context.Context, ownerUserID string) error {
	_, err := m.convMod.Collection.DeleteMany(ctx, bson.M{"owner_user_id": ownerUserID})
	return err
}

func (m *defaultConversationModel) FindNoRecvConversationUserIDs(ctx context.Context, conversationID string) ([]string, error) {
	var filterUserIDs []string
	filter := bson.M{
		"conversation_id": conversationID,
		"recv_msg_opt":    bson.M{"$ne": constant.ReceiveMessage},
	}
	findOpts := options.Find().SetProjection(bson.M{"_id": 0, "owner_user_id": 1})
	cursor, err := m.convMod.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &filterUserIDs); err != nil {
		return nil, err
	}
	return filterUserIDs, nil
}

func (m *defaultConversationModel) FindUserNotNotifyConversationIDs(ctx context.Context, ownerUserID string) ([]string, error) {
	var filterUserIDs []string
	filter := bson.M{
		"owner_user_id": ownerUserID,
		"recv_msg_opt":  constant.ReceiveNotNotifyMessage,
	}
	findOpts := options.Find().SetProjection(bson.M{"_id": 0, "conversation_id": 1})
	cursor, err := m.convMod.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &filterUserIDs); err != nil {
		return nil, err
	}
	return filterUserIDs, nil
}

func (m *defaultConversationModel) UpsertConversationLatestMsg(ctx context.Context, latest *model.ConversationLatestMsg) error {
	filter := bson.M{
		"owner_user_id":   latest.OwnerUserID,
		"conversation_id": latest.ConversationID,
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.latestMod.Collection.UpdateOne(ctx, filter, bson.M{"$set": latest}, opts)
	return err
}

func (m *defaultConversationModel) FindLatestMsg(ctx context.Context, owner, convID string) (*model.ConversationLatestMsg, error) {
	var latest model.ConversationLatestMsg
	filter := bson.M{
		"owner_user_id":   owner,
		"conversation_id": convID,
	}
	result, err := m.latestMod.Collection.FindOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrLatestMsgNotFound
		}
		return nil, err
	}
	if err := result.Decode(&latest); err != nil {
		return nil, err
	}
	return &latest, nil
}

func (m *defaultConversationModel) FindLatestMsgsByOwner(ctx context.Context, owner string, limit int) ([]*model.ConversationLatestMsg, error) {
	var latestMsgs []*model.ConversationLatestMsg
	opts := options.Find().SetSort(bson.M{"latest_msg_recv_time": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := m.latestMod.Collection.Find(ctx, bson.M{"owner_user_id": owner}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &latestMsgs); err != nil {
		return nil, err
	}
	return latestMsgs, nil
}

func (m *defaultConversationModel) DeleteLatestMsg(ctx context.Context, owner, convID string) error {
	filter := bson.M{
		"owner_user_id":   owner,
		"conversation_id": convID,
	}
	_, err := m.latestMod.Collection.DeleteOne(ctx, filter)
	return err
}
