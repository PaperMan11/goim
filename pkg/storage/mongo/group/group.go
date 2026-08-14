package group

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GroupCountByDay 按天统计群组创建数
type GroupCountByDay struct {
	ID    string `bson:"_id"`   // 日期，格式 YYYY-MM-DD
	Count int64  `bson:"count"` // 当天创建数量
}

var (
	ErrGroupNotFound       = errors.New("group not found")
	ErrGroupMemberNotFound = errors.New("group member not found")
)

type GroupModel interface {
	InsertGroup(ctx context.Context, group *model.Group) error
	FindGroup(ctx context.Context, groupID string) (*model.Group, error)
	FindGroupsByIDs(ctx context.Context, groupIDs []string) ([]*model.Group, error)
	UpdateGroup(ctx context.Context, group *model.Group) error
	UpdateGroupEx(ctx context.Context, groupID string, updates map[string]any) error
	DeleteGroup(ctx context.Context, groupID string) error
	CheckGroupExists(ctx context.Context, groupIDs []string) (map[string]bool, error)
	PageGroups(ctx context.Context, page, size int64, groupName string) ([]*model.Group, int64, error)
	CountGroups(ctx context.Context) (int64, error)
	CountGroupsBefore(ctx context.Context, before time.Time) (int64, error)
	CountGroupsByTimeRange(ctx context.Context, start, end time.Time) ([]GroupCountByDay, error)

	InsertMember(ctx context.Context, member *model.GroupMember) error
	InsertMembers(ctx context.Context, members []*model.GroupMember) error
	UpdateMember(ctx context.Context, groupID, userID string, updates map[string]any) error
	UpsertMember(ctx context.Context, member *model.GroupMember) error
	DeleteMember(ctx context.Context, groupID, userID string) error
	DeleteMembers(ctx context.Context, groupID string, userIDs []string) error
	ClearMembers(ctx context.Context, groupID string) error
	FindMember(ctx context.Context, groupID, userID string) (*model.GroupMember, error)
	FindMembersByIDs(ctx context.Context, groupID string, userIDs []string) ([]*model.GroupMember, error)
	FindMembersByGroup(ctx context.Context, groupID string) ([]*model.GroupMember, error)
	FindMemberIDsByGroup(ctx context.Context, groupID string) ([]string, error)
	FindMembersByUser(ctx context.Context, userID string) ([]*model.GroupMember, error)
	FindMembersByRoleLevels(ctx context.Context, groupID string, roles []int32) ([]*model.GroupMember, error)
	CountMembers(ctx context.Context, groupID string) (int64, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
	GetMemberRole(ctx context.Context, groupID, userID string) (int, error)
	IncrMemberCount(ctx context.Context, groupID string, delta int) error
}

type defaultGroupModel struct {
	groupMod  *mon.Model
	memberMod *mon.Model
}

func NewGroupModel(groupMod, memberMod *mon.Model) GroupModel {
	return &defaultGroupModel{
		groupMod:  groupMod,
		memberMod: memberMod,
	}
}

func (m *defaultGroupModel) InsertGroup(ctx context.Context, group *model.Group) error {
	_, err := m.groupMod.Collection.InsertOne(ctx, group)
	return err
}

func (m *defaultGroupModel) FindGroup(ctx context.Context, groupID string) (*model.Group, error) {
	var group model.Group
	result, err := m.groupMod.Collection.FindOne(ctx, bson.M{"group_id": groupID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	if err := result.Decode(&group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (m *defaultGroupModel) FindGroupsByIDs(ctx context.Context, groupIDs []string) ([]*model.Group, error) {
	var groups []*model.Group
	cursor, err := m.groupMod.Collection.Find(ctx, bson.M{"group_id": bson.M{"$in": groupIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (m *defaultGroupModel) UpdateGroup(ctx context.Context, group *model.Group) error {
	_, err := m.groupMod.Collection.UpdateOne(ctx, bson.M{"group_id": group.GroupID}, bson.M{"$set": group})
	return err
}

func (m *defaultGroupModel) UpdateGroupEx(ctx context.Context, groupID string, updates map[string]any) error {
	_, err := m.groupMod.Collection.UpdateOne(ctx, bson.M{"group_id": groupID}, bson.M{"$set": updates})
	return err
}

func (m *defaultGroupModel) DeleteGroup(ctx context.Context, groupID string) error {
	_, err := m.groupMod.Collection.DeleteOne(ctx, bson.M{"group_id": groupID})
	return err
}

func (m *defaultGroupModel) CheckGroupExists(ctx context.Context, groupIDs []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, id := range groupIDs {
		result[id] = false
	}

	var groups []*model.Group
	cursor, err := m.groupMod.Collection.Find(ctx, bson.M{"group_id": bson.M{"$in": groupIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	for _, group := range groups {
		result[group.GroupID] = true
	}

	return result, nil
}

func (m *defaultGroupModel) PageGroups(ctx context.Context, page, size int64, groupName string) ([]*model.Group, int64, error) {
	filter := bson.M{}
	if groupName != "" {
		filter["group_name"] = bson.M{"$regex": groupName, "$options": "i"}
	}

	total, err := m.groupMod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var groups []*model.Group
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"updated_at": -1})
	cursor, err := m.groupMod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

func (m *defaultGroupModel) CountGroups(ctx context.Context) (int64, error) {
	return m.groupMod.Collection.CountDocuments(ctx, bson.M{})
}

func (m *defaultGroupModel) CountGroupsBefore(ctx context.Context, before time.Time) (int64, error) {
	return m.groupMod.Collection.CountDocuments(ctx, bson.M{"create_time": bson.M{"$lt": before}})
}

func (m *defaultGroupModel) CountGroupsByTimeRange(ctx context.Context, start, end time.Time) ([]GroupCountByDay, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"create_time": bson.M{"$gte": start, "$lt": end},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$create_time",
					},
				},
				"count": bson.M{"$sum": 1},
			},
		},
		{"$sort": bson.M{"_id": 1}},
	}
	cursor, err := m.groupMod.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []GroupCountByDay
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (m *defaultGroupModel) InsertMember(ctx context.Context, member *model.GroupMember) error {
	_, err := m.memberMod.Collection.InsertOne(ctx, member)
	return err
}

func (m *defaultGroupModel) InsertMembers(ctx context.Context, members []*model.GroupMember) error {
	var docs []any
	for _, member := range members {
		docs = append(docs, member)
	}
	_, err := m.memberMod.Collection.InsertMany(ctx, docs)
	return err
}

func (m *defaultGroupModel) UpdateMember(ctx context.Context, groupID, userID string, updates map[string]any) error {
	_, err := m.memberMod.Collection.UpdateOne(ctx,
		bson.M{"group_id": groupID, "user_id": userID},
		bson.M{"$set": updates})
	return err
}

func (m *defaultGroupModel) UpsertMember(ctx context.Context, member *model.GroupMember) error {
	opts := options.UpdateOne().SetUpsert(true)
	_, err := m.memberMod.Collection.UpdateOne(ctx,
		bson.M{"group_id": member.GroupID, "user_id": member.UserID},
		bson.M{"$set": member}, opts)
	return err
}

func (m *defaultGroupModel) DeleteMember(ctx context.Context, groupID, userID string) error {
	_, err := m.memberMod.Collection.DeleteOne(ctx,
		bson.M{"group_id": groupID, "user_id": userID})
	return err
}

func (m *defaultGroupModel) DeleteMembers(ctx context.Context, groupID string, userIDs []string) error {
	_, err := m.memberMod.Collection.DeleteMany(ctx,
		bson.M{"group_id": groupID, "user_id": bson.M{"$in": userIDs}})
	return err
}

func (m *defaultGroupModel) ClearMembers(ctx context.Context, groupID string) error {
	_, err := m.memberMod.Collection.DeleteMany(ctx,
		bson.M{"group_id": groupID})
	return err
}

func (m *defaultGroupModel) FindMember(ctx context.Context, groupID, userID string) (*model.GroupMember, error) {
	var member model.GroupMember
	result, err := m.memberMod.Collection.FindOne(ctx,
		bson.M{"group_id": groupID, "user_id": userID})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrGroupMemberNotFound
		}
		return nil, err
	}
	if err := result.Decode(&member); err != nil {
		return nil, err
	}
	return &member, nil
}

func (m *defaultGroupModel) FindMembersByIDs(ctx context.Context, groupID string, userIDs []string) ([]*model.GroupMember, error) {
	members := make([]*model.GroupMember, 0, len(userIDs))
	cursor, err := m.memberMod.Collection.Find(ctx, bson.M{"group_id": groupID, "user_id": bson.M{"$in": userIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (m *defaultGroupModel) FindMembersByGroup(ctx context.Context, groupID string) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	cursor, err := m.memberMod.Collection.Find(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (m *defaultGroupModel) FindMemberIDsByGroup(ctx context.Context, groupID string) ([]string, error) {
	var members []*model.GroupMember
	opts := options.Find().SetProjection(bson.M{"user_id": 1})
	cursor, err := m.memberMod.Collection.Find(ctx, bson.M{"group_id": groupID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids, nil
}

func (m *defaultGroupModel) FindMembersByUser(ctx context.Context, userID string) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	cursor, err := m.memberMod.Collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (m *defaultGroupModel) FindMembersByRoleLevels(ctx context.Context, groupID string, roles []int32) ([]*model.GroupMember, error) {
	var members []*model.GroupMember
	cursor, err := m.memberMod.Collection.Find(ctx, bson.M{"group_id": groupID, "role_level": bson.M{"$in": roles}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (m *defaultGroupModel) CountMembers(ctx context.Context, groupID string) (int64, error) {
	return m.memberMod.Collection.CountDocuments(ctx, bson.M{"group_id": groupID})
}

func (m *defaultGroupModel) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	count, err := m.memberMod.Collection.CountDocuments(ctx,
		bson.M{"group_id": groupID, "user_id": userID})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *defaultGroupModel) GetMemberRole(ctx context.Context, groupID, userID string) (int, error) {
	member, err := m.FindMember(ctx, groupID, userID)
	if err != nil {
		return 0, err
	}
	return member.RoleLevel, nil
}

func (m *defaultGroupModel) IncrMemberCount(ctx context.Context, groupID string, delta int) error {
	_, err := m.groupMod.Collection.UpdateOne(ctx,
		bson.M{"group_id": groupID},
		bson.M{
			"$inc": bson.M{"member_count": delta},
			"$set": bson.M{"updated_at": timex.Now()},
		})
	return err
}
