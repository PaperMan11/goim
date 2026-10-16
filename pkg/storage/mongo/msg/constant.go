package msg

import "fmt"

const (
	KeyMsgBySeq  = "mongo:msg:seq:%s:%d"
	KeyMsgLatest = "mongo:msg:latest:%s"
	KeyMsgMaxSeq = "mongo:msg:maxseq:%s"
	KeyMsgMinSeq = "mongo:msg:minseq:%s"
)

func GetMsgBySeqKey(convID string, seq int64) string {
	return fmt.Sprintf(KeyMsgBySeq, convID, seq)
}

func GetMsgLatestKey(convID string) string {
	return fmt.Sprintf(KeyMsgLatest, convID)
}

func GetMsgMaxSeqKey(convID string) string {
	return fmt.Sprintf(KeyMsgMaxSeq, convID)
}

func GetMsgMinSeqKey(convID string) string {
	return fmt.Sprintf(KeyMsgMinSeq, convID)
}

const (
	msgDefaultExpireSeconds = 10 * 60
	msgNilExpireSeconds     = 60

	ttlJitterRatioPct = 10

	// deleteDocChunkSize 按时间戳物理清理时单次 DeleteMany 的文档数上限，避免 $in 查询过大
	deleteDocChunkSize = 200
)

var (
	sfKeyPrefixMsgSeq    = "uq:ms:"
	sfKeyPrefixMsgLatest = "uq:ml:"
	sfKeyPrefixMsgMax    = "uq:mx:"
	sfKeyPrefixMsgMin    = "uq:mn:"
)
