package redis

import (
	"crypto/tls"
	"errors"
	"strings"

	"github.com/redis/go-redis/v9"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"
)

var (
	ErrEmptyHost = errors.New("empty redis host")
	ErrEmptyType = errors.New("empty redis type")
)

func MustNewRedis(conf zredis.RedisConf) redis.UniversalClient {
	client, err := NewRedisClient(conf)
	if err != nil {
		panic(err)
	}
	return client
}

func NewRedisClient(conf zredis.RedisConf) (redis.UniversalClient, error) {
	if conf.Host == "" {
		return nil, ErrEmptyHost
	}
	if conf.Type == "" {
		return nil, ErrEmptyType
	}

	switch conf.Type {
	case "cluster":
		return redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:     parseAddrs(conf.Host),
			Username:  conf.User,
			Password:  conf.Pass,
			TLSConfig: buildTLSConfig(conf.Tls),
		}), nil
	default:
		return redis.NewClient(&redis.Options{
			Addr:      conf.Host,
			Username:  conf.User,
			Password:  conf.Pass,
			DB:        0,
			TLSConfig: buildTLSConfig(conf.Tls),
		}), nil
	}
}

func parseAddrs(host string) []string {
	if host == "" {
		return nil
	}
	parts := strings.Split(host, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if addr := strings.TrimSpace(part); addr != "" {
			result = append(result, addr)
		}
	}
	return result
}

func buildTLSConfig(enable bool) *tls.Config {
	if !enable {
		return nil
	}
	return &tls.Config{}
}
