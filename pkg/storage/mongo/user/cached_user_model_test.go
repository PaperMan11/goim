package user

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PaperMan11/goim/pkg/storage/model"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/syncx"
)

type stubUserModel struct {
	mu sync.Mutex

	// key = userID + "|" + str(platform) + "|" + deviceID → *UserStatus，模拟 Mongo 唯一键 (userID, platformID, deviceID)
	statuses map[string]*model.UserStatus

	// 统计调用次数，用于断言缓存是否生效（命中后不再回源 DB）
	nGetUserStatus    atomic.Int32
	nUpdateUserStatus atomic.Int32
	nInsertUserStatus atomic.Int32
	nSetUserOnline    atomic.Int32
}

func newStubUserModel() *stubUserModel {
	return &stubUserModel{statuses: make(map[string]*model.UserStatus)}
}

func stubStatusKey(userID string, platformID int, deviceID string) string {
	return fmt.Sprintf("%s|%d|%s", userID, platformID, deviceID)
}

func (s *stubUserModel) InsertUserStatus(_ context.Context, status *model.UserStatus) error {
	if status == nil {
		return errors.New("nil status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nInsertUserStatus.Add(1)
	key := stubStatusKey(status.UserID, status.PlatformID, status.DeviceID)
	s.statuses[key] = cloneUserStatus(status)
	return nil
}

func (s *stubUserModel) UpdateUserStatus(_ context.Context, userID string, platformID int, deviceID string, status int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nUpdateUserStatus.Add(1)
	key := stubStatusKey(userID, platformID, deviceID)
	existing, ok := s.statuses[key]
	if !ok {
		existing = &model.UserStatus{
			UserID:     userID,
			PlatformID: platformID,
			DeviceID:   deviceID,
		}
		s.statuses[key] = existing
	}
	existing.Status = status
	existing.UpdatedAt = time.UnixMilli(timex.UnixMilli())
	return nil
}

func (s *stubUserModel) GetUserStatus(_ context.Context, userIDs []string) ([]*model.UserStatus, error) {
	s.nGetUserStatus.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()

	// userID -> platform -> { status, maxUpdateAt }
	agg := make(map[string]map[int]*model.UserStatus, len(userIDs))
	for _, k := range userIDs {
		agg[k] = make(map[int]*model.UserStatus)
	}
	for _, row := range s.statuses {
		perUser, ok := agg[row.UserID]
		if !ok {
			continue
		}
		cur, exists := perUser[row.PlatformID]
		if !exists {
			cloned := cloneUserStatus(row)
			// 归零 DeviceID：因为聚合到 platform 粒度后不保留 device
			cloned.DeviceID = ""
			perUser[row.PlatformID] = cloned
			continue
		}
		// OR 语义：任一 device 在线 → Status=1
		if row.Status == 1 {
			cur.Status = 1
		}
		if row.UpdatedAt.After(cur.UpdatedAt) {
			cur.UpdatedAt = row.UpdatedAt
		}
	}

	result := make([]*model.UserStatus, 0)
	for _, userID := range userIDs {
		perUser := agg[userID]
		plats := make([]int, 0, len(perUser))
		for p := range perUser {
			plats = append(plats, p)
		}
		sort.Ints(plats)
		for _, p := range plats {
			result = append(result, perUser[p])
		}
	}
	return result, nil
}

func (s *stubUserModel) SetUserOnlineStatus(_ context.Context, statuses []*model.UserStatus) error {
	s.nSetUserOnline.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, st := range statuses {
		if st == nil {
			continue
		}
		key := stubStatusKey(st.UserID, st.PlatformID, st.DeviceID)
		s.statuses[key] = cloneUserStatus(st)
	}
	return nil
}

func (s *stubUserModel) GetAllOnlineUsers(context.Context) ([]string, error) { return nil, nil }

func (s *stubUserModel) Insert(context.Context, []*model.User) error {
	panic("stubUserModel.Insert not implemented in test")
}
func (s *stubUserModel) FindByIDs(context.Context, []string) ([]*model.User, error) {
	panic("not implemented")
}
func (s *stubUserModel) FindByID(context.Context, string) (*model.User, error) {
	panic("not implemented")
}
func (s *stubUserModel) Update(context.Context, *model.User) error { panic("not implemented") }
func (s *stubUserModel) UpdateEx(context.Context, string, map[string]any) error {
	panic("not implemented")
}
func (s *stubUserModel) Delete(context.Context, string) error { panic("not implemented") }
func (s *stubUserModel) Count(context.Context) (int64, error) { return 0, nil }
func (s *stubUserModel) Page(context.Context, int64, int64, string, string) ([]*model.User, int64, error) {
	return nil, 0, nil
}
func (s *stubUserModel) SortQuery(context.Context, map[string]string, bool) ([]*model.User, error) {
	return nil, nil
}
func (s *stubUserModel) CheckExists(context.Context, []string) (map[string]bool, error) {
	return nil, nil
}
func (s *stubUserModel) GetAllUserIDs(context.Context, int64, int64) ([]string, int64, error) {
	return nil, 0, nil
}
func (s *stubUserModel) SetGlobalRecvMsgOpt(context.Context, string, int) error   { return nil }
func (s *stubUserModel) GetGlobalRecvMsgOpt(context.Context, string) (int, error) { return 0, nil }
func (s *stubUserModel) RegisterCount(context.Context, int64, int64) (int64, int64, map[string]int64, error) {
	return 0, 0, nil, nil
}
func (s *stubUserModel) InsertUserCommand(context.Context, *model.UserCommand) error {
	panic("not implemented")
}
func (s *stubUserModel) UpdateUserCommand(context.Context, string, string, string) error {
	panic("not implemented")
}
func (s *stubUserModel) DeleteUserCommand(context.Context, string, string) error {
	panic("not implemented")
}
func (s *stubUserModel) GetUserCommand(context.Context, string, string) (*model.UserCommand, error) {
	panic("not implemented")
}
func (s *stubUserModel) GetAllUserCommands(context.Context, string) ([]*model.UserCommand, error) {
	panic("not implemented")
}
func (s *stubUserModel) IsIMAdmin(context.Context, string) (bool, error) { return false, nil }

func cloneUserStatus(s *model.UserStatus) *model.UserStatus {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}

// testHarness 组装 miniredis + stub + cache 层。
type testHarness struct {
	t        *testing.T
	Ctx      context.Context
	Mini     *miniredis.Miniredis
	Rdb      goredis.UniversalClient
	Stub     *stubUserModel
	Cached   UserModel        // 外层缓存
	Cached2  *cachedUserModel // 内部类型，方便断言未导出字段
	Teardown func()
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewUniversalClient(&goredis.UniversalOptions{Addrs: []string{mr.Addr()}})

	stub := newStubUserModel()
	cached := NewCachedUserModel(stub, rdb, syncx.NewSingleFlight())
	inner, _ := cached.(*cachedUserModel)

	return &testHarness{
		t:       t,
		Ctx:     context.Background(),
		Mini:    mr,
		Rdb:     rdb,
		Stub:    stub,
		Cached:  cached,
		Cached2: inner,
		Teardown: func() {
			_ = rdb.Close()
			mr.Close()
		},
	}
}

// ============================================================
// 测试用例
// ============================================================

// TestGetUserStatus_BasicCacheHit —— 第一次查回源，缓存生效后第二次不回源，结果一致。
func TestGetUserStatus_BasicCacheHit(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	// 预置 stub DB 数据：user1 在 platform 1 在线，platform 2 离线（两个 device 分别不同状态）
	nowMs := timex.UnixMilli()
	now := time.UnixMilli(nowMs)
	_ = h.Stub.InsertUserStatus(h.Ctx, &model.UserStatus{
		UserID: "u1", PlatformID: 1, DeviceID: "dev1", Status: 1, UpdatedAt: now,
	})
	_ = h.Stub.InsertUserStatus(h.Ctx, &model.UserStatus{
		UserID: "u1", PlatformID: 2, DeviceID: "dev2", Status: 0, UpdatedAt: now,
	})

	// 第一次：DB 调用 1 次，返回只有 Status=1 的 platform=1（Status=0 不返，由逻辑层预填）
	rows, err := h.Cached.GetUserStatus(h.Ctx, []string{"u1"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load(), "第一次查询应回源 1 次")
	assert.Len(t, rows, 1, "ZSET 只存在线，应只返 platform=1 一条")
	assert.Equal(t, "u1", rows[0].UserID)
	assert.Equal(t, 1, rows[0].PlatformID)
	assert.Equal(t, 1, rows[0].Status)

	// 验证缓存 key 真的被写了（miniredis 可以直接看 key 类型）
	zKey := GetUserStatusZKey("u1")
	assert.True(t, h.Mini.Exists(zKey), "缓存 ZSET key 应存在")
	score, _ := h.Rdb.ZScore(h.Ctx, zKey, PlatformZMember(1)).Result()
	assert.Greater(t, score, float64(0))

	// 第二次：命中缓存，DB 调用次数不变
	rows2, err := h.Cached.GetUserStatus(h.Ctx, []string{"u1"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load(), "命中缓存不应再回源 DB")
	assert.Len(t, rows2, 1)
	assert.Equal(t, 1, rows2[0].PlatformID)
	assert.Equal(t, 1, rows2[0].Status)
}

// TestGetUserStatus_NilMarker - 用户确实全离线，第一次回源写 Nil Marker，第二次查命中 Marker 不回源。
func TestGetUserStatus_NilMarker(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	// stub DB 里没有 user2 的任何 status 行（全离线语义）
	rows1, err := h.Cached.GetUserStatus(h.Ctx, []string{"u2"})
	assert.NoError(t, err)
	assert.Empty(t, rows1, "全离线应返回 0 行")
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load())

	nilKey := GetUserStatusZNilKey("u2")
	assert.True(t, h.Mini.Exists(nilKey), "Nil Marker key 应存在")

	rows2, err := h.Cached.GetUserStatus(h.Ctx, []string{"u2"})
	assert.NoError(t, err)
	assert.Empty(t, rows2)
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load(), "命中 Nil Marker 不应回源 DB")
}

// TestGetUserStatus_ZombieLazyClean —— 把一个平台 score 写得极老（僵尸），查时会过滤，且顺手真的 ZREM 清掉。
func TestGetUserStatus_ZombieLazyClean(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	const userID = "u_zombie"
	zKey := GetUserStatusZKey(userID)

	// 直接写 Redis：platform 1 = 在线（score=nowMs, 活的）
	// platform 2 = 僵尸（score=nowMs - 1小时，远超 120s 阈值）
	nowMs := timex.UnixMilli()
	_, _ = h.Rdb.ZAdd(h.Ctx, zKey, goredis.Z{Member: PlatformZMember(1), Score: float64(nowMs)}).Result()
	_, _ = h.Rdb.ZAdd(h.Ctx, zKey, goredis.Z{Member: PlatformZMember(2), Score: float64(nowMs - 3600*1000)}).Result()
	// TTL: 90s
	_ = h.Rdb.Expire(h.Ctx, zKey, 90*time.Second).Err()

	assert.Equal(t, int64(2), h.Rdb.ZCard(h.Ctx, zKey).Val(), "初始 2 个 member")

	rows, err := h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.NoError(t, err)
	assert.Len(t, rows, 1, "只应返回 1 个活平台")
	assert.Equal(t, 1, rows[0].PlatformID)

	// 懒清理真的把僵尸删了
	assert.Equal(t, int64(1), h.Rdb.ZCard(h.Ctx, zKey).Val(), "僵尸 member 应被 ZREMRANGEBYSCORE 清掉")
	_, zErr := h.Rdb.ZScore(h.Ctx, zKey, PlatformZMember(2)).Result()
	assert.ErrorIs(t, zErr, goredis.Nil, "僵尸 platform 2 已不存在，ZScore 返回 redis.Nil")
}

// TestGetUserStatus_BatchMix —— 一批 userID 混合命中/未命中/Nil，验证 SingleFlight 合并与缓存。
func TestGetUserStatus_BatchMix(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	now := time.Now()
	// a 在线
	_ = h.Stub.InsertUserStatus(h.Ctx, &model.UserStatus{UserID: "a", PlatformID: 1, Status: 1, UpdatedAt: now})
	// b 全离线（不插入）
	// c 在线
	_ = h.Stub.InsertUserStatus(h.Ctx, &model.UserStatus{UserID: "c", PlatformID: 5, Status: 1, UpdatedAt: now})

	// 第一次：回源一次，整批 userIDs
	rows, err := h.Cached.GetUserStatus(h.Ctx, []string{"a", "b", "c"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load())

	// 做行结果统计：按 userID -> count(rows)
	count := map[string]int{}
	for _, r := range rows {
		count[r.UserID]++
	}
	assert.Equal(t, 1, count["a"], "a 应 1 行（平台 1 在线）")
	assert.Equal(t, 0, count["b"], "b 全离线 0 行，由逻辑层兜底")
	assert.Equal(t, 1, count["c"], "c 应 1 行（平台 5 在线）")

	// b 的 Nil Marker 存在
	assert.True(t, h.Mini.Exists(GetUserStatusZNilKey("b")))
	// a, c 的 ZSET 存在
	assert.True(t, h.Mini.Exists(GetUserStatusZKey("a")))
	assert.True(t, h.Mini.Exists(GetUserStatusZKey("c")))

	// 第二次：整批都命中缓存或 Nil Marker，DB 调用不增加
	rows2, err := h.Cached.GetUserStatus(h.Ctx, []string{"a", "b", "c"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load(), "第二次应完全命中缓存/Marker，零回源")
	// rows2 长度同 rows（含 b 的 0 条不含；最终 total=2）
	assert.Len(t, rows2, 2)
}

// TestUpdateUserStatus_OnlineOfflineRoundTrip
func TestUpdateUserStatus_OnlineOfflineRoundTrip(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	const userID = "u_toggle"
	const plat = 3
	const dev = "phone1"

	// 1. 上线：UpdateUserStatus Status=1
	err := h.Cached.UpdateUserStatus(h.Ctx, userID, plat, dev, 1)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nUpdateUserStatus.Load())

	// 验证：ZSET key 有 member=3，Nil Marker 删除
	zKey := GetUserStatusZKey(userID)
	nilKey := GetUserStatusZNilKey(userID)
	assert.True(t, h.Mini.Exists(zKey))
	assert.False(t, h.Mini.Exists(nilKey))
	card, _ := h.Rdb.ZCard(h.Ctx, zKey).Result()
	assert.Equal(t, int64(1), card)

	// 查 GetUserStatus：有 1 条 online，不触发回源（UpdateUserStatus 已直接写缓存）
	rows, err := h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), h.Stub.nGetUserStatus.Load(), "Update 已直接写缓存，GetUserStatus 不应回源")
	assert.Len(t, rows, 1)
	assert.Equal(t, plat, rows[0].PlatformID)
	assert.Equal(t, 1, rows[0].Status)

	// 2. 下线：Status=0
	err = h.Cached.UpdateUserStatus(h.Ctx, userID, plat, dev, 0)
	assert.NoError(t, err)
	assert.Equal(t, int32(2), h.Stub.nUpdateUserStatus.Load())

	// ZREM 后 ZSET 为空 → 写 Nil Marker
	card, _ = h.Rdb.ZCard(h.Ctx, zKey).Result()
	assert.Equal(t, int64(0), card)
	assert.True(t, h.Mini.Exists(nilKey), "空 zset 后应写 Nil Marker")

	// 查 GetUserStatus：Nil Marker 命中，0 行不回源
	rows2, err := h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.NoError(t, err)
	assert.Empty(t, rows2)
	assert.Equal(t, int32(0), h.Stub.nGetUserStatus.Load())
}

// TestSetUserOnlineStatus_BatchMixed —— 批量 SetUserOnlineStatus 含在线+离线、跨多用户、同 user 同 platform 多 device 的 OR 语义。
func TestSetUserOnlineStatus_BatchMixed(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	now := time.Now()
	batch := []*model.UserStatus{
		// user_a: plat 1 在线(dev1)，plat 2 离线(dev2) → 最终 plat1=online, plat2=offline
		{UserID: "user_a", PlatformID: 1, DeviceID: "d1", Status: 1, UpdatedAt: now},
		{UserID: "user_a", PlatformID: 2, DeviceID: "d2", Status: 0, UpdatedAt: now},

		// user_b: 同 platform 3，一个离线(dev3) 一个在线(dev4) → OR 语义 plat3=online
		{UserID: "user_b", PlatformID: 3, DeviceID: "d3", Status: 0, UpdatedAt: now},
		{UserID: "user_b", PlatformID: 3, DeviceID: "d4", Status: 1, UpdatedAt: now},

		// user_c: 全离线 (plat 4 offline)
		{UserID: "user_c", PlatformID: 4, DeviceID: "d5", Status: 0, UpdatedAt: now},
	}
	err := h.Cached.SetUserOnlineStatus(h.Ctx, batch)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nSetUserOnline.Load())

	// 验证缓存写的 ZSET：
	// user_a: plat 1 有 member，plat 2 无
	za, _ := h.Rdb.ZRange(h.Ctx, GetUserStatusZKey("user_a"), 0, -1).Result()
	assert.Contains(t, za, PlatformZMember(1), "user_a plat1 在线")
	assert.NotContains(t, za, PlatformZMember(2), "user_a plat2 离线不应写入 zset")

	// user_b: plat 3 在线（OR 语义）
	zb, _ := h.Rdb.ZRange(h.Ctx, GetUserStatusZKey("user_b"), 0, -1).Result()
	assert.Contains(t, zb, PlatformZMember(3), "user_b 任一 device online → plat3 算在线")

	// user_c: 全离线 → Nil Marker 存在（因为 SetUserOnlineStatus 最后走 CacheDelOnEmptyZ）
	assert.False(t, h.Mini.Exists(GetUserStatusZKey("user_c")), "user_c 全离线不应有 ZSET key")
	assert.True(t, h.Mini.Exists(GetUserStatusZNilKey("user_c")), "user_c 全离线应有 Nil Marker")

	// 再 GetUserStatus 一次验证零回源
	rows, err := h.Cached.GetUserStatus(h.Ctx, []string{"user_a", "user_b", "user_c"})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), h.Stub.nGetUserStatus.Load())

	cnt := map[string]int{}
	for _, r := range rows {
		cnt[r.UserID]++
	}
	assert.Equal(t, 1, cnt["user_a"])
	assert.Equal(t, 1, cnt["user_b"])
	assert.Equal(t, 0, cnt["user_c"])
}

// TestInsertUserStatus_InvalidateCache —— InsertUserStatus 会把该用户 zset + nil marker 整俩 key 都 DEL。
func TestInsertUserStatus_InvalidateCache(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	userID := "u_inv"
	now := time.Now()

	// 第一步：填充缓存（查一次，回源返回空写 Nil Marker）
	rows1, err := h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.NoError(t, err)
	assert.Empty(t, rows1)
	assert.True(t, h.Mini.Exists(GetUserStatusZNilKey(userID)))

	// 第二步：InsertUserStatus（用户上线了）
	err = h.Cached.InsertUserStatus(h.Ctx, &model.UserStatus{
		UserID: userID, PlatformID: 7, DeviceID: "d", Status: 1, UpdatedAt: now,
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), h.Stub.nInsertUserStatus.Load())

	// 第三步：Nil Marker 已被删（因为 InsertUserStatus 调用了 CacheDelDouble）
	assert.False(t, h.Mini.Exists(GetUserStatusZNilKey(userID)), "Insert 后 Nil Marker 应被清")
	assert.False(t, h.Mini.Exists(GetUserStatusZKey(userID)), "Insert 后 ZSET 应被清")

	// 第四步：再查 —— 会回源第 2 次（因为缓存被失效）
	rows2, err := h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), h.Stub.nGetUserStatus.Load(), "缓存失效后应重新回源")
	assert.Len(t, rows2, 1)
	assert.Equal(t, 7, rows2[0].PlatformID)
	assert.Equal(t, 1, rows2[0].Status)
}

// TestZRowsForUser_AllZombieTriggersDB —— 所有 member 都是僵尸→ needDB=true，回源重算。
func TestZRowsForUser_AllZombieTriggersDB(t *testing.T) {
	h := newTestHarness(t)
	defer h.Teardown()

	userID := "all_zombie"
	zKey := GetUserStatusZKey(userID)

	// 写两个 member，score 都比阈值老
	stale := float64(timex.UnixMilli() - 10*60*1000) // 10 分钟前
	_, _ = h.Rdb.ZAdd(h.Ctx, zKey, goredis.Z{Member: "1", Score: stale}).Result()
	_, _ = h.Rdb.ZAdd(h.Ctx, zKey, goredis.Z{Member: "2", Score: stale}).Result()

	// 先不写 stub DB（查也返回空）
	rows, err := h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.NoError(t, err)
	// 回源被触发 → nGetUserStatus=1
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load(), "全僵尸必须回源 DB 确认")
	assert.Empty(t, rows, "DB 无数据 + 过滤僵尸 → 空 rows")
	// 懒清理后 ZSET 为空，写 Nil Marker
	card, _ := h.Rdb.ZCard(h.Ctx, zKey).Result()
	assert.Equal(t, int64(0), card, "懒清理后 zset 应空")
	assert.True(t, h.Mini.Exists(GetUserStatusZNilKey(userID)), "确认全离线 → 写 Nil Marker")

	// 第二次查：命中 Nil Marker → 0 回源
	_, _ = h.Cached.GetUserStatus(h.Ctx, []string{userID})
	assert.Equal(t, int32(1), h.Stub.nGetUserStatus.Load())
}

// TestGetUserStatus_RedisNil —— 走 rdb == nil 分支（纯透传 DB，不影响）。
func TestGetUserStatus_NoRedisPassthrough(t *testing.T) {
	stub := newStubUserModel()
	cachedNoRdb := NewCachedUserModel(stub, nil, syncx.NewSingleFlight()) // redis=nil 时走纯 DB
	ctx := context.Background()

	now := time.Now()
	_ = stub.InsertUserStatus(ctx, &model.UserStatus{UserID: "u_noredis", PlatformID: 1, Status: 1, UpdatedAt: now})

	rows, err := cachedNoRdb.GetUserStatus(ctx, []string{"u_noredis"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), stub.nGetUserStatus.Load())
	assert.Len(t, rows, 1)
}
