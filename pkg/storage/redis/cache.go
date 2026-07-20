package redis

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	goredis "github.com/redis/go-redis/v9"
)

const (
	NilMarker    = "null"
	verKeySuffix = ":ver"

	// DefaultDoubleDeleteMinMs 延时双删的默认最小延迟（毫秒）。
	DefaultDoubleDeleteMinMs = 200
	// DefaultDoubleDeleteMaxMs 延时双删的默认最大延迟（毫秒）。
	DefaultDoubleDeleteMaxMs = 500
)

var (
	casSetScript = goredis.NewScript(`
-- KEYS[1] = dataKey, KEYS[2] = versionKey
-- ARGV[1] = jsonData (or NilMarker for nil), ARGV[2] = ttlSeconds, ARGV[3] = newVersion
-- Returns 1 if wrote, 0 if skipped (existing version >= newVersion or newer)
local existVer = redis.call('GET', KEYS[2])
if existVer then
    if tonumber(existVer) >= tonumber(ARGV[3]) then
        return 0
    end
end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
redis.call('SET', KEYS[2], ARGV[3], 'EX', ARGV[2])
return 1
`)
)

func cacheVerKey(dataKey string) string {
	return dataKey + verKeySuffix
}

func CacheGet(ctx context.Context, rdb goredis.UniversalClient, key string, result any) (found bool, err error) {
	if rdb == nil {
		return false, nil
	}
	data, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return false, nil
		}
		return false, err
	}
	if len(data) == 0 {
		return false, nil
	}
	if string(data) == NilMarker {
		return true, nil
	}
	if err := json.Unmarshal(data, result); err != nil {
		_ = rdb.Del(ctx, key).Err()
		return false, nil
	}
	return true, nil
}

func CacheSet(ctx context.Context, rdb goredis.UniversalClient, key string, value any, ttlSeconds int) error {
	if rdb == nil {
		return nil
	}
	ttl := time.Duration(ttlSeconds) * time.Second
	if value == nil {
		return rdb.Set(ctx, key, NilMarker, ttl).Err()
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, data, ttl).Err()
}

func CacheDel(ctx context.Context, rdb goredis.UniversalClient, keys ...string) error {
	if rdb == nil || len(keys) == 0 {
		return nil
	}
	delKeys := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		delKeys = append(delKeys, k, cacheVerKey(k))
	}
	return rdb.Del(ctx, delKeys...).Err()
}

func CacheGetString(ctx context.Context, rdb goredis.UniversalClient, key string) (val string, found bool, err error) {
	if rdb == nil {
		return "", false, nil
	}
	data, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return "", false, nil
		}
		return "", false, err
	}
	if data == "" {
		return "", false, nil
	}
	return data, true, nil
}

func CacheSetString(ctx context.Context, rdb goredis.UniversalClient, key string, value string, ttlSeconds int) error {
	if rdb == nil {
		return nil
	}
	return rdb.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
}

func CacheIsNil(result any) bool {
	switch v := result.(type) {
	case nil:
		return true
	case string:
		return v == ""
	}
	return false
}

// CacheSetCAS 以「版本号原子比对」的方式写入 JSON 缓存。
//
// 通过 Lua 脚本在 Redis 侧保证：仅当 Redis 上已存在的 versionKey 值 < newVersion（或 versionKey 尚不存在）时，
// 才同时写入 dataKey 与 versionKey（两者 TTL 相同）。
//
// 典型用法：
//
//	dbUser, err := mongo.FindOne(...)
//	if err != nil {
//	    // 空值缓存 version 恒为 0，真实数据写入后会自动将其覆盖
//	    sredis.CacheSetCAS(ctx, rdb, key, nil, 0, ttl)
//	} else {
//	    sredis.CacheSetCAS(ctx, rdb, key, dbUser, dbUser.UpdatedAt.UnixMilli(), ttl)
//	}
//
// newVersion 必须单调递增（如 UpdatedAt.UnixMilli()）；写 nil 会序列化为 NilMarker。
// 返回 wrote=true 表示实际写入，false 表示因「Redis 上已有更新版本」而跳过。
func CacheSetCAS(ctx context.Context, rdb goredis.UniversalClient, key string, value any, newVersion int64, ttlSeconds int) (wrote bool, err error) {
	if rdb == nil {
		return false, nil
	}

	var payload string
	if value == nil {
		payload = NilMarker
	} else {
		data, jerr := json.Marshal(value)
		if jerr != nil {
			return false, jerr
		}
		payload = string(data)
	}

	vk := cacheVerKey(key)
	res, err := casSetScript.Run(ctx, rdb,
		[]string{key, vk},
		payload, strconv.Itoa(ttlSeconds), convert.ToString(newVersion),
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// CacheSetCASString 是 CacheSetCAS 的字符串版本，用法完全一致，仅 value 类型为 string。
// 典型用于存在性标记、0/1 布尔、枚举数字等简单字段。
func CacheSetCASString(ctx context.Context, rdb goredis.UniversalClient, key string, value string, newVersion int64, ttlSeconds int) (wrote bool, err error) {
	if rdb == nil {
		return false, nil
	}
	vk := cacheVerKey(key)
	res, err := casSetScript.Run(ctx, rdb,
		[]string{key, vk},
		value, strconv.Itoa(ttlSeconds), strconv.FormatInt(newVersion, 10),
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// CacheDelDouble 执行「延时双删」：立即同步删除 keys，再异步随机延时 [DefaultDoubleDeleteMinMs, DefaultDoubleDeleteMaxMs]
// 后再删除一次 keys，以兜住多实例部署下「慢回源读写晚于 Update 删除」的脏数据竞态。
//
// 内部处理：
//   - keys 深拷贝后进入异步 goroutine，避免外部修改切片内容。
//   - 每个删除会自动带上每个 key 对应的 :ver 版本小 key（底层复用 CacheDel）。
//   - 异步 goroutine 带 recover() 保护，即便 ctx 被取消或 Redis 报错也不会 panic 泄漏。
//   - 若 rdb == nil 或 keys 为空，直接返回空。
//
// 典型用法：
//
//	if err := mongo.Update(user); err != nil { return err }
//	sredis.CacheDelDouble(ctx, rdb,
//	    GetUserInfoKey(user.UserID), GetUserExistsKey(user.UserID),
//	)
func CacheDelDouble(ctx context.Context, rdb goredis.UniversalClient, keys ...string) {
	CacheDelDoubleWithDelay(ctx, rdb, DefaultDoubleDeleteMinMs, DefaultDoubleDeleteMaxMs, keys...)
}

// CacheDelDoubleWithDelay 与 CacheDelDouble 语义一致，但允许自定义异步第二次删除的
// 延时范围（[delayMinMs, delayMaxMs) 毫秒）。
//
// 若 delayMaxMs <= delayMinMs，会直接取 delayMinMs 作为固定延时。
// delayMinMs < 0 时会被钳位为 0。
func CacheDelDoubleWithDelay(ctx context.Context, rdb goredis.UniversalClient, delayMinMs, delayMaxMs int, keys ...string) {
	if rdb == nil || len(keys) == 0 {
		return
	}

	_ = CacheDel(ctx, rdb, keys...)

	if delayMinMs < 0 {
		delayMinMs = 0
	}
	var delayMs int
	if delayMaxMs <= delayMinMs {
		delayMs = delayMinMs
	} else {
		delayMs = randx.IntnRange(delayMinMs, delayMaxMs)
	}
	delay := time.Duration(delayMs) * time.Millisecond

	keysCopy := make([]string, len(keys))
	copy(keysCopy, keys)
	go func() {
		defer func() { _ = recover() }()
		if delay > 0 {
			time.Sleep(delay)
		}
		_ = CacheDel(ctx, rdb, keysCopy...)
	}()
}
