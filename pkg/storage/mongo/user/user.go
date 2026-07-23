package user

import (
	"context"
	"errors"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/syncx"
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
	SortQuery(ctx context.Context, userIDName map[string]string, asc bool) ([]*model.User, error)
	CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error)
	GetAllUserIDs(ctx context.Context, page, size int64) ([]string, int64, error)
	SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error
	GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error)
	RegisterCount(ctx context.Context, start, end int64) (int64, int64, map[string]int64, error)

	InsertUserStatus(ctx context.Context, status *model.UserStatus) error
	UpdateUserStatus(ctx context.Context, userID string, platformID int, deviceID string, status int) error
	GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error)
	// SetUserOnlineStatus 批量 upsert 用户在线状态快照。
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
	userMod   *mon.Model
	statusMod *mon.Model
	cmdMod    *mon.Model
	barrier   syncx.SingleFlight
}

func NewUserModel(userMod, statusMod, cmdMod *mon.Model, barrier syncx.SingleFlight) UserModel {
	m := &defaultUserModel{
		userMod:   userMod,
		statusMod: statusMod,
		cmdMod:    cmdMod,
		barrier:   barrier,
	}
	_ = m.ensureUserStatusIndexes(context.Background())
	return m
}

// =====================================================
// 索引：幂等创建必需的唯一索引 / 查询索引
// =====================================================

// ensureUserStatusIndexes 为 im_user_status 集合创建必需的唯一索引与查询索引。
//
// 启动时幂等：先 drop 旧 P0 索引（不存在就跳过），再创建 P1 新索引；
// 结构冲突（如旧数据里唯一键字段不同导致创建失败）会返回错误，上层吞掉日志告警。
func (m *defaultUserModel) ensureUserStatusIndexes(ctx context.Context) error {
	uniq := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "platform_id", Value: 1},
			{Key: "device_id", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("uniq_user_platform_device"),
	}
	if _, err := m.statusMod.Collection.Indexes().CreateOne(ctx, uniq); err != nil {
		return err
	}

	ordinary := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_user_id"),
		},
		{
			Keys:    bson.D{{Key: "device_id", Value: 1}},
			Options: options.Index().SetName("idx_device_id"),
		},
	}
	if _, err := m.statusMod.Collection.Indexes().CreateMany(ctx, ordinary); err != nil {
		return err
	}

	ttl := mongo.IndexModel{
		Keys:    bson.D{{Key: "expire_at", Value: 1}},
		Options: options.Index().SetName("idx_expire_at_ttl").SetExpireAfterSeconds(0),
	}
	_, _ = m.statusMod.Collection.Indexes().CreateOne(ctx, ttl)
	return nil
}

// =====================================================
// 用户资料（im_users）
// =====================================================

func (m *defaultUserModel) Insert(ctx context.Context, users []*model.User) error {
	docs := make([]any, 0, len(users))
	for _, u := range users {
		docs = append(docs, u)
	}
	_, err := m.userMod.Collection.InsertMany(ctx, docs)
	return err
}

func (m *defaultUserModel) FindByIDs(ctx context.Context, userIDs []string) ([]*model.User, error) {
	var users []*model.User
	cursor, err := m.userMod.Collection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}})
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
	if userID == "" {
		return nil, ErrUserNotFound
	}
	sfKey := "user:find:id:" + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var user model.User
		result, err := m.userMod.Collection.FindOne(ctx, bson.M{"user_id": userID})
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
	})
	if err != nil {
		return nil, err
	}
	return v.(*model.User), nil
}

func (m *defaultUserModel) Update(ctx context.Context, user *model.User) error {
	_, err := m.userMod.Collection.UpdateOne(ctx, bson.M{"user_id": user.UserID}, bson.M{"$set": user})
	return err
}

func (m *defaultUserModel) UpdateEx(ctx context.Context, userID string, updates map[string]any) error {
	_, err := m.userMod.Collection.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": updates})
	return err
}

func (m *defaultUserModel) Delete(ctx context.Context, userID string) error {
	_, err := m.userMod.Collection.DeleteOne(ctx, bson.M{"user_id": userID})
	return err
}

func (m *defaultUserModel) Count(ctx context.Context) (int64, error) {
	return m.userMod.Collection.CountDocuments(ctx, bson.M{})
}

func (m *defaultUserModel) Page(ctx context.Context, page, size int64, userID, nickname string) ([]*model.User, int64, error) {
	// $options:
	//   i = case Insensitive（忽略大小写）：nickname: "Alice" 搜索 "ali" 也能命中
	//   其它常用 flag：m=multiline、x=extended、s=dotall
	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	if nickname != "" {
		filter["nickname"] = bson.M{"$regex": nickname, "$options": "i"}
	}

	total, err := m.userMod.Collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetSort(bson.M{"created_at": -1})
	cursor, err := m.userMod.Collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// SortQuery 按「自定义覆盖名优先，库内 nickname 兜底」规则对一批用户排序后返回。
//
// 两条执行路径（按数据量和场景选择最优）：
//
//	A. 没有任何 userID 配置覆盖名 → 直接走普通 Find + 索引排序，性能最好。
//	B. 存在覆盖名 → 走 MongoDB 聚合管道，在 DB 侧计算排序键，避免 Go 端加载全部数据。
func (m *defaultUserModel) SortQuery(ctx context.Context, userIDName map[string]string, asc bool) ([]*model.User, error) {
	// 1) 空输入直接返回，避免后续无意义的查询。
	if len(userIDName) == 0 {
		return nil, nil
	}

	// 2) 预处理：拆分「要查的 userID 集合」与「真正有覆盖名的 userID 子集」。
	//    覆盖名为空串的 userID 直接使用库里 nickname 参与排序，因此不需要放入 attached。
	userIDs := make([]string, 0, len(userIDName))
	attached := make(map[string]string, len(userIDName))
	for userID, customName := range userIDName {
		userIDs = append(userIDs, userID)
		if customName != "" {
			attached[userID] = customName
		}
	}

	// 3) 排序方向映射：asc=true→1(升序 A→Z)，asc=false→-1(倒序 Z→A)。
	sortDir := -1
	if asc {
		sortDir = 1
	}

	// ============================================
	// 快路径 A：所有 userID 都没配置覆盖名。
	// 直接使用 nickname 字段排序，可命中索引，性能最佳。
	// ============================================
	if len(attached) == 0 {
		filter := bson.M{"user_id": bson.M{"$in": userIDs}}
		opts := options.Find().SetSort(bson.M{"nickname": sortDir})

		cursor, err := m.userMod.Collection.Find(ctx, filter, opts)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		var users []*model.User
		if err := cursor.All(ctx, &users); err != nil {
			return nil, err
		}
		return users, nil
	}

	// ============================================
	// 慢路径 B：存在自定义覆盖名。
	// 用聚合管道给每条文档临时生成一个排序键 _query_sort_name，再按它排序。
	//
	// 整体思路：
	//   ① 先按 user_id 过滤（$match），只处理我们关心的用户集合。
	//   ② 把 attached 这个 Go 端的 map 转成 Mongo 能按条件查询的 [{k,v}] 数组，
	//      再从里面挑出「k == 当前文档 user_id」的那一项，得到 {k,v} 对象（可能为 null）。
	//   ③ 取上面对象的 .v，若为 null（表示该 userID 没配覆盖名）则 fallback 到 $nickname。
	//   ④ 按最终得到的字符串排序。
	// ============================================

	// Stage 1：缩小扫描范围。
	stageMatch := bson.M{
		"$match": bson.M{
			"user_id": bson.M{"$in": userIDs},
		},
	}

	// Stage 2：从 attached（常量 map）里捞出当前文档对应的「覆盖名条目」{k,v}。
	//
	// 关键点：MongoDB 不允许直接写 `attached["$user_id"]` 这种「按文档字段动态取常量 map 的 key」，
	// 所以我们用「对象转数组 → 按条件过滤数组 → 取数组首项」的通用组合拳实现：
	//   $objectToArray : {u1:"小明", u3:"阿强"}  =>  [{k:"u1",v:"小明"}, {k:"u3",v:"阿强"}]
	//   $filter         : 只留下 k == 当前文档 user_id 的那一项（数组，长度 0 或 1）
	//   $arrayElemAt [, 0] : 取数组第 0 项，找不到就是 null
	stagePickOverrideEntry := bson.M{
		"$addFields": bson.M{
			"_query_sort_name": bson.M{
				"$arrayElemAt": []any{
					bson.M{
						"$filter": bson.M{
							"input": bson.M{"$objectToArray": attached},
							"as":    "item",
							"cond":  bson.M{"$eq": []any{"$$item.k", "$user_id"}},
						},
					},
					0,
				},
			},
		},
	}

	// Stage 3：把上一步得到的 {k,v} 对象变成「最终用来排序的字符串」。
	//   - 配了覆盖名的 userID  → 取 $_query_sort_name.v（比如 "小明"）
	//   - 没配覆盖名（null）   → $ifNull 分支落到 $nickname（库内原始昵称）
	stageFinalSortKey := bson.M{
		"$addFields": bson.M{
			"_query_sort_name": bson.M{
				"$ifNull": []any{"$_query_sort_name.v", "$nickname"},
			},
		},
	}

	// Stage 4：按最终排序键输出有序结果。
	stageSort := bson.M{
		"$sort": bson.M{
			"_query_sort_name": sortDir,
		},
	}

	// 用 []bson.M 而不是 mongo.Pipeline（后者是 []bson.D），这样每个 stage 可以用更直观的 map 写法。
	pipeline := []bson.M{stageMatch, stagePickOverrideEntry, stageFinalSortKey, stageSort}
	cursor, err := m.userMod.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (m *defaultUserModel) CheckExists(ctx context.Context, userIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}

	var users []*model.User
	cursor, err := m.userMod.Collection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	for _, u := range users {
		result[u.UserID] = true
	}
	return result, nil
}

func (m *defaultUserModel) GetAllUserIDs(ctx context.Context, page, size int64) ([]string, int64, error) {
	total, err := m.userMod.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	opts := options.Find().SetSkip((page - 1) * size).SetLimit(size).SetProjection(bson.M{"_id": 0, "user_id": 1})
	cursor, err := m.userMod.Collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.UserID)
	}
	return ids, total, nil
}

func (m *defaultUserModel) SetGlobalRecvMsgOpt(ctx context.Context, userID string, opt int) error {
	_, err := m.userMod.Collection.UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"global_recv_msg_opt": opt, "updated_at": time.Now()}})
	return err
}

func (m *defaultUserModel) GetGlobalRecvMsgOpt(ctx context.Context, userID string) (int, error) {
	if userID == "" {
		return 0, ErrUserNotFound
	}
	sfKey := "user:recvopt:" + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var user model.User
		result, err := m.userMod.Collection.FindOne(ctx, bson.M{"user_id": userID})
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
	})
	if err != nil {
		return 0, err
	}
	return v.(int), nil
}

func (m *defaultUserModel) RegisterCount(ctx context.Context, start, end int64) (int64, int64, map[string]int64, error) {
	total, err := m.userMod.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, 0, nil, err
	}
	before, err := m.userMod.Collection.CountDocuments(ctx, bson.M{"created_at": bson.M{"$lt": time.Unix(start, 0)}})
	if err != nil {
		return 0, 0, nil, err
	}
	count := make(map[string]int64)
	return total, before, count, nil
}

// =====================================================
// 在线状态快照（im_user_status）
// =====================================================

func (m *defaultUserModel) InsertUserStatus(ctx context.Context, status *model.UserStatus) error {
	_, err := m.statusMod.Collection.InsertOne(ctx, status)
	return err
}

// UpdateUserStatus 设置某 user+platform(+device) 的在线/离线状态。
func (m *defaultUserModel) UpdateUserStatus(ctx context.Context, userID string, platformID int, deviceID string, status int) error {
	now := time.Now()

	filter := bson.M{"user_id": userID, "platform_id": platformID}
	if deviceID != "" {
		filter["device_id"] = deviceID
	}

	update := bson.M{
		"$set": bson.M{
			"status":           status,
			"last_online_time": now,
			"last_seen_at":     now,
			"updated_at":       now,
		},
	}

	// 只有传了 deviceID 才走 Upsert（精确匹配唯一键一行）；
	// deviceID 为空是「批量更新该平台所有设备」，不应该 Upsert 出一条 deviceID="" 的脏行。
	if deviceID != "" {
		update["$setOnInsert"] = bson.M{
			"created_at":  now,
			"device_id":   deviceID,
			"user_id":     userID,
			"platform_id": platformID,
		}
		_, err := m.statusMod.Collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
		return err
	}

	_, err := m.statusMod.Collection.UpdateMany(ctx, filter, update)
	return err
}

func (m *defaultUserModel) GetUserStatus(ctx context.Context, userIDs []string) ([]*model.UserStatus, error) {
	var statuses []*model.UserStatus
	cursor, err := m.statusMod.Collection.Find(ctx, bson.M{"user_id": bson.M{"$in": userIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}

// SetUserOnlineStatus 批量写入用户在线状态快照。
func (m *defaultUserModel) SetUserOnlineStatus(ctx context.Context, statuses []*model.UserStatus) error {
	if len(statuses) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(statuses))
	for _, s := range statuses {
		filter := bson.M{
			"user_id":     s.UserID,
			"platform_id": s.PlatformID,
		}
		if s.DeviceID != "" {
			filter["device_id"] = s.DeviceID
		}

		setDoc := bson.M{
			"status":           s.Status,
			"last_online_time": s.LastOnlineTime,
			"last_seen_at":     s.UpdatedAt,
			"updated_at":       s.UpdatedAt,
		}
		// ConnID 每次连接都不一样，所以放到 $set 里（同 device 重连时刷新）
		if s.ConnID != "" {
			setDoc["conn_id"] = s.ConnID
		}

		setOnInsert := bson.M{
			"created_at":  s.CreatedAt,
			"user_id":     s.UserID,
			"platform_id": s.PlatformID,
		}
		// 设备身份信息只在首次插入时固定，后续不覆盖
		if s.DeviceID != "" {
			setOnInsert["device_id"] = s.DeviceID
		}
		if s.TokenUUID != "" {
			setOnInsert["token_uuid"] = s.TokenUUID
		}
		if s.DeviceName != "" {
			setOnInsert["device_name"] = s.DeviceName
		}
		if s.LoginIP != "" {
			setOnInsert["login_ip"] = s.LoginIP
		}
		if s.Extra != "" {
			setOnInsert["extra"] = s.Extra
		}
		// ExpireAt：首次插入时设置一个过期时间点（心跳续期会另外刷新）
		if !s.ExpireAt.IsZero() {
			setDoc["expire_at"] = s.ExpireAt
		}

		update := bson.M{"$set": setDoc, "$setOnInsert": setOnInsert}

		models = append(models,
			mongo.NewUpdateOneModel().
				SetFilter(filter).
				SetUpdate(update).
				SetUpsert(true),
		)
	}

	// ordered=false 让互不冲突的多条写入可以并发执行；
	// 同一批内若有重复 key（虽然我们用了 unique 索引理论上不会冲突）也不影响最终结果。
	_, err := m.statusMod.Collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}

// GetAllOnlineUsers 返回当前所有「在线」用户 ID（去重后）。
//
// 注意：这里只看 status==1 的快照行。如果服务进程异常崩溃导致状态行未及时清理，
// 引入定时任务 TTL/心跳机制自动清理僵尸「假在线」记录。
func (m *defaultUserModel) GetAllOnlineUsers(ctx context.Context) ([]string, error) {
	var statuses []*model.UserStatus
	cursor, err := m.statusMod.Collection.Find(ctx, bson.M{"status": 1})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &statuses); err != nil {
		return nil, err
	}

	userIDs := make([]string, 0, len(statuses))
	seen := make(map[string]struct{}, len(statuses))
	for _, s := range statuses {
		if _, ok := seen[s.UserID]; ok {
			continue
		}
		seen[s.UserID] = struct{}{}
		userIDs = append(userIDs, s.UserID)
	}
	return userIDs, nil
}

// =====================================================
// 用户自定义命令（im_user_commands）
// =====================================================

func (m *defaultUserModel) InsertUserCommand(ctx context.Context, cmd *model.UserCommand) error {
	_, err := m.cmdMod.Collection.InsertOne(ctx, cmd)
	return err
}

func (m *defaultUserModel) UpdateUserCommand(ctx context.Context, userID, uuid string, value string) error {
	_, err := m.cmdMod.Collection.UpdateOne(ctx,
		bson.M{"user_id": userID, "uuid": uuid},
		bson.M{"$set": bson.M{"value": value, "updated_at": time.Now()}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (m *defaultUserModel) DeleteUserCommand(ctx context.Context, userID, uuid string) error {
	_, err := m.cmdMod.Collection.DeleteOne(ctx, bson.M{"user_id": userID, "uuid": uuid})
	return err
}

func (m *defaultUserModel) GetUserCommand(ctx context.Context, userID, uuid string) (*model.UserCommand, error) {
	var cmd model.UserCommand
	result, err := m.cmdMod.Collection.FindOne(ctx, bson.M{"user_id": userID, "uuid": uuid})
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
	cursor, err := m.cmdMod.Collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &cmds); err != nil {
		return nil, err
	}
	return cmds, nil
}

// =====================================================
// 管理员权限（基于 im_users.app_manager_level）
// =====================================================

func (m *defaultUserModel) IsIMAdmin(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, ErrUserNotFound
	}
	sfKey := "user:imadmin:" + userID
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var user model.User
		result, err := m.userMod.Collection.FindOne(ctx, bson.M{"user_id": userID})
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
	})
	if err != nil {
		return false, err
	}
	return v.(bool), nil
}
