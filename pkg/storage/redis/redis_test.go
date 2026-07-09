package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	zredis "github.com/zeromicro/go-zero/core/stores/redis"
)

func TestNewRedisClient_NodeMode(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "192.168.1.1:6379",
		Type: "node",
		Pass: "password",
		User: "user",
		Tls:  false,
	}

	client, err := NewRedisClient(conf)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestNewRedisClient_ClusterMode(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "192.168.1.1:6379,192.168.1.2:6379,192.168.1.3:6379",
		Type: "cluster",
		Pass: "password",
		User: "user",
		Tls:  false,
	}

	client, err := NewRedisClient(conf)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestNewRedisClient_ClusterModeSingleAddr(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "192.168.1.1:6379",
		Type: "cluster",
		Pass: "",
		User: "",
		Tls:  false,
	}

	client, err := NewRedisClient(conf)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestNewRedisClient_EmptyHost(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "",
		Type: "node",
	}

	client, err := NewRedisClient(conf)
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyHost, err)
	assert.Nil(t, client)
}

func TestNewRedisClient_EmptyType(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "192.168.1.1:6379",
		Type: "",
	}

	client, err := NewRedisClient(conf)
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyType, err)
	assert.Nil(t, client)
}

func TestNewRedisClient_TLSEnabled(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "192.168.1.1:6379",
		Type: "node",
		Tls:  true,
	}

	client, err := NewRedisClient(conf)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestNewRedisClient_TLSDisabled(t *testing.T) {
	conf := zredis.RedisConf{
		Host: "192.168.1.1:6379",
		Type: "node",
		Tls:  false,
	}

	client, err := NewRedisClient(conf)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Close()
}

func TestParseAddrs(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "192.168.1.1:6379", []string{"192.168.1.1:6379"}},
		{"multiple", "192.168.1.1:6379,192.168.1.2:6379", []string{"192.168.1.1:6379", "192.168.1.2:6379"}},
		{"with spaces", "192.168.1.1:6379 , 192.168.1.2:6379", []string{"192.168.1.1:6379", "192.168.1.2:6379"}},
		{"trailing comma", "192.168.1.1:6379,", []string{"192.168.1.1:6379"}},
		{"leading comma", ",192.168.1.1:6379", []string{"192.168.1.1:6379"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseAddrs(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		cfg := buildTLSConfig(false)
		assert.Nil(t, cfg)
	})

	t.Run("enabled", func(t *testing.T) {
		cfg := buildTLSConfig(true)
		assert.NotNil(t, cfg)
	})
}
