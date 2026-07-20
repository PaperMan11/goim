package allocator

import (
	"context"
	"embed"
	"errors"
	"strconv"
	"time"

	"github.com/PaperMan11/goim/pkg/utils/convert"
	"github.com/PaperMan11/goim/pkg/utils/timex"
	goredis "github.com/redis/go-redis/v9"
)

//go:embed lua/*.lua
var luaScripts embed.FS

var (
	ErrRetryExhausted = errors.New("retry exhausted")
	ErrLockTimeout    = errors.New("lock timeout")
)

type SeqAllocator interface {
	Allocate(ctx context.Context, conversationID string) (int64, error)
	AllocateBatch(ctx context.Context, conversationID string, count int) (start, end int64, err error)
	GetCurrent(ctx context.Context, conversationID string) (int64, error)
	GetSeqRange(ctx context.Context, conversationID string) (curr, last int64, err error)
	Set(ctx context.Context, conversationID string, value int64) error
	Reset(ctx context.Context, conversationID string) error
	SyncFromDB(ctx context.Context, conversationID string, getMaxSeqFn func(ctx context.Context, conversationID string) (int64, error)) error
}

type RedisSeqAllocator struct {
	redisClient    goredis.UniversalClient
	allocateSeq    string
	commitSeq      string
	allocateScript *goredis.Script
	commitScript   *goredis.Script
	poolSize       int
	lockSecond     int
	dataSecond     int
	getMaxSeqFn    GetMaxSeqFn
	maxRetries     int
	retryInterval  time.Duration
	retryBackoff   bool
}

type GetMaxSeqFn func(ctx context.Context, conversationID string) (int64, error)

type RedisSeqAllocatorOption func(*RedisSeqAllocator)

func WithPoolSize(poolSize int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.poolSize = poolSize
	}
}

func WithLockSecond(lockSecond int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.lockSecond = lockSecond
	}
}

func WithDataSecond(dataSecond int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.dataSecond = dataSecond
	}
}

func WithGetMaxSeqFn(fn GetMaxSeqFn) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.getMaxSeqFn = fn
	}
}

func WithMaxRetries(maxRetries int) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.maxRetries = maxRetries
	}
}

func WithRetryInterval(retryInterval time.Duration) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.retryInterval = retryInterval
	}
}

func WithRetryBackoff(enable bool) RedisSeqAllocatorOption {
	return func(r *RedisSeqAllocator) {
		r.retryBackoff = enable
	}
}

func NewRedisSeqAllocator(redisClient goredis.UniversalClient, options ...RedisSeqAllocatorOption) (*RedisSeqAllocator, error) {
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
		allocateScript: goredis.NewScript(string(allocateSeq)),
		commitScript:   goredis.NewScript(string(commitSeq)),
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

func (a *RedisSeqAllocator) Allocate(ctx context.Context, conversationID string) (int64, error) {
	start, _, err := a.AllocateBatch(ctx, conversationID, 1)
	return start, err
}

func (a *RedisSeqAllocator) AllocateBatch(ctx context.Context, conversationID string, count int) (start, end int64, err error) {
	if count <= 0 {
		return 0, 0, nil
	}

	key := GetMessageSeqKey(conversationID)
	mallocTime := convert.ToString(timex.UnixMilli())

	retryCount := 0
	backoffInterval := a.retryInterval

	for {
		result, err := a.allocateScript.Run(ctx, a.redisClient, []string{key},
			count, a.lockSecond, a.dataSecond, mallocTime).Result()
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

func (a *RedisSeqAllocator) GetCurrent(ctx context.Context, conversationID string) (int64, error) {
	curr, _, err := a.GetSeqRange(ctx, conversationID)
	return curr, err
}

func (a *RedisSeqAllocator) GetSeqRange(ctx context.Context, conversationID string) (curr, last int64, err error) {
	key := GetMessageSeqKey(conversationID)
	mallocTime := strconv.FormatInt(timex.UnixMilli(), 10)

	retryCount := 0
	backoffInterval := a.retryInterval

	for {
		result, err := a.allocateScript.Run(ctx, a.redisClient, []string{key},
			0, a.lockSecond, a.dataSecond, mallocTime).Result()
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

func (a *RedisSeqAllocator) Set(ctx context.Context, conversationID string, value int64) error {
	key := GetMessageSeqKey(conversationID)
	mallocTime := convert.ToString(timex.UnixMilli())

	_, err := a.commitScript.Run(ctx, a.redisClient, []string{key},
		"", a.dataSecond, value, value+int64(a.poolSize), mallocTime).Result()
	return err
}

func (a *RedisSeqAllocator) Reset(ctx context.Context, conversationID string) error {
	key := GetMessageSeqKey(conversationID)
	return a.redisClient.Del(ctx, key).Err()
}

func (a *RedisSeqAllocator) SyncFromDB(ctx context.Context, conversationID string, getMaxSeqFn func(ctx context.Context, conversationID string) (int64, error)) error {
	key := GetMessageSeqKey(conversationID)
	mallocTime := convert.ToString(timex.UnixMilli())

	retryCount := 0
	backoffInterval := a.retryInterval

	for {
		result, err := a.allocateScript.Run(ctx, a.redisClient, []string{key},
			0, a.lockSecond, a.dataSecond, mallocTime).Result()
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
	_, err := a.commitScript.Run(ctx, a.redisClient, []string{key},
		lockValue, a.dataSecond, dbMaxSeq, newLastSeq, mallocTime).Result()
	return err
}

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
	_, err := a.commitScript.Run(ctx, a.redisClient, []string{key},
		lockValue, a.dataSecond, dbMaxSeq, newLastSeq, mallocTime).Result()
	return err
}
