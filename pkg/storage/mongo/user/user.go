package user

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrCommandNotFound      = errors.New("command not found")
	ErrNotificationNotFound = errors.New("notification account not found")
)

type UserModel interface {
	Insert(ctx context.Context, users []*model.User) error
	FindByIDs(ctx context.Context, userIDs []string) ([]*model.User, error)
	FindByID(ctx context.Context, userID string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	UpdateEx(ctx context.Context, userID string, updates map[string]any) error
	Delete(ctx context.Context, userID string) error
	Count(ctx context.Context) (int64, error)
	Page(ctx context.Context, page, size int64, userID, nickname string) ([]*model.User, int64, error)
	CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error)
	GetAllUserIDs(ctx context.Context, page, size int64) ([]string, int64, error)
	SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error
	GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error)
	RegisterCount(ctx context.Context, start, end int64) (int64, int64, map[string]int64, error)

	InsertUserStatus(ctx context.Context, status *model.UserStatus) error
	UpdateUserStatus(ctx context.Context, userID string, platformID int, status int) error
	GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error)
	SetUserOnlineStatus(ctx context.Context, statuses []*model.UserStatus) error
	GetAllOnlineUsers(ctx context.Context) ([]string, error)

	InsertUserCommand(ctx context.Context, cmd *model.UserCommand) error
	UpdateUserCommand(ctx context.Context, userID, uuid string, value string) error
	DeleteUserCommand(ctx context.Context, userID, uuid string) error
	GetUserCommand(ctx context.Context, userID, uuid string) (*model.UserCommand, error)
	GetAllUserCommands(ctx context.Context, userID string) ([]*model.UserCommand, error)

	IsIMAdmin(ctx context.Context, userID string) (bool, error)
}

type defaultUserModel struct {
	mod *mon.Model
}

func NewUserModel(mod *mon.Model) UserModel {
	return &defaultUserModel{
		mod: mod,
	}
}

func (m *defaultUserModel) Insert(ctx context.Context, users []*model.User) error {
	var docs []any
	for _, user := range users {
		docs = append(docs, user)
	}
	_, err := m.mod.Collection.InsertMany(ctx, docs)
	return err
}

func (m *defaultUserModel) FindByIDs(ctx context.Context, userIDs []string) ([]*model.User, error) {
	var users []*model.User
	cursor, err := m.mod.Collection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (m *defaultUserModel) FindByID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	result, err := m.mod.Collection.FindOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if err := result.Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *defaultUserModel) Update(ctx context.Context, user *model.User) error {
	_, err := m.mod.Collection.UpdateOne(ctx, bson.M{"user_id": user.UserID}, bson.M{"$set": user})
	return err
}

func (m *defaultUserModel) UpdateEx(ctx context.Context, userID string, updates map[string]any) error {
	_, err := m.mod.Collection.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": updates})
	return err
}

func (m *defaultUserModel) Delete(ctx context.Context, userID string) error {
	_, err := m.mod.Collection.DeleteOne(ctx, bson.M{"user_id": userID})
	return err
}

func (m *defaultUserModel) Count(ctx context.Context) (int64, error) {
	return m.mod.Collection.CountDocuments(ctx, bson.M{})
}

func (m *defaultUserModel) Page(ctx context.Context, page, size int64, userID, nickname string) ([]*model.User, int64, error) {
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	if nickname != "" {
		filter["nickname"] = bson.M{"$regex": nickname, "$options": "i"}
	}

	total, err := m.mod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"created_at": -1})
	cursor, err := m.mod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (m *defaultUserModel) CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range userIDs {
		result[id] = false
	}

	var users []*model.User
	cursor, err := m.mod.Collection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	for _, user := range users {
		result[user.UserID] = true
	}

	return result, nil
}

func (m *defaultUserModel) GetAllUserIDs(ctx context.Context, page, size int64) ([]string, int64, error) {
	total, err := m.mod.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetProjection(bson.M{"user_id": 1})
	cursor, err := m.mod.Collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.UserID)
	}

	return ids, total, nil
}

func (m *defaultUserModel) SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error {
	_, err := m.mod.Collection.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": bson.M{"global_recv_msg_opt": opt}})
	return err
}

func (m *defaultUserModel) GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error) {
	var user model.User
	result, err := m.mod.Collection.FindOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	if err := result.Decode(&user); err != nil {
		return 0, err
	}
	return user.GlobalRecvMsgOpt, nil
}

func (m *defaultUserModel) RegisterCount(ctx context.Context, start, end int64) (int64, int64, map[string]int64, error) {
	total, err := m.mod.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, 0, nil, err
	}

	before, err := m.mod.Collection.CountDocuments(ctx, bson.M{"created_at": bson.M{"$lt": time.Unix(start, 0)}})
	if err != nil {
		return 0, 0, nil, err
	}

	count := make(map[string]int64)
	return total, before, count, nil
}

func (m *defaultUserModel) InsertUserStatus(ctx context.Context, status *model.UserStatus) error {
	_, err := m.mod.Collection.InsertOne(ctx, status)
	return err
}

func (m *defaultUserModel) UpdateUserStatus(ctx context.Context, userID string, platformID int, status int) error {
	_, err := m.mod.Collection.UpdateOne(ctx,
		bson.M{"user_id": userID, "platform_id": platformID},
		bson.M{"$set": bson.M{"status": status}})
	return err
}

func (m *defaultUserModel) GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error) {
	var statuses []*model.UserStatus
	cursor, err := m.mod.Collection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (m *defaultUserModel) SetUserOnlineStatus(ctx context.Context, statuses []*model.UserStatus) error {
	var docs []any
	for _, status := range statuses {
		docs = append(docs, status)
	}
	_, err := m.mod.Collection.InsertMany(ctx, docs)
	return err
}

func (m *defaultUserModel) GetAllOnlineUsers(ctx context.Context) ([]string, error) {
	var statuses []*model.UserStatus
	cursor, err := m.mod.Collection.Find(ctx, bson.M{"status": 1})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &statuses); err != nil {
		return nil, err
	}

	userIDs := make([]string, 0, len(statuses))
	seen := make(map[string]bool)
	for _, status := range statuses {
		if !seen[status.UserID] {
			seen[status.UserID] = true
			userIDs = append(userIDs, status.UserID)
		}
	}

	return userIDs, nil
}

func (m *defaultUserModel) InsertUserCommand(ctx context.Context, cmd *model.UserCommand) error {
	_, err := m.mod.Collection.InsertOne(ctx, cmd)
	return err
}

func (m *defaultUserModel) UpdateUserCommand(ctx context.Context, userID, uuid string, value string) error {
	_, err := m.mod.Collection.UpdateOne(ctx,
		bson.M{"user_id": userID, "uuid": uuid},
		bson.M{"$set": bson.M{"value": value}})
	return err
}

func (m *defaultUserModel) DeleteUserCommand(ctx context.Context, userID, uuid string) error {
	_, err := m.mod.Collection.DeleteOne(ctx, bson.M{"user_id": userID, "uuid": uuid})
	return err
}

func (m *defaultUserModel) GetUserCommand(ctx context.Context, userID, uuid string) (*model.UserCommand, error) {
	var cmd model.UserCommand
	result, err := m.mod.Collection.FindOne(ctx, bson.M{"user_id": userID, "uuid": uuid})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrCommandNotFound
		}
		return nil, err
	}
	if err := result.Decode(&cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (m *defaultUserModel) GetAllUserCommands(ctx context.Context, userID string) ([]*model.UserCommand, error) {
	var cmds []*model.UserCommand
	cursor, err := m.mod.Collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &cmds); err != nil {
		return nil, err
	}
	return cmds, nil
}

func (m *defaultUserModel) IsIMAdmin(ctx context.Context, userID string) (bool, error) {
	var user model.User
	result, err := m.mod.Collection.FindOne(ctx, bson.M{"user_id": userID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, ErrUserNotFound
		}
		return false, err
	}
	if err := result.Decode(&user); err != nil {
		return false, err
	}
	return user.AppManagerLevel > 0, nil
}
