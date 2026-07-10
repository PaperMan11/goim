package rpccache

import "github.com/PaperMan11/goim/pkg/rpcclient"

type RpcWithCacheConf struct {
	rpcclient.RpcConf
	Topic string `json:",default=rpccache"` // 订阅主题，触发删除缓存的key
}
