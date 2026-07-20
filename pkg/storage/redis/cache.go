package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	NilMarker = "null"
)

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
	return rdb.Del(ctx, keys...).Err()
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
