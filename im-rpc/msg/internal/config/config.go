package config

import (
	"github.com/PaperMan11/goim/pkg/localcache"
	queuex "github.com/PaperMan11/goim/pkg/queue/kafka"
	"github.com/PaperMan11/goim/pkg/rpcclient"
	storagemongo "github.com/PaperMan11/goim/pkg/storage/mongo"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	// 发送消息是否需要关系验证
	SendMsgNeedRelationVerify bool `json:",default=false"`

	Mongo          storagemongo.MongoConf
	LocalCacheConf localcache.CacheConfig

	MsgTransferProducer queuex.KafkaConfig

	UserRpc     rpcclient.RpcConf
	ConvRpc     rpcclient.RpcConf
	RelationRpc rpcclient.RpcConf
	GroupRpc    rpcclient.RpcConf
	MsgRpc      rpcclient.RpcConf
}
