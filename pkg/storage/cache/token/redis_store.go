package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type RedisStore struct {
	redisClient *redis.Redis
}

func NewRedisStore(redisClient *redis.Redis) *RedisStore {
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
		err = rs.redisClient.SetexCtx(ctx, GetTokenKey(info.UUID), string(data), int(ttl.Seconds()))
	} else {
		err = rs.redisClient.SetCtx(ctx, GetTokenKey(info.UUID), string(data))
	}
	if err != nil {
		return err
	}

	_, err = rs.redisClient.ZaddCtx(ctx, GetUserTokensKey(info.UserID), info.ExpireAt, info.UUID)
	if err != nil {
		return err
	}

	_, err = rs.redisClient.ZaddCtx(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), info.ExpireAt, info.UUID)
	if err != nil {
		return err
	}

	// if ttl > 0 {
	// 	rs.redisClient.ExpireCtx(ctx, GetUserTokensKey(info.UserID), int(ttl.Seconds())+3600)
	// 	rs.redisClient.ExpireCtx(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), int(ttl.Seconds())+3600)
	// }

	return nil
}

func (rs *RedisStore) GetToken(ctx context.Context, uuid string) (*TokenInfo, error) {
	data, err := rs.redisClient.GetCtx(ctx, GetTokenKey(uuid))
	if err != nil || data == "" {
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
	data, err := rs.redisClient.GetCtx(ctx, GetTokenKey(uuid))
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
		data, err := rs.redisClient.GetCtx(ctx, GetTokenKey(uuid))
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
		_, _ = rs.redisClient.DelCtx(ctx, tokenKeys...)

		for _, info := range infos {
			_, _ = rs.redisClient.ZremCtx(ctx, GetUserTokensKey(info.UserID), info.UUID)
			_, _ = rs.redisClient.ZremCtx(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), info.UUID)
			_ = rs.publishTokenDelete(ctx, info.UUID)
		}
	}

	return nil
}

func (rs *RedisStore) deleteTokenInternal(ctx context.Context, info *TokenInfo) error {
	_, err := rs.redisClient.DelCtx(ctx, GetTokenKey(info.UUID))
	if err != nil {
		return err
	}
	_, err = rs.redisClient.ZremCtx(ctx, GetUserTokensKey(info.UserID), info.UUID)
	if err != nil {
		return err
	}

	_, err = rs.redisClient.ZremCtx(ctx, GetPlatformTokenKey(info.UserID, info.PlatformID), info.UUID)
	if err != nil {
		return err
	}

	_ = rs.publishTokenDelete(ctx, info.UUID)

	return nil
}

func (rs *RedisStore) DeleteUserTokens(ctx context.Context, userID string, platformID ...int32) error {
	if len(platformID) > 0 {
		platformTokensKey := GetPlatformTokenKey(userID, platformID[0])
		tokenUUIDs, err := rs.redisClient.ZrangeCtx(ctx, platformTokensKey, 0, -1)
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
			_, _ = rs.redisClient.DelCtx(ctx, tokenKeys...)
			_, _ = rs.redisClient.ZremCtx(ctx, GetUserTokensKey(userID), tokenValues...)
			_ = rs.publishTokenDeleteBatch(ctx, tokenUUIDs)
		}
		_, _ = rs.redisClient.DelCtx(ctx, platformTokensKey)
	} else {
		userTokensKey := GetUserTokensKey(userID)
		tokenUUIDs, err := rs.redisClient.ZrangeCtx(ctx, userTokensKey, 0, -1)
		if err != nil {
			return err
		}
		if len(tokenUUIDs) > 0 {
			tokenKeys := make([]string, len(tokenUUIDs))
			for i, uuid := range tokenUUIDs {
				tokenKeys[i] = GetTokenKey(uuid)
			}
			_, _ = rs.redisClient.DelCtx(ctx, tokenKeys...)
			_ = rs.publishTokenDeleteBatch(ctx, tokenUUIDs)
		}
		platformKeys := make([]string, 0, constant.BotPlatformID-constant.IOSPlatformID+1)
		for pid := constant.IOSPlatformID; pid <= constant.BotPlatformID; pid++ {
			platformKeys = append(platformKeys, GetPlatformTokenKey(userID, int32(pid)))
		}
		_, _ = rs.redisClient.DelCtx(ctx, platformKeys...)
		_, _ = rs.redisClient.DelCtx(ctx, userTokensKey)
	}
	return nil
}

func (rs *RedisStore) CheckTokenExists(ctx context.Context, userID string, platformID int32) (bool, error) {
	now := time.Now().Unix()
	count, err := rs.redisClient.ZcountCtx(ctx, GetPlatformTokenKey(userID, platformID), now, 9223372036854775807)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (rs *RedisStore) GetUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	userTokensKey := GetUserTokensKey(userID)
	now := time.Now().Unix()
	pairs, err := rs.redisClient.ZrangebyscoreWithScoresCtx(ctx, userTokensKey, now, 9223372036854775807)
	if err != nil {
		return nil, err
	}

	if len(pairs) == 0 {
		return nil, ErrTokenNotFound
	}

	var tokens []*TokenInfo
	for _, pair := range pairs {
		data, err := rs.redisClient.Get(GetTokenKey(pair.Key))
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
	pairs, err := rs.redisClient.ZrangebyscoreWithScoresCtx(ctx, platformTokensKey, now, 9223372036854775807)
	if err != nil {
		return nil, err
	}

	if len(pairs) == 0 {
		return nil, ErrTokenNotFound
	}

	var tokens []*TokenInfo
	for _, pair := range pairs {
		data, err := rs.redisClient.Get(GetTokenKey(pair.Key))
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
	_, err := rs.redisClient.PublishCtx(ctx, TokenDeleteChannel, uuid)
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

	_, _ = rs.redisClient.ZremrangebyscoreCtx(ctx, GetUserTokensKey(userID), 0, now-1)
	for pid := constant.IOSPlatformID; pid <= constant.BotPlatformID; pid++ {
		_, _ = rs.redisClient.ZremrangebyscoreCtx(ctx, GetPlatformTokenKey(userID, int32(pid)), 0, now-1)
	}
	return nil
}

func (rs *RedisStore) ClearAll(ctx context.Context) error {
	keys, err := rs.redisClient.KeysCtx(ctx, fmt.Sprintf("%s*", TokenPrefix))
	if err != nil {
		return err
	}
	for _, key := range keys {
		_, _ = rs.redisClient.DelCtx(ctx, key)
	}

	userKeys, err := rs.redisClient.KeysCtx(ctx, fmt.Sprintf("%s*", UserTokensPrefix))
	if err != nil {
		return err
	}
	for _, key := range userKeys {
		_, _ = rs.redisClient.DelCtx(ctx, key)
	}

	return nil
}
