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
)

var (
	sfKeyPrefixMsgSeq    = "uq:ms:"
	sfKeyPrefixMsgLatest = "uq:ml:"
	sfKeyPrefixMsgMax    = "uq:mx:"
	sfKeyPrefixMsgMin    = "uq:mn:"
)
