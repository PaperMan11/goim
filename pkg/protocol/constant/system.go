// Package constant 定义了 OpenIM 系统中使用的各种常量
package constant

// Token 状态
const (
	NormalToken  = 0 // 正常 Token
	InValidToken = 1 // 无效 Token
	KickedToken  = 2 // 被踢 Token
	ExpiredToken = 3 // 过期 Token
)

// 多端登录策略
const (
	LoginStrategyAllowMulti          = "allow_multi"           // 允许全端登录，但同端互斥
	LoginStrategySingle              = "single"                // 允许单端登录
	LoginStrategyReplace             = "replace"               // 替换登录
	LoginStrategyReplaceSamePlatform = "replace_same_platform" // 替换相同平台登录
)

// RPC 和 HTTP 请求头键名常量
const (
	OperationID     = "operationID"  // 操作 ID，用于请求追踪
	OpUserID        = "opUserID"     // 操作用户 ID
	ConnID          = "connID"       // 连接 ID
	OpUserPlatform  = "platform"     // 用户平台
	Token           = "token"        // 认证令牌
	RpcCustomHeader = "customHeader" // 自定义 RPC 头 (用于 RPC 中间件)
	CheckKey        = "checkKey"     // 校验键
	TriggerID       = "triggerID"    // 触发器 ID
	ClientIP        = "clientIP"     // 客户端 IP
)

// 对象存储超时时间 (秒)
const (
	MinioDurationTimes = 3600 // Minio 预签名 URL 有效期
	AwsDurationTimes   = 3600 // AWS 预签名 URL 有效期
)

// 命令行参数标志
// 用于解析启动时的命令行参数
const (
	FlagPort                  = "port"                  // 服务端口
	FlagWsPort                = "ws_port"               // WebSocket 端口
	FlagTransferProgressIndex = "transferProgressIndex" // 传输进度索引
	FlagPrometheusPort        = "prometheus_port"       // Prometheus 监控端口
	FlagConf                  = "config_folder_path"    // 配置文件目录路径
)

// 日志文件名
const LogFileName = "OpenIM.log"

// 本地监听地址
const LocalHost = "0.0.0.0"

// OpenIM 通用配置键
const OpenIMCommonConfigKey = "OpenIMServerConfig"

// 批量操作数量
const BatchNum = 100

// 分页相关常量
const (
	FirstPageNumber   = 1   // 首页页码
	MaxSyncPullNumber = 500 // 最大同步拉取数量
)
