package token

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

type TokenWithExpire struct {
	Info     *TokenInfo
	ExpireAt int64
}

type LocalStore struct {
	tokens         sync.Map // token:{uuid} -> *TokenWithExpire
	userTokens     sync.Map // user:{userID}:tokens -> map[string]*TokenWithExpire
	platformTokens sync.Map // user:{userID}:platform:{pid}:tokens -> map[string]*TokenWithExpire
	stopCleanup    chan struct{}
	cleanupRunning atomic.Bool
}

var _ TokenStore = (*LocalStore)(nil)

func NewLocalStore() *LocalStore {
	ls := &LocalStore{}
	ls.startCleanup()
	return ls
}

func (ls *LocalStore) startCleanup() {
	if ls.cleanupRunning.CompareAndSwap(false, true) {
		ls.stopCleanup = make(chan struct{})
		go ls.periodicCleanup()
	}
}

func (ls *LocalStore) Close() error {
	if ls.cleanupRunning.CompareAndSwap(true, false) {
		close(ls.stopCleanup)
	}
	return nil
}

func (ls *LocalStore) periodicCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ls.cleanupExpiredTokens()
		case <-ls.stopCleanup:
			return
		}
	}
}

func (ls *LocalStore) cleanupExpiredTokens() {
	now := time.Now().Unix()

	ls.tokens.Range(func(key, value interface{}) bool {
		if tw, ok := value.(*TokenWithExpire); ok {
			if tw.ExpireAt > 0 && tw.ExpireAt < now {
				ls.tokens.Delete(key)
				ls.removeFromUserTokens(tw.Info.UserID, tw.Info.UUID)
				ls.removeFromPlatformTokens(tw.Info.UserID, tw.Info.PlatformID, tw.Info.UUID)
			}
		}
		return true
	})
}

func (ls *LocalStore) StoreToken(ctx context.Context, info *TokenInfo) error {
	if info.ExpireAt > 0 {
		ttl := time.Until(time.Unix(info.ExpireAt, 0))
		if ttl <= 0 {
			return ErrTokenExpired
		}
	}

	tw := &TokenWithExpire{
		Info:     info,
		ExpireAt: info.ExpireAt,
	}

	ls.tokens.Store(GetTokenKey(info.UUID), tw)
	ls.addToUserTokens(info.UserID, tw)
	ls.addToPlatformTokens(info.UserID, info.PlatformID, tw)

	return nil
}

func (ls *LocalStore) GetToken(ctx context.Context, uuid string) (*TokenInfo, error) {
	val, found := ls.tokens.Load(GetTokenKey(uuid))
	if !found {
		return nil, ErrTokenNotFound
	}
	tw, ok := val.(*TokenWithExpire)
	if !ok {
		return nil, ErrTokenInvalid
	}
	if tw.ExpireAt > 0 && tw.ExpireAt < time.Now().Unix() {
		ls.DeleteToken(ctx, uuid)
		return nil, ErrTokenExpired
	}
	return tw.Info, nil
}

func (ls *LocalStore) DeleteToken(ctx context.Context, uuid string) error {
	val, found := ls.tokens.LoadAndDelete(GetTokenKey(uuid))
	if !found {
		return nil
	}
	tw, ok := val.(*TokenWithExpire)
	if !ok {
		return nil
	}

	ls.removeFromUserTokens(tw.Info.UserID, uuid)
	ls.removeFromPlatformTokens(tw.Info.UserID, tw.Info.PlatformID, uuid)

	return nil
}

func (ls *LocalStore) DeleteTokens(ctx context.Context, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}

	for _, uuid := range uuids {
		val, found := ls.tokens.LoadAndDelete(GetTokenKey(uuid))
		if !found {
			continue
		}
		tw, ok := val.(*TokenWithExpire)
		if !ok {
			continue
		}

		ls.removeFromUserTokens(tw.Info.UserID, uuid)
		ls.removeFromPlatformTokens(tw.Info.UserID, tw.Info.PlatformID, uuid)
	}

	return nil
}

func (ls *LocalStore) DeleteUserTokens(ctx context.Context, userID string, platformID ...int32) error {
	if len(platformID) > 0 {
		platformKey := GetPlatformTokenKey(userID, platformID[0])
		if val, found := ls.platformTokens.LoadAndDelete(platformKey); found {
			if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
				for _, tw := range tokenMap {
					ls.tokens.Delete(GetTokenKey(tw.Info.UUID))
					ls.removeFromUserTokens(userID, tw.Info.UUID)
				}
			}
		}
	} else {
		userKey := GetUserTokensKey(userID)
		if val, found := ls.userTokens.LoadAndDelete(userKey); found {
			if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
				for _, tw := range tokenMap {
					ls.tokens.Delete(GetTokenKey(tw.Info.UUID))
				}
			}
		}
		for pid := constant.IOSPlatformID; pid <= constant.BotPlatformID; pid++ {
			ls.platformTokens.Delete(GetPlatformTokenKey(userID, int32(pid)))
		}
	}
	return nil
}

func (ls *LocalStore) CheckTokenExists(ctx context.Context, userID string, platformID int32) (bool, error) {
	platformKey := GetPlatformTokenKey(userID, platformID)
	if val, found := ls.platformTokens.Load(platformKey); found {
		if tokenMap, ok := val.(map[string]*TokenWithExpire); ok && len(tokenMap) > 0 {
			now := time.Now().Unix()
			for _, tw := range tokenMap {
				if tw.ExpireAt <= 0 || tw.ExpireAt >= now {
					return true, nil
				}
			}
			// 所有 token 都已过期，清理
			ls.cleanExpiredPlatformTokens(userID, platformID)
		}
	}
	return false, nil
}

func (ls *LocalStore) GetUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	userKey := GetUserTokensKey(userID)
	if val, found := ls.userTokens.Load(userKey); found {
		if tokenMap, ok := val.(map[string]*TokenWithExpire); ok && len(tokenMap) > 0 {
			var tokens []*TokenInfo
			var expired []string
			now := time.Now().Unix()

			for uuid, tw := range tokenMap {
				if tw.ExpireAt <= 0 || tw.ExpireAt >= now {
					tokens = append(tokens, tw.Info)
				} else {
					expired = append(expired, uuid)
				}
			}

			// 清理过期 token
			for _, uuid := range expired {
				ls.removeFromUserTokens(userID, uuid)
				ls.tokens.Delete(GetTokenKey(uuid))
			}

			if len(tokens) > 0 {
				return tokens, nil
			}
		}
	}
	return nil, ErrTokenNotFound
}

func (ls *LocalStore) GetUserTokensByPlatform(ctx context.Context, userID string, platformID int32) ([]*TokenInfo, error) {
	platformKey := GetPlatformTokenKey(userID, platformID)
	if val, found := ls.platformTokens.Load(platformKey); found {
		if tokenMap, ok := val.(map[string]*TokenWithExpire); ok && len(tokenMap) > 0 {
			var tokens []*TokenInfo
			var expired []string
			now := time.Now().Unix()

			for uuid, tw := range tokenMap {
				if tw.ExpireAt <= 0 || tw.ExpireAt >= now {
					tokens = append(tokens, tw.Info)
				} else {
					expired = append(expired, uuid)
				}
			}

			for _, uuid := range expired {
				ls.removeFromPlatformTokens(userID, platformID, uuid)
				ls.tokens.Delete(GetTokenKey(uuid))
			}

			if len(tokens) > 0 {
				return tokens, nil
			}
		}
	}
	return nil, ErrTokenNotFound
}

func (ls *LocalStore) addToUserTokens(userID string, tw *TokenWithExpire) {
	userKey := GetUserTokensKey(userID)
	val, _ := ls.userTokens.LoadOrStore(userKey, make(map[string]*TokenWithExpire))
	if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
		tokenMap[tw.Info.UUID] = tw
	}
}

func (ls *LocalStore) removeFromUserTokens(userID string, uuid string) {
	userKey := GetUserTokensKey(userID)
	if val, found := ls.userTokens.Load(userKey); found {
		if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
			delete(tokenMap, uuid)
			if len(tokenMap) == 0 {
				ls.userTokens.Delete(userKey)
			}
		}
	}
}

func (ls *LocalStore) addToPlatformTokens(userID string, platformID int32, tw *TokenWithExpire) {
	platformKey := GetPlatformTokenKey(userID, platformID)
	val, _ := ls.platformTokens.LoadOrStore(platformKey, make(map[string]*TokenWithExpire))
	if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
		tokenMap[tw.Info.UUID] = tw
	}
}

func (ls *LocalStore) removeFromPlatformTokens(userID string, platformID int32, uuid string) {
	platformKey := GetPlatformTokenKey(userID, platformID)
	if val, found := ls.platformTokens.Load(platformKey); found {
		if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
			delete(tokenMap, uuid)
			if len(tokenMap) == 0 {
				ls.platformTokens.Delete(platformKey)
			}
		}
	}
}

func (ls *LocalStore) cleanExpiredPlatformTokens(userID string, platformID int32) {
	platformKey := GetPlatformTokenKey(userID, platformID)
	if val, found := ls.platformTokens.Load(platformKey); found {
		if tokenMap, ok := val.(map[string]*TokenWithExpire); ok {
			now := time.Now().Unix()
			for uuid, tw := range tokenMap {
				if tw.ExpireAt > 0 && tw.ExpireAt < now {
					delete(tokenMap, uuid)
					ls.tokens.Delete(GetTokenKey(uuid))
					ls.removeFromUserTokens(userID, uuid)
				}
			}
			if len(tokenMap) == 0 {
				ls.platformTokens.Delete(platformKey)
			}
		}
	}
}
