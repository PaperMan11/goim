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
	NilMarker = "null"

	DefaultDoubleDeleteMinMs = 200
	DefaultDoubleDeleteMaxMs = 500
)

type cachedEnvelope struct {
	V int64           `json:"v"`
	D json.RawMessage `json:"d"`
}

var (
	casSetScript = goredis.NewScript(`
-- KEYS[1] = cacheKey (单 key，合并存储 {v, d} JSON，完全兼容 Redis Cluster，无跨 slot)
-- ARGV[1] = 最终写入 JSON 字符串（已在 client 端把 version + data 合并好）
-- ARGV[2] = ttlSeconds
-- ARGV[3] = newVersion (字符串)
-- 返回 1 = 已写入，0 = 已存在更新版本，跳过
local raw = redis.call('GET', KEYS[1])
if raw then
    local ok, cur = pcall(cjson.decode, raw)
    if ok and cur and type(cur.v) == 'number' and cur.v >= tonumber(ARGV[3]) then
        return 0
    end
end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
return 1
`)
)

func newNilRawEnvelope() json.RawMessage { return json.RawMessage(NilMarker) }

func marshalEnvelopeRaw(version int64, inner json.RawMessage) ([]byte, error) {
	if len(inner) == 0 {
		inner = newNilRawEnvelope()
	}
	env := cachedEnvelope{V: version, D: inner}
	return json.Marshal(&env)
}

func marshalEnvelopeAny(version int64, value any) ([]byte, error) {
	var inner json.RawMessage
	if value == nil {
		inner = newNilRawEnvelope()
	} else {
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		inner = json.RawMessage(data)
	}
	return marshalEnvelopeRaw(version, inner)
}

func marshalEnvelopeString(version int64, value string) ([]byte, error) {
	inner, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return marshalEnvelopeRaw(version, json.RawMessage(inner))
}

func unmarshalEnvelope(data []byte) (*cachedEnvelope, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var env cachedEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func (e *cachedEnvelope) isNil() bool {
	return e == nil || len(e.D) == 0 || string(e.D) == NilMarker
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
	env, err := unmarshalEnvelope(data)
	if err != nil || env == nil {
		_ = rdb.Del(ctx, key).Err()
		return false, nil
	}
	if env.isNil() {
		return true, nil
	}
	if err := json.Unmarshal(env.D, result); err != nil {
		_ = rdb.Del(ctx, key).Err()
		return false, nil
	}
	return true, nil
}

func CacheSet(ctx context.Context, rdb goredis.UniversalClient, key string, value any, ttlSeconds int) error {
	if rdb == nil {
		return nil
	}
	payload, err := marshalEnvelopeAny(0, value)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, payload, time.Duration(ttlSeconds)*time.Second).Err()
}

func CacheDel(ctx context.Context, rdb goredis.UniversalClient, keys ...string) error {
	if rdb == nil || len(keys) == 0 {
		return nil
	}
	pipe := rdb.Pipeline()
	for _, k := range keys {
		if k == "" {
			continue
		}
		pipe.Del(ctx, k)
	}
	_, err := pipe.Exec(ctx)
	return err
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
	env, err := unmarshalEnvelope([]byte(data))
	if err != nil || env == nil {
		_ = rdb.Del(ctx, key).Err()
		return "", false, nil
	}
	if env.isNil() {
		return NilMarker, true, nil
	}
	var s string
	if err := json.Unmarshal(env.D, &s); err != nil {
		_ = rdb.Del(ctx, key).Err()
		return "", false, nil
	}
	return s, true, nil
}

func CacheSetString(ctx context.Context, rdb goredis.UniversalClient, key string, value string, ttlSeconds int) error {
	if rdb == nil {
		return nil
	}
	payload, err := marshalEnvelopeString(0, value)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, payload, time.Duration(ttlSeconds)*time.Second).Err()
}

func CacheIsNil(result any) bool {
	switch v := result.(type) {
	case nil:
		return true
	case string:
		return v == "" || v == NilMarker
	}
	return false
}

func CacheSetCAS(ctx context.Context, rdb goredis.UniversalClient, key string, value any, newVersion int64, ttlSeconds int) (wrote bool, err error) {
	if rdb == nil {
		return false, nil
	}
	payload, err := marshalEnvelopeAny(newVersion, value)
	if err != nil {
		return false, err
	}
	res, err := casSetScript.Run(ctx, rdb,
		[]string{key},
		string(payload), strconv.Itoa(ttlSeconds), convert.ToString(newVersion),
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

func CacheSetCASString(ctx context.Context, rdb goredis.UniversalClient, key string, value string, newVersion int64, ttlSeconds int) (wrote bool, err error) {
	if rdb == nil {
		return false, nil
	}
	payload, err := marshalEnvelopeString(newVersion, value)
	if err != nil {
		return false, err
	}
	res, err := casSetScript.Run(ctx, rdb,
		[]string{key},
		string(payload), strconv.Itoa(ttlSeconds), strconv.FormatInt(newVersion, 10),
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
