package versionlog

import (
	"context"
	"errors"

	"github.com/PaperMan11/goim/pkg/storage/model"
	sredis "github.com/PaperMan11/goim/pkg/storage/redis"
	"github.com/PaperMan11/goim/pkg/utils/randx"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/mon"
	"github.com/zeromicro/go-zero/core/syncx"
)

type cachedVersionLogModel struct {
	VersionLogModel
	redis   goredis.UniversalClient
	barrier syncx.SingleFlight
}

// NewCachedVersionLogModel 包装一个带 Redis 缓存的版本日志模型。
// 若 rdb 为 nil 则退化为直读内层模型。
func NewCachedVersionLogModel(inner VersionLogModel, rdb goredis.UniversalClient, barrier syncx.SingleFlight) VersionLogModel {
	return &cachedVersionLogModel{
		VersionLogModel: inner,
		redis:           rdb,
		barrier:         barrier,
	}
}

// NewCachedVersionLogModelFromMongo 便捷构造：用默认 mongo 实现作为内层。
func NewCachedVersionLogModelFromMongo(versionMod *mon.Model, rdb goredis.UniversalClient, barrier syncx.SingleFlight) VersionLogModel {
	return NewCachedVersionLogModel(NewVersionLogModel(versionMod), rdb, barrier)
}

func (m *cachedVersionLogModel) jitterTTL(baseSeconds int) int {
	return randx.JitterInt(baseSeconds, ttlJitterRatioPct)
}

func (m *cachedVersionLogModel) IncrVersionLog(ctx context.Context, did, eid string, state int32) (*model.VersionLog, error) {
	ver, err := m.VersionLogModel.IncrVersionLog(ctx, did, eid, state)
	if err != nil {
		return nil, err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetVersionLogKey(did))
	return ver, nil
}

func (m *cachedVersionLogModel) IncrVersionLogBatch(ctx context.Context, did string, eids []string, state int32) (*model.VersionLog, error) {
	ver, err := m.VersionLogModel.IncrVersionLogBatch(ctx, did, eids, state)
	if err != nil {
		return nil, err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetVersionLogKey(did))
	return ver, nil
}

func (m *cachedVersionLogModel) GetVersionLog(ctx context.Context, did string) (*model.VersionLog, error) {
	if m.redis == nil {
		return m.VersionLogModel.GetVersionLog(ctx, did)
	}

	var ver model.VersionLog
	key := GetVersionLogKey(did)
	found, err := sredis.CacheGet(ctx, m.redis, key, &ver)
	if err != nil {
		return nil, err
	}
	if found {
		if ver.DID == "" {
			return nil, ErrVersionLogNotFound
		}
		return &ver, nil
	}

	sfKey := sfKeyPrefixVersion + did
	v, err := m.barrier.Do(sfKey, func() (any, error) {
		var innerVer model.VersionLog
		found2, err2 := sredis.CacheGet(ctx, m.redis, key, &innerVer)
		if err2 != nil {
			return nil, err2
		}
		if found2 {
			if innerVer.DID == "" {
				return nil, ErrVersionLogNotFound
			}
			return &innerVer, nil
		}

		dbVer, err2 := m.VersionLogModel.GetVersionLog(ctx, did)
		if err2 != nil {
			if errors.Is(err2, ErrVersionLogNotFound) {
				_, _ = sredis.CacheSetCAS(ctx, m.redis, key, nil, 0, m.jitterTTL(nilExpireSeconds))
			}
			return nil, err2
		}
		version := dbVer.LastUpdate.UnixMilli()
		if version <= 0 {
			version = timex.UnixMilli()
		}
		_, _ = sredis.CacheSetCAS(ctx, m.redis, key, dbVer, version, m.jitterTTL(defaultExpireSeconds))
		return dbVer, nil
	})
	if err != nil {
		if errors.Is(err, ErrVersionLogNotFound) {
			return nil, ErrVersionLogNotFound
		}
		return nil, err
	}
	if v == nil {
		return nil, ErrVersionLogNotFound
	}
	return v.(*model.VersionLog), nil
}

func (m *cachedVersionLogModel) DeleteVersionLog(ctx context.Context, did string) error {
	err := m.VersionLogModel.DeleteVersionLog(ctx, did)
	if err != nil {
		return err
	}
	sredis.CacheDelDouble(ctx, m.redis, GetVersionLogKey(did))
	return nil
}
