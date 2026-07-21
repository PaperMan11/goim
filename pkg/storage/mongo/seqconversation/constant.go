package seqconversation

import "fmt"

const (
	KeySeqConv = "mongo:seqconv:%s"
)

func GetSeqConvKey(convID string) string {
	return fmt.Sprintf(KeySeqConv, convID)
}

const (
	seqDefaultExpireSeconds = 60 * 60
	seqNilExpireSeconds     = 30

	ttlJitterRatioPct = 10
)

var (
	sfKeyPrefixSeqConv = "uq:sc:"
)
