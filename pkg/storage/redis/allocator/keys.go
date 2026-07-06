package allocator

const (
	MessageSeqPrefix = "goim:msg:seq:"
)

func GetMessageSeqKey(conversationID string) string {
	return MessageSeqPrefix + conversationID
}