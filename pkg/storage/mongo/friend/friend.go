package friend

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
	ErrFriendNotFound = errors.New("friend not found")
	ErrBlackNotFound  = errors.New("black not found")
)

type FriendModel interface {
	InsertFriend(ctx context.Context, friend *model.Friend) error
	InsertFriends(ctx context.Context, friends []*model.Friend) error
	UpdateFriend(ctx context.Context, owner, friendUserID string, updates map[string]any) error
	DeleteFriend(ctx context.Context, owner, friendUserID string) error
	DeleteFriendBoth(ctx context.Context, userA, userB string) error
	FindFriend(ctx context.Context, owner, friendUserID string) (*model.Friend, error)
	FindFriendsByOwner(ctx context.Context, owner string) ([]*model.Friend, error)
	FindFriendsByIDs(ctx context.Context, owner string, friendIDs []string) ([]*model.Friend, error)
	IsFriend(ctx context.Context, userA, userB string) (bool, error)
	CountFriends(ctx context.Context, owner string) (int64, error)

	InsertBlack(ctx context.Context, black *model.Black) error
	DeleteBlack(ctx context.Context, owner, blackUserID string) error
	FindBlack(ctx context.Context, owner, blackUserID string) (*model.Black, error)
	FindBlacksByOwner(ctx context.Context, owner string) ([]*model.Black, error)
	IsBlack(ctx context.Context, owner, targetUserID string) (bool, error)
}

type defaultFriendModel struct {
	friendMod *mon.Model
	blackMod  *mon.Model
}

func NewFriendModel(friendMod, blackMod *mon.Model) FriendModel {
	return &defaultFriendModel{
		friendMod: friendMod,
		blackMod:  blackMod,
	}
}

func (m *defaultFriendModel) InsertFriend(ctx context.Context, friend *model.Friend) error {
	_, err := m.friendMod.Collection.InsertOne(ctx, friend)
	return err
}

func (m *defaultFriendModel) InsertFriends(ctx context.Context, friends []*model.Friend) error {
	var docs []any
	for _, f := range friends {
		docs = append(docs, f)
	}
	_, err := m.friendMod.Collection.InsertMany(ctx, docs)
	return err
}

func (m *defaultFriendModel) UpdateFriend(ctx context.Context, owner, friendUserID string, updates map[string]any) error {
	updates["updated_at"] = timex.Now()
	_, err := m.friendMod.Collection.UpdateOne(ctx,
		bson.M{"owner_user_id": owner, "friend_user_id": friendUserID},
		bson.M{"$set": updates})
	return err
}

func (m *defaultFriendModel) DeleteFriend(ctx context.Context, owner, friendUserID string) error {
	_, err := m.friendMod.Collection.DeleteOne(ctx,
		bson.M{"owner_user_id": owner, "friend_user_id": friendUserID})
	return err
}

func (m *defaultFriendModel) DeleteFriendBoth(ctx context.Context, userA, userB string) error {
	_, err := m.friendMod.Collection.DeleteOne(ctx,
		bson.M{"owner_user_id": userA, "friend_user_id": userB})
	if err != nil {
		return err
	}
	_, err = m.friendMod.Collection.DeleteOne(ctx,
		bson.M{"owner_user_id": userB, "friend_user_id": userA})
	return err
}

func (m *defaultFriendModel) FindFriend(ctx context.Context, owner, friendUserID string) (*model.Friend, error) {
	var friend model.Friend
	result, err := m.friendMod.Collection.FindOne(ctx,
		bson.M{"owner_user_id": owner, "friend_user_id": friendUserID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrFriendNotFound
		}
		return nil, err
	}
	if err := result.Decode(&friend); err != nil {
		return nil, err
	}
	return &friend, nil
}

func (m *defaultFriendModel) FindFriendsByOwner(ctx context.Context, owner string) ([]*model.Friend, error) {
	var friends []*model.Friend
	cursor, err := m.friendMod.Collection.Find(ctx, bson.M{"owner_user_id": owner})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &friends); err != nil {
		return nil, err
	}
	return friends, nil
}

func (m *defaultFriendModel) FindFriendsByIDs(ctx context.Context, owner string, friendIDs []string) ([]*model.Friend, error) {
	var friends []*model.Friend
	cursor, err := m.friendMod.Collection.Find(ctx,
		bson.M{"owner_user_id": owner, "friend_user_id": bson.M{"$in": friendIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &friends); err != nil {
		return nil, err
	}
	return friends, nil
}

func (m *defaultFriendModel) IsFriend(ctx context.Context, userA, userB string) (bool, error) {
	countA, err := m.friendMod.Collection.CountDocuments(ctx,
		bson.M{"owner_user_id": userA, "friend_user_id": userB})
	if err != nil {
		return false, err
	}
	if countA == 0 {
		return false, nil
	}
	countB, err := m.friendMod.Collection.CountDocuments(ctx,
		bson.M{"owner_user_id": userB, "friend_user_id": userA})
	if err != nil {
		return false, err
	}
	return countB > 0, nil
}

func (m *defaultFriendModel) CountFriends(ctx context.Context, owner string) (int64, error) {
	return m.friendMod.Collection.CountDocuments(ctx, bson.M{"owner_user_id": owner})
}

func (m *defaultFriendModel) InsertBlack(ctx context.Context, black *model.Black) error {
	_, err := m.blackMod.Collection.InsertOne(ctx, black)
	return err
}

func (m *defaultFriendModel) DeleteBlack(ctx context.Context, owner, blackUserID string) error {
	_, err := m.blackMod.Collection.DeleteOne(ctx,
		bson.M{"owner_user_id": owner, "black_user_id": blackUserID})
	return err
}

func (m *defaultFriendModel) FindBlack(ctx context.Context, owner, blackUserID string) (*model.Black, error) {
	var black model.Black
	result, err := m.blackMod.Collection.FindOne(ctx,
		bson.M{"owner_user_id": owner, "black_user_id": blackUserID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrBlackNotFound
		}
		return nil, err
	}
	if err := result.Decode(&black); err != nil {
		return nil, err
	}
	return &black, nil
}

func (m *defaultFriendModel) FindBlacksByOwner(ctx context.Context, owner string) ([]*model.Black, error) {
	var blacks []*model.Black
	cursor, err := m.blackMod.Collection.Find(ctx, bson.M{"owner_user_id": owner})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &blacks); err != nil {
		return nil, err
	}
	return blacks, nil
}

func (m *defaultFriendModel) IsBlack(ctx context.Context, owner, targetUserID string) (bool, error) {
	count, err := m.blackMod.Collection.CountDocuments(ctx,
		bson.M{"owner_user_id": owner, "black_user_id": targetUserID})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
