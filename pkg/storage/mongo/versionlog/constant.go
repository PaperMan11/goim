package versionlog

import "fmt"

const (
	KeyVersionLog = "mongo:version_log:%s"
)

func GetVersionLogKey(did string) string {
	return fmt.Sprintf(KeyVersionLog, did)
}

const (
	defaultExpireSeconds = 10 * 60
	nilExpireSeconds     = 60

	ttlJitterRatioPct = 10

	// MaxLogEntries 单个 DID 文档最多保留的日志条数，超出后自动裁剪旧条目并推进 deleted 水位线
	MaxLogEntries = 1000
)

var (
	sfKeyPrefixVersion = "vl:v:"
)
