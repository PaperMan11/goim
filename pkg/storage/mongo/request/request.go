package request

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
	ErrFriendRequestNotFound = errors.New("friend request not found")
	ErrGroupRequestNotFound  = errors.New("group request not found")
)

type RequestModel interface {
	InsertFriendRequest(ctx context.Context, req *model.FriendRequest) error
	UpsertFriendRequest(ctx context.Context, req *model.FriendRequest) error
	HandleFriendRequest(ctx context.Context, fromUserID, toUserID, handlerUserID string, handleResult int, handleMsg string) error
	FindFriendRequest(ctx context.Context, from, to string) (*model.FriendRequest, error)
	FindFriendRequestsByFrom(ctx context.Context, from string, page, size int64) ([]*model.FriendRequest, int64, error)
	FindFriendRequestsByTo(ctx context.Context, to string, page, size int64) ([]*model.FriendRequest, int64, error)
	DeleteFriendRequest(ctx context.Context, from, to string) error

	InsertGroupRequest(ctx context.Context, req *model.GroupRequest) error
	UpsertGroupRequest(ctx context.Context, req *model.GroupRequest) error
	HandleGroupRequest(ctx context.Context, userID, groupID, handleUserID string, handleResult int, handleMsg string) error
	FindGroupRequest(ctx context.Context, userID, groupID string) (*model.GroupRequest, error)
	FindGroupRequestsByUser(ctx context.Context, userID string, page, size int64) ([]*model.GroupRequest, int64, error)
	FindGroupRequestsByGroup(ctx context.Context, groupID string, page, size int64) ([]*model.GroupRequest, int64, error)
	DeleteGroupRequest(ctx context.Context, userID, groupID string) error
}

type defaultRequestModel struct {
	friendReqMod *mon.Model
	groupReqMod  *mon.Model
}

func NewRequestModel(friendReqMod, groupReqMod *mon.Model) RequestModel {
	return &defaultRequestModel{
		friendReqMod: friendReqMod,
		groupReqMod:  groupReqMod,
	}
}

func (m *defaultRequestModel) InsertFriendRequest(ctx context.Context, req *model.FriendRequest) error {
	_, err := m.friendReqMod.Collection.InsertOne(ctx, req)
	return err
}

func (m *defaultRequestModel) UpsertFriendRequest(ctx context.Context, req *model.FriendRequest) error {
	filter := bson.M{"from_user_id": req.FromUserID, "to_user_id": req.ToUserID}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.friendReqMod.Collection.UpdateOne(ctx, filter, bson.M{"$set": req}, opts)
	return err
}

func (m *defaultRequestModel) HandleFriendRequest(ctx context.Context, fromUserID, toUserID, handlerUserID string, handleResult int, handleMsg string) error {
	filter := bson.M{"from_user_id": fromUserID, "to_user_id": toUserID}
	update := bson.M{
		"$set": bson.M{
			"handle_result":   handleResult,
			"handler_user_id": handlerUserID,
			"handle_msg":      handleMsg,
			"handle_time":     timex.Now(),
		},
	}
	_, err := m.friendReqMod.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (m *defaultRequestModel) FindFriendRequest(ctx context.Context, from, to string) (*model.FriendRequest, error) {
	var req model.FriendRequest
	filter := bson.M{"from_user_id": from, "to_user_id": to}
	result, err := m.friendReqMod.Collection.FindOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrFriendRequestNotFound
		}
		return nil, err
	}
	if err := result.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (m *defaultRequestModel) FindFriendRequestsByFrom(ctx context.Context, from string, page, size int64) ([]*model.FriendRequest, int64, error) {
	filter := bson.M{"from_user_id": from}
	total, err := m.friendReqMod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var requests []*model.FriendRequest
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"create_time": -1})
	cursor, err := m.friendReqMod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func (m *defaultRequestModel) FindFriendRequestsByTo(ctx context.Context, to string, page, size int64) ([]*model.FriendRequest, int64, error) {
	filter := bson.M{"to_user_id": to}
	total, err := m.friendReqMod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var requests []*model.FriendRequest
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"create_time": -1})
	cursor, err := m.friendReqMod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func (m *defaultRequestModel) DeleteFriendRequest(ctx context.Context, from, to string) error {
	filter := bson.M{"from_user_id": from, "to_user_id": to}
	_, err := m.friendReqMod.Collection.DeleteOne(ctx, filter)
	return err
}

func (m *defaultRequestModel) InsertGroupRequest(ctx context.Context, req *model.GroupRequest) error {
	_, err := m.groupReqMod.Collection.InsertOne(ctx, req)
	return err
}

func (m *defaultRequestModel) UpsertGroupRequest(ctx context.Context, req *model.GroupRequest) error {
	filter := bson.M{"user_id": req.UserID, "group_id": req.GroupID}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.groupReqMod.Collection.UpdateOne(ctx, filter, bson.M{"$set": req}, opts)
	return err
}

func (m *defaultRequestModel) HandleGroupRequest(ctx context.Context, userID, groupID, handleUserID string, handleResult int, handleMsg string) error {
	filter := bson.M{"user_id": userID, "group_id": groupID}
	update := bson.M{
		"$set": bson.M{
			"handle_result":  handleResult,
			"handle_user_id": handleUserID,
			"handle_msg":     handleMsg,
			"handle_time":    timex.Now(),
		},
	}
	_, err := m.groupReqMod.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (m *defaultRequestModel) FindGroupRequest(ctx context.Context, userID, groupID string) (*model.GroupRequest, error) {
	var req model.GroupRequest
	filter := bson.M{"user_id": userID, "group_id": groupID}
	result, err := m.groupReqMod.Collection.FindOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrGroupRequestNotFound
		}
		return nil, err
	}
	if err := result.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (m *defaultRequestModel) FindGroupRequestsByUser(ctx context.Context, userID string, page, size int64) ([]*model.GroupRequest, int64, error) {
	filter := bson.M{"user_id": userID}
	total, err := m.groupReqMod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var requests []*model.GroupRequest
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"req_time": -1})
	cursor, err := m.groupReqMod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func (m *defaultRequestModel) FindGroupRequestsByGroup(ctx context.Context, groupID string, page, size int64) ([]*model.GroupRequest, int64, error) {
	filter := bson.M{"group_id": groupID}
	total, err := m.groupReqMod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var requests []*model.GroupRequest
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"req_time": -1})
	cursor, err := m.groupReqMod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

func (m *defaultRequestModel) DeleteGroupRequest(ctx context.Context, userID, groupID string) error {
	filter := bson.M{"user_id": userID, "group_id": groupID}
	_, err := m.groupReqMod.Collection.DeleteOne(ctx, filter)
	return err
}
