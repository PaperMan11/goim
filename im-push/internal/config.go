package internal

import (
	offlnepush "github.com/PaperMan11/goim/im-push/internal/offlnepush"
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	service.ServiceConf
	PushConsumer     queuex.KafkaConfig
	OfflinePushTopic queuex.KafkaConfig
	Redis            redis.RedisConf
	Mongo            storagemongo.MongoConf

	// OfflinePush 离线推送第三方配置
	OfflinePush offlnepush.Config

	UserRpc    rpcclient.RpcConf
	GatewayRpc rpcclient.RpcConf
	GroupRpc   rpcclient.RpcConf
}
