package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// CacheZAddGT 原子性执行带 GT（Greater Than）保护的 ZADD：
// 仅当 member 不存在，或 member 当前 score < newScore 时才写入。
// 用于心跳上报场景：迟到的旧心跳无法覆盖新心跳的 score，避免 status 回退。
//
// 若 expireSeconds > 0，写入成功后会顺带对整个 ZSET key 执行 EXPIRE（NX 不覆盖）。
// 返回值 added = 本次实际修改或新增的 member 数量。
func CacheZAddGT(ctx context.Context, rdb goredis.UniversalClient, key string, newScore float64, member string, expireSeconds int) (added int64, err error) {
	if rdb == nil || key == "" || member == "" {
		return 0, nil
	}

	pipe := rdb.Pipeline()

	// ZAddArgs 支持 GT：只在新 score 严格大于当前 score 才生效
	pipe.ZAddArgs(ctx, key, goredis.ZAddArgs{
		GT:      true,
		Members: []goredis.Z{{Score: newScore, Member: member}},
	})

	// EXPIRE NX：只在 key 没有 TTL 时（新建/刚写过）设置，不缩短已存在的过期
	if expireSeconds > 0 {
		pipe.ExpireNX(ctx, key, time.Duration(expireSeconds)*time.Second)
	}

	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	// 取第 0 条（ZADD）返回值
	zcmd, ok := cmders[0].(*goredis.IntCmd)
	if !ok {
		return 0, nil
	}
	return zcmd.Val(), nil
}

func CacheZAddGTBatch(ctx context.Context, rdb goredis.UniversalClient, key string, members map[string]int64, expireSeconds int) (addedTotal int64, err error) {
	if rdb == nil || key == "" || len(members) == 0 {
		return 0, nil
	}

	zs := make([]goredis.Z, 0, len(members))
	for m, s := range members {
		zs = append(zs, goredis.Z{Score: float64(s), Member: m})
	}
	pipe := rdb.Pipeline()
	pipe.ZAddArgs(ctx, key, goredis.ZAddArgs{GT: true, Members: zs})
	if expireSeconds > 0 {
		pipe.ExpireNX(ctx, key, time.Duration(expireSeconds)*time.Second)
	}
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	zcmd, ok := cmders[0].(*goredis.IntCmd)
	if !ok {
		return 0, nil
	}
	return zcmd.Val(), nil
}

func CacheZRem(ctx context.Context, rdb goredis.UniversalClient, key string, members ...string) (removed int64, err error) {
	if rdb == nil || key == "" || len(members) == 0 {
		return 0, nil
	}
	// convert []string → []any
	anyMembers := make([]any, 0, len(members))
	for _, m := range members {
		if m == "" {
			continue
		}
		anyMembers = append(anyMembers, m)
	}
	if len(anyMembers) == 0 {
		return 0, nil
	}
	return rdb.ZRem(ctx, key, anyMembers...).Result()
}

// CacheZRangeWithScores 返回整个 ZSET 的所有成员与 score（按 score 升序）。
func CacheZRangeWithScores(ctx context.Context, rdb goredis.UniversalClient, key string) (zs []goredis.Z, found bool, err error) {
	if rdb == nil || key == "" {
		return nil, false, nil
	}
	res, err := rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(res) == 0 {
		return nil, false, nil
	}
	return res, true, nil
}

// CacheZCard 获取 key 的 member 数（key 不存在或空返回 0）。
func CacheZCard(ctx context.Context, rdb goredis.UniversalClient, key string) (int64, error) {
	if rdb == nil || key == "" {
		return 0, nil
	}
	return rdb.ZCard(ctx, key).Result()
}

func CacheZRemRangeByScore(ctx context.Context, rdb goredis.UniversalClient, key, min, max string) (removed int64, err error) {
	if rdb == nil || key == "" {
		return 0, nil
	}
	return rdb.ZRemRangeByScore(ctx, key, min, max).Result()
}

func CacheExpireNX(ctx context.Context, rdb goredis.UniversalClient, key string, expireSeconds int) error {
	if rdb == nil || key == "" || expireSeconds <= 0 {
		return nil
	}
	return rdb.ExpireNX(ctx, key, time.Duration(expireSeconds)*time.Second).Err()
}

// CacheDelOnEmptyZ 是工具方法：当 ZSET 处理完（比如 worker 清僵尸后）如果 ZCARD=0，
// 则 DEL 整个 key，避免 Redis 留下空 key 以及对应 Nil Marker 不生效的问题。
// 若同时传入对应的 nilMarkerKey（可为空），则在 DEL 空 key 后顺手把 Nil Marker 写好（TTL = nilTTL 秒）。
// 写入 Nil Marker 走的是 STRING CacheSetCAS with nil value + version 0（与 marker 模式一致）。
func CacheDelOnEmptyZ(ctx context.Context, rdb goredis.UniversalClient, key, nilMarkerKey string, nilTTL int) error {
	if rdb == nil || key == "" {
		return nil
	}
	card, err := rdb.ZCard(ctx, key).Result()
	if err != nil {
		return err
	}
	if card > 0 {
		return nil
	}
	// ZSET 为空：删掉空 key，顺手写 Nil Marker
	_ = rdb.Del(ctx, key).Err()
	if nilMarkerKey != "" && nilTTL > 0 {
		_, _ = CacheSetCAS(ctx, rdb, nilMarkerKey, nil, 0, nilTTL)
	}
	return nil
}
