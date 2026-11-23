package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaperMan11/goim/pkg/utils/convert"
	"time"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
)

// expireZSetAtMaxScoreScript 把 ZSet 的过期时间对齐到成员中的最大 score（秒）。
// token 索引 ZSet 的 member score 就是 token 的过期时间，因此取最大 score 即为
// "最后一个 token 的过期时刻"；key 到期后由 Redis 自动回收，避免索引永久驻留。
// 单 key 脚本，兼容 Redis Cluster（不跨 slot）。
var expireZSetAtMaxScoreScript = goredis.NewScript(`
local top = redis.call('ZRANGE', KEYS[1], -1, -1, 'WITHSCORES')
if not top[1] then
    return 0
end
local expireAtMs = math.floor(tonumber(top[2]) * 1000)
if expireAtMs <= 0 then
    redis.call('PERSIST', KEYS[1])
    return 0
end
redis.call('PEXPIREAT', KEYS[1], tostring(expireAtMs))
return expireAtMs
`)

type RedisStore struct {
	redisClient goredis.UniversalClient
}

func NewRedisStore(redisClient goredis.UniversalClient) *RedisStore {
	return &RedisStore{
		redisClient: redisClient,
	}
}

var _ TokenStore = (*RedisStore)(nil)

func (rs *RedisStore) Close() error {
	return nil
}

func (rs *RedisStore) StoreToken(ctx context.Context, info *TokenInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	ttl := time.Duration(0)
	if info.ExpireAt > 0 {
		ttl = time.Until(time.Unix(info.ExpireAt, 0))
		if ttl <= 0 {
			return ErrTokenExpired
		}
	}

	if ttl > 0 {
		err = rs.redisClient.Set(ctx, GetTokenKey(info.UUID), string(data), ttl).Err()
	} else {
		err = rs.redisClient.Set(ctx, GetTokenKey(info.UUID), string(data), 0).Err()
	}
	if err != nil {
		return err
	}

	_, err = rs.redisClient.ZAdd(ctx, GetUserTokensKey(info.UserID), goredis.Z{
		Score:  float64(info.ExpireAt),
		Member: info.UUID,
	}).Result()
	if err != nil {
		return err
	}

	_, err = rs.redisClient.ZAdd(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), goredis.Z{
		Score:  float64(info.ExpireAt),
		Member: info.UUID,
	}).Result()
	if err != nil {
		return err
	}

	// 给两个索引 ZSet 设置"最大 token 过期时刻"作为自身 TTL，
	// 保证最后一个 token 过期后索引 key 被 Redis 自动删除，不产生僵尸数据。
	if err = rs.setZSetExpireAtMaxScore(ctx, GetUserTokensKey(info.UserID)); err != nil {
		return err
	}
	if err = rs.setZSetExpireAtMaxScore(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID)); err != nil {
		return err
	}

	return nil
}

// setZSetExpireAtMaxScore 在 Redis 端原子读取 ZSet 最大 score 并设置 PEXPIREAT。
func (rs *RedisStore) setZSetExpireAtMaxScore(ctx context.Context, key string) error {
	return expireZSetAtMaxScoreScript.Run(ctx, rs.redisClient, []string{key}).Err()
}

func (rs *RedisStore) GetToken(ctx context.Context, uuid string) (*TokenInfo, error) {
	data, err := rs.redisClient.Get(ctx, GetTokenKey(uuid)).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, ErrTokenNotFound
		}
		return nil, ErrTokenNotFound
	}
	if data == "" {
		return nil, ErrTokenNotFound
	}

	var info TokenInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, ErrTokenInvalid
	}

	if info.ExpireAt > 0 && timex.Unix() > info.ExpireAt {
		_ = rs.deleteTokenInternal(ctx, &info)
		return nil, ErrTokenExpired
	}

	return &info, nil
}

func (rs *RedisStore) DeleteToken(ctx context.Context, uuid string) error {
	data, err := rs.redisClient.Get(ctx, GetTokenKey(uuid)).Result()
	if err != nil || data == "" {
		return nil
	}

	var info TokenInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil
	}

	return rs.deleteTokenInternal(ctx, &info)
}

func (rs *RedisStore) DeleteTokens(ctx context.Context, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}

	var tokenKeys []string
	var infos []*TokenInfo

	for _, uuid := range uuids {
		data, err := rs.redisClient.Get(ctx, GetTokenKey(uuid)).Result()
		if err != nil || data == "" {
			continue
		}

		var info TokenInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			continue
		}

		tokenKeys = append(tokenKeys, GetTokenKey(uuid))
		infos = append(infos, &info)
	}

	if len(tokenKeys) > 0 {
		_ = rs.redisClient.Del(ctx, tokenKeys...).Err()

		for _, info := range infos {
			_, _ = rs.redisClient.ZRem(ctx, GetUserTokensKey(info.UserID), info.UUID).Result()
			_, _ = rs.redisClient.ZRem(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), info.UUID).Result()
			_ = rs.publishTokenDelete(ctx, info.UUID)
		}
	}

	return nil
}

func (rs *RedisStore) deleteTokenInternal(ctx context.Context, info *TokenInfo) error {
	if err := rs.redisClient.Del(ctx, GetTokenKey(info.UUID)).Err(); err != nil {
		return err
	}
	if _, err := rs.redisClient.ZRem(ctx, GetUserTokensKey(info.UserID), info.UUID).Result(); err != nil {
		return err
	}
	if _, err := rs.redisClient.ZRem(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), info.UUID).Result(); err != nil {
		return err
	}

	_ = rs.publishTokenDelete(ctx, info.UUID)

	return nil
}

func (rs *RedisStore) DeleteUserTokens(ctx context.Context, userID string, platformID ...int32) error {
	if len(platformID) > 0 {
		platformTokensKey := GetPlatformTokenKey(userID, platformID[0])
		tokenUUIDs, err := rs.redisClient.ZRange(ctx, platformTokensKey, 0, -1).Result()
		if err != nil {
			return err
		}
		if len(tokenUUIDs) > 0 {
			tokenKeys := make([]string, len(tokenUUIDs))
			tokenValues := make([]any, len(tokenUUIDs))
			for i, uuid := range tokenUUIDs {
				tokenKeys[i] = GetTokenKey(uuid)
				tokenValues[i] = uuid
			}
			_ = rs.redisClient.Del(ctx, tokenKeys...).Err()
			_, _ = rs.redisClient.ZRem(ctx, GetUserTokensKey(userID), tokenValues...).Result()
			_ = rs.publishTokenDeleteBatch(ctx, tokenUUIDs)
		}
		_ = rs.redisClient.Del(ctx, platformTokensKey).Err()
	} else {
		userTokensKey := GetUserTokensKey(userID)
		tokenUUIDs, err := rs.redisClient.ZRange(ctx, userTokensKey, 0, -1).Result()
		if err != nil {
			return err
		}
		if len(tokenUUIDs) > 0 {
			tokenKeys := make([]string, len(tokenUUIDs))
			for i, uuid := range tokenUUIDs {
				tokenKeys[i] = GetTokenKey(uuid)
			}
			_ = rs.redisClient.Del(ctx, tokenKeys...).Err()
			_ = rs.publishTokenDeleteBatch(ctx, tokenUUIDs)
		}
		platformKeys := make([]string, 0, constant.BotPlatformID-constant.IOSPlatformID+1)
		for pid := constant.IOSPlatformID; pid <= constant.BotPlatformID; pid++ {
			platformKeys = append(platformKeys, GetPlatformTokenKey(userID, int32(pid)))
		}
		_ = rs.redisClient.Del(ctx, platformKeys...).Err()
		_ = rs.redisClient.Del(ctx, userTokensKey).Err()
	}
	return nil
}

func (rs *RedisStore) CheckTokenExists(ctx context.Context, userID string, platformID int32) (bool, error) {
	now := timex.Unix()
	count, err := rs.redisClient.ZCount(ctx, GetPlatformTokenKey(userID, platformID),
		convert.ToString(now), "9223372036854775807").Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (rs *RedisStore) GetUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	_ = rs.CleanExpiredTokens(ctx, userID)
	userTokensKey := GetUserTokensKey(userID)
	now := timex.Unix()
	zs, err := rs.redisClient.ZRangeByScoreWithScores(ctx, userTokensKey, &goredis.ZRangeBy{
		Min: convert.ToString(now),
		Max: "9223372036854775807",
	}).Result()
	if err != nil {
		return nil, err
	}

	if len(zs) == 0 {
		return nil, ErrTokenNotFound
	}

	var tokens []*TokenInfo
	for _, z := range zs {
		uuid, _ := z.Member.(string)
		data, err := rs.redisClient.Get(ctx, GetTokenKey(uuid)).Result()
		if err != nil || data == "" {
			continue
		}
		var info TokenInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			continue
		}
		tokens = append(tokens, &info)
	}

	if len(tokens) == 0 {
		return nil, ErrTokenNotFound
	}

	return tokens, nil
}

func (rs *RedisStore) GetUserTokensByPlatform(ctx context.Context, userID string, platformID int32) ([]*TokenInfo, error) {
	_ = rs.CleanExpiredTokens(ctx, userID)
	platformTokensKey := GetPlatformTokenKey(userID, platformID)
	now := timex.Unix()
	zs, err := rs.redisClient.ZRangeByScoreWithScores(ctx, platformTokensKey, &goredis.ZRangeBy{
		Min: convert.ToString(now),
		Max: "9223372036854775807",
	}).Result()
	if err != nil {
		return nil, err
	}

	if len(zs) == 0 {
		return nil, ErrTokenNotFound
	}

	var tokens []*TokenInfo
	for _, z := range zs {
		uuid, _ := z.Member.(string)
		data, err := rs.redisClient.Get(ctx, GetTokenKey(uuid)).Result()
		if err != nil || data == "" {
			continue
		}
		var info TokenInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			continue
		}
		tokens = append(tokens, &info)
	}

	if len(tokens) == 0 {
		return nil, ErrTokenNotFound
	}

	return tokens, nil
}

func (rs *RedisStore) publishTokenDelete(ctx context.Context, uuid string) error {
	_, err := rs.redisClient.Publish(ctx, TokenDeleteChannel, uuid).Result()
	return err
}

func (rs *RedisStore) publishTokenDeleteBatch(ctx context.Context, uuids []string) error {
	for _, uuid := range uuids {
		_ = rs.publishTokenDelete(ctx, uuid)
	}
	return nil
}

func (rs *RedisStore) CleanExpiredTokens(ctx context.Context, userID string) error {
	now := timex.Unix()
	maxScore := convert.ToString(now - 1)

	keys := make([]string, 0, constant.BotPlatformID-constant.IOSPlatformID+2)
	keys = append(keys, GetUserTokensKey(userID))
	for pid := constant.IOSPlatformID; pid <= constant.BotPlatformID; pid++ {
		keys = append(keys, GetPlatformTokenKey(userID, int32(pid)))
	}

	for _, key := range keys {
		if err := rs.trimExpiredMembers(ctx, key, maxScore); err != nil {
			return err
		}
	}
	return nil
}

// trimExpiredMembers 删除 ZSet 中 score 已过期的成员；若删除后集合为空，直接删除整个 key。
// key 不存在或类型不是 ZSet 时忽略错误（并发删除/类型异常不应阻塞清理）。
func (rs *RedisStore) trimExpiredMembers(ctx context.Context, key, maxScore string) error {
	if err := rs.redisClient.ZRemRangeByScore(ctx, key, "0", maxScore).Err(); err != nil {
		return nil
	}
	if rs.redisClient.ZCard(ctx, key).Val() == 0 {
		_ = rs.redisClient.Del(ctx, key).Err()
	}
	return nil
}

// CleanExpiredTokensGlobal 扫描所有用户 token 索引 ZSet，清理已过期成员并删除空 key。
// 用于兜底清理历史遗留的僵尸索引（这些 key 在本次 TTL 修复前创建，自身没有过期时间）。
// 使用 SCAN 游标遍历，避免 KEYS 阻塞 Redis；单批失败不中断整体扫描。
func (rs *RedisStore) CleanExpiredTokensGlobal(ctx context.Context) (scanned, removed, deleted int64, err error) {
	now := timex.Unix()
	maxScore := convert.ToString(now - 1)

	iter := rs.redisClient.Scan(ctx, 0, UserTokensPrefix+"*", cleanScanBatchSize).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		scanned++
		n, e := rs.redisClient.ZRemRangeByScore(ctx, key, "0", maxScore).Result()
		if e != nil {
			continue
		}
		removed += n
		if rs.redisClient.ZCard(ctx, key).Val() == 0 {
			if rs.redisClient.Del(ctx, key).Val() > 0 {
				deleted++
			}
		}
	}
	return scanned, removed, deleted, iter.Err()
}

func (rs *RedisStore) ClearAll(ctx context.Context) error {
	keys, err := rs.redisClient.Keys(ctx, fmt.Sprintf("%s*", TokenPrefix)).Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		_ = rs.redisClient.Del(ctx, key).Err()
	}

	userKeys, err := rs.redisClient.Keys(ctx, fmt.Sprintf("%s*", UserTokensPrefix)).Result()
	if err != nil {
		return err
	}
	for _, key := range userKeys {
		_ = rs.redisClient.Del(ctx, key).Err()
	}

	return nil
}
