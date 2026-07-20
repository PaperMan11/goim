package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	goredis "github.com/redis/go-redis/v9"
)

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

	return nil
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

	if info.ExpireAt > 0 && time.Now().Unix() > info.ExpireAt {
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
	now := time.Now().Unix()
	count, err := rs.redisClient.ZCount(ctx, GetPlatformTokenKey(userID, platformID),
		strconv.FormatInt(now, 10), "9223372036854775807").Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (rs *RedisStore) GetUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	userTokensKey := GetUserTokensKey(userID)
	now := time.Now().Unix()
	zs, err := rs.redisClient.ZRangeByScoreWithScores(ctx, userTokensKey, &goredis.ZRangeBy{
		Min: strconv.FormatInt(now, 10),
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
	platformTokensKey := GetPlatformTokenKey(userID, platformID)
	now := time.Now().Unix()
	zs, err := rs.redisClient.ZRangeByScoreWithScores(ctx, platformTokensKey, &goredis.ZRangeBy{
		Min: strconv.FormatInt(now, 10),
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
	now := time.Now().Unix()
	nowStr := strconv.FormatInt(now-1, 10)

	_, _ = rs.redisClient.ZRemRangeByScore(ctx, GetUserTokensKey(userID), "0", nowStr).Result()
	for pid := constant.IOSPlatformID; pid <= constant.BotPlatformID; pid++ {
		_, _ = rs.redisClient.ZRemRangeByScore(ctx, GetPlatformTokenKey(userID, int32(pid)), "0", nowStr).Result()
	}
	return nil
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
