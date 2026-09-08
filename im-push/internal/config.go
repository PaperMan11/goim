package internal

import (
	offlnepush "github.com/PaperMan11/goim/im-push/internal/offlnepush"
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf
	PushConsumer     queuex.KafkaConfig
	OfflinePushTopic queuex.KafkaConfig
	Redis            redis.RedisConf
	Mongo            storagemongo.MongoConf

	// OfflinePush 离线推送第三方配置
	OfflinePush offlnepush.Config

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
