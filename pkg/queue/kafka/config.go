package kafka

type Offset string

const (
	OffsetFirst Offset = "first"
	OffsetLast  Offset = "last"
)

type KafkaConfig struct {
	Brokers     []string
	Topic       string
	GroupID     string
	Offset      Offset `json:",options=first|last,default=last"`
	MinBytes    int    `json:",default=10240"`
	MaxBytes    int    `json:",default=10485760"`
	Username    string `json:",optional"`
	Password    string `json:",optional"`
	CaFile      string `json:",optional"`
	InsecureTLS bool   `json:",default=false"`
	Consumers   int    `json:",default=1"`
	Processors  int    `json:",default=8"`
	ForceCommit bool   `json:",default=true"`
	OrderCommit bool   `json:",default=false"` // 是否按顺序提交)
	// KeySharding 按消息 key（如 conversationID）hash 分发到固定 consumeLoop，
	// 使同 key 消息严格串行处理。与 OrderCommit 互斥，OrderCommit 优先。
	KeySharding bool `json:",default=false"`
}
