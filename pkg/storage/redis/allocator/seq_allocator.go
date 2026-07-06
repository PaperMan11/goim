package allocator

import (
	"context"
	"embed"
	"errors"
	"strconv"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/convert"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"
)

//go:embed lua/*.lua
var luaScripts embed.FS

var (
	ErrRetryExhausted = errors.New("retry exhausted")
	ErrLockTimeout    = errors.New("lock timeout")
)

// SeqAllocator 分布式序列号分配器接口
// 基于预分配池模式，减少 Redis 写次数，提高高并发性能
type SeqAllocator interface {
	Allocate(ctx context.Context, conversationID string) (int64, error)
	AllocateBatch(ctx context.Context, conversationID string, count int) (start, end int64, err error)
	GetCurrent(ctx context.Context, conversationID string) (int64, error)
	GetSeqRange(ctx context.Context, conversationID string) (curr, last int64, err error)
	Set(ctx context.Context, conversationID string, value int64) error
	Reset(ctx context.Context, conversationID string) error
	SyncFromDB(ctx context.Context, conversationID string, getMaxSeqFn func(ctx context.Context, conversationID string) (int64, error)) error
}

// RedisSeqAllocator Redis 序列号分配器实现
// 使用预分配池模式：维护 [CURR, LAST] 区间作为可用序列号池
// 当 CURR + size <= LAST 时，直接从池中分配，无需写数据库
// 当 CURR + size > LAST 时，触发扩容流程
type RedisSeqAllocator struct {
	redisClient    *zredis.Redis
	allocateSeq    string         // 分配脚本
	commitSeq      string         // 提交脚本
	allocateScript *zredis.Script // 分配脚本
	commitScript   *zredis.Script // 提交脚本
	poolSize       int            // 预分配池大小
	lockSecond     int            // 锁过期时间（秒）
	dataSecond     int            // 数据过期时间（秒）
	getMaxSeqFn    GetMaxSeqFn    // 获取最大序列号回调
	maxRetries     int            // 最大重试次数
	retryInterval  time.Duration  // 重试间隔
	retryBackoff   bool           // 是否启用指数退避
}

// GetMaxSeqFn 获取最大序列号的回调函数
// 用于从数据库同步初始值或扩容时获取新的边界
type GetMaxSeqFn func(ctx context.Context, conversationID string) (int64, error)

// RedisSeqAllocatorOption RedisSeqAllocator 配置选项
type RedisSeqAllocatorOption func(*RedisSeqAllocator)

// WithPoolSize 设置预分配池大小，默认 1000
func WithPoolSize(poolSize int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.poolSize = poolSize
	}
}

// WithLockSecond 设置锁过期时间（秒），默认 30
func WithLockSecond(lockSecond int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.lockSecond = lockSecond
	}
}

// WithDataSecond 设置数据过期时间（秒），默认 86400（24小时）
func WithDataSecond(dataSecond int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.dataSecond = dataSecond
	}
}

// WithGetMaxSeqFn 设置获取最大序列号的回调函数
func WithGetMaxSeqFn(fn GetMaxSeqFn) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.getMaxSeqFn = fn
	}
}

// WithMaxRetries 设置最大重试次数，默认 10
// 当被其他节点锁定时，会重试最多 maxRetries 次
func WithMaxRetries(maxRetries int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.maxRetries = maxRetries
	}
}

// WithRetryInterval 设置重试间隔，默认 50ms
func WithRetryInterval(retryInterval time.Duration) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.retryInterval = retryInterval
	}
}

// WithRetryBackoff 启用指数退避，重试间隔会翻倍增长
// 例如：50ms → 100ms → 200ms → ...
func WithRetryBackoff(enable bool) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.retryBackoff = enable
	}
}

// NewRedisSeqAllocator 创建 RedisSeqAllocator 实例
// redisClient: Redis 客户端
// options: 配置选项
func NewRedisSeqAllocator(redisClient *zredis.Redis, options ...RedisSeqAllocatorOption) (*RedisSeqAllocator, error) {
	allocateSeq, err := luaScripts.ReadFile("lua/allocate_seq.lua")
	if err != nil {
		return nil, err
	}
	commitSeq, err := luaScripts.ReadFile("lua/commit_seq.lua")
	if err != nil {
		return nil, err
	}

	allocator := &RedisSeqAllocator{
		redisClient:    redisClient,
		allocateSeq:    string(allocateSeq),
		commitSeq:      string(commitSeq),
		allocateScript: zredis.NewScript(string(allocateSeq)),
		commitScript:   zredis.NewScript(string(commitSeq)),
		poolSize:       1000,
		lockSecond:     30,
		dataSecond:     86400,
		maxRetries:     10,
		retryInterval:  50 * time.Millisecond,
		retryBackoff:   true,
	}

	for _, option := range options {
		option(allocator)
	}

	return allocator, nil
}

// Allocate 分配单个序列号
func (a *RedisSeqAllocator) Allocate(ctx context.Context, conversationID string) (int64, error) {
	start, _, err := a.AllocateBatch(ctx, conversationID, 1)
	return start, err
}

// AllocateBatch 批量分配多个连续序列号
// 返回起始序列号和结束序列号 [start, end]，包含两端
func (a *RedisSeqAllocator) AllocateBatch(ctx context.Context, conversationID string, count int) (start, end int64, err error) {
	if count <= 0 {
		return 0, 0, nil
	}

	key := GetMessageSeqKey(conversationID)
	mallocTime := convert.ToString(time.Now().UnixMilli())

	retryCount := 0
	backoffInterval := a.retryInterval

	for {
		result, err := a.redisClient.ScriptRunCtx(ctx, a.allocateScript, []string{key},
			[]interface{}{count, a.lockSecond, a.dataSecond, mallocTime})
		if err != nil {
			return 0, 0, err
		}

		resultArray, ok := result.([]interface{})
		if !ok {
			return 0, 0, errors.New("invalid result format")
		}

		returnCode := convert.ToInt64(resultArray[0])
		switch returnCode {
		case 0:
			currSeq := convert.ToInt64(resultArray[1])
			return currSeq + 1, currSeq + int64(count), nil

		case 1:
			if err := a.syncFromDBWithLock(ctx, key, resultArray); err != nil {
				return 0, 0, err
			}

		case 2:
			retryCount++
			if retryCount >= a.maxRetries {
				return 0, 0, ErrRetryExhausted
			}
			time.Sleep(backoffInterval)
			if a.retryBackoff {
				backoffInterval *= 2
				if backoffInterval > time.Second*2 {
					backoffInterval = time.Second * 2
				}
			}

		case 3:
			if err := a.expandPool(ctx, key, resultArray); err != nil {
				return 0, 0, err
			}

		default:
			return 0, 0, errors.New("unknown return code")
		}
	}
}

// GetCurrent 获取当前序列号（CURR）
func (a *RedisSeqAllocator) GetCurrent(ctx context.Context, conversationID string) (int64, error) {
	curr, _, err := a.GetSeqRange(ctx, conversationID)
	return curr, err
}

// GetSeqRange 获取序列号范围（CURR 和 LAST）
func (a *RedisSeqAllocator) GetSeqRange(ctx context.Context, conversationID string) (curr, last int64, err error) {
	key := GetMessageSeqKey(conversationID)
	mallocTime := strconv.FormatInt(time.Now().UnixMilli(), 10)

	retryCount := 0
	backoffInterval := a.retryInterval

	for {
		result, err := a.redisClient.ScriptRunCtx(ctx, a.allocateScript, []string{key},
			[]interface{}{0, a.lockSecond, a.dataSecond, mallocTime})
		if err != nil {
			return 0, 0, err
		}

		resultArray, ok := result.([]interface{})
		if !ok {
			return 0, 0, errors.New("invalid result format")
		}

		returnCode := convert.ToInt64(resultArray[0])
		switch returnCode {
		case 0:
			curr = convert.ToInt64(resultArray[1])
			last = convert.ToInt64(resultArray[2])
			return curr, last, nil

		case 1:
			if err := a.syncFromDBWithLock(ctx, key, resultArray); err != nil {
				return 0, 0, err
			}
			continue

		case 2:
			retryCount++
			if retryCount >= a.maxRetries {
				return 0, 0, ErrRetryExhausted
			}
			time.Sleep(backoffInterval)
			if a.retryBackoff {
				backoffInterval *= 2
				if backoffInterval > time.Second*2 {
					backoffInterval = time.Second * 2
				}
			}

		default:
			return 0, 0, errors.New("unknown return code")
		}
	}
}

// Set 设置序列号（直接覆盖）
func (a *RedisSeqAllocator) Set(ctx context.Context, conversationID string, value int64) error {
	key := GetMessageSeqKey(conversationID)
	// mallocTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mallocTime := convert.ToString(time.Now().UnixMilli())

	_, err := a.redisClient.ScriptRunCtx(ctx, a.commitScript, []string{key},
		[]interface{}{"", a.dataSecond, value, value + int64(a.poolSize), mallocTime})
	return err
}

// Reset 重置序列号（删除 key）
func (a *RedisSeqAllocator) Reset(ctx context.Context, conversationID string) error {
	key := GetMessageSeqKey(conversationID)
	_, err := a.redisClient.DelCtx(ctx, key)
	return err
}

// SyncFromDB 从数据库同步初始值
func (a *RedisSeqAllocator) SyncFromDB(ctx context.Context, conversationID string, getMaxSeqFn func(ctx context.Context, conversationID string) (int64, error)) error {
	key := GetMessageSeqKey(conversationID)
	// mallocTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mallocTime := convert.ToString(time.Now().UnixMilli())

	retryCount := 0
	backoffInterval := a.retryInterval

	for {
		result, err := a.redisClient.ScriptRunCtx(ctx, a.allocateScript, []string{key},
			[]interface{}{0, a.lockSecond, a.dataSecond, mallocTime})
		if err != nil {
			return err
		}

		resultArray, ok := result.([]interface{})
		if !ok {
			return errors.New("invalid result format")
		}

		returnCode := convert.ToInt64(resultArray[0])
		switch returnCode {
		case 0:
			return nil

		case 1:
			oldFn := a.getMaxSeqFn
			a.getMaxSeqFn = getMaxSeqFn
			defer func() { a.getMaxSeqFn = oldFn }()

			return a.syncFromDBWithLock(ctx, key, resultArray)

		case 2:
			retryCount++
			if retryCount >= a.maxRetries {
				return ErrRetryExhausted
			}
			time.Sleep(backoffInterval)
			if a.retryBackoff {
				backoffInterval *= 2
				if backoffInterval > time.Second*2 {
					backoffInterval = time.Second * 2
				}
			}

		default:
			return errors.New("unknown return code")
		}
	}
}

// syncFromDBWithLock 从数据库同步初始值（带锁）
func (a *RedisSeqAllocator) syncFromDBWithLock(ctx context.Context, key string, resultArray []interface{}) error {
	if len(resultArray) < 3 {
		return errors.New("invalid result array length")
	}

	lockValue := convert.ToInt64(resultArray[1])
	mallocTime := convert.ToInt64(resultArray[2])

	var dbMaxSeq int64
	if a.getMaxSeqFn != nil {
		var err error
		dbMaxSeq, err = a.getMaxSeqFn(ctx, key[len(MessageSeqPrefix):])
		if err != nil {
			return err
		}
	}

	newLastSeq := dbMaxSeq + int64(a.poolSize)
	_, err := a.redisClient.ScriptRunCtx(ctx, a.commitScript, []string{key},
		[]interface{}{lockValue, a.dataSecond, dbMaxSeq, newLastSeq, mallocTime})
	return err
}

// expandPool 扩容预分配池
func (a *RedisSeqAllocator) expandPool(ctx context.Context, key string, resultArray []interface{}) error {
	if len(resultArray) < 5 {
		return errors.New("invalid result array length")
	}

	lockValue := convert.ToInt64(resultArray[3])
	mallocTime := convert.ToInt64(resultArray[4])

	var dbMaxSeq int64
	if a.getMaxSeqFn != nil {
		var err error
		dbMaxSeq, err = a.getMaxSeqFn(ctx, key[len(MessageSeqPrefix):])
		if err != nil {
			return err
		}
	}

	newLastSeq := dbMaxSeq + int64(a.poolSize)
	_, err := a.redisClient.ScriptRunCtx(ctx, a.commitScript, []string{key},
		[]interface{}{lockValue, a.dataSecond, dbMaxSeq, newLastSeq, mallocTime})
	return err
}
