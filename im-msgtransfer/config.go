package immsgtransfer

import (
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	service.ServiceConf
	MsgTransferConsumer queuex.KafkaConfig
	MsgPersistentTopic  queuex.KafkaConfig
	MsgPushProducer     queuex.KafkaConfig
	Redis               redis.RedisConf
	Mongo               storagemongo.MongoConf

	MsgRpc rpcclient.RpcConf
}
