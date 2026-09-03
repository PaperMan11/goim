package immsgtransfer

import (
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf
	MsgTransferConsumer   queuex.KafkaConfig
	MsgPersistentConsumer queuex.KafkaConfig
	MsgPersistentProducer queuex.KafkaConfig
	MsgPushProducer       queuex.KafkaConfig
	Redis                 redis.RedisConf
	Mongo                 storagemongo.MongoConf

	AuthRpc         RpcConf
	UserRpc         RpcConf
	MsgRpc          RpcConf
	PushRpc         RpcConf
	GatewayRpc      RpcConf
	GroupRpc        RpcConf
	ConversationRpc RpcConf
}

type RpcConf struct {
	zrpc.RpcClientConf
	Stub bool `json:",default=false"`
}
