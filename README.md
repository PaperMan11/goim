# GoIM

基于 [go-zero](https://github.com/zeromicro/go-zero) 微服务框架构建的即时通讯（IM）系统，覆盖单聊、群聊、会话、好友关系、消息推送等完整链路，支持 WebSocket 长连接与离线推送。

## 整体架构

```
客户端 ──HTTP──▶ im-api ──gRPC──▶ RPC 服务层（auth/user/relation/group/conversation/msg）
                        │                    │
客户端 ──WebSocket──▶ im-msggateway ◀──gRPC──┘
                                     │
                              msg.rpc ──▶ Kafka(msg-transfer-topic)
                                              │
                                        im-msgtransfer ──▶ MongoDB（消息持久化）
                                              │
                                              ├──▶ Kafka(msg-push-topic) ──▶ im-push ──▶ 在线推送(msggateway)
                                              └──▶ Kafka(msg-offline-push-topic) ──▶ im-push ──▶ 离线推送
```

## 服务清单（11 个）

### 应用层

| 服务 | 入口 | 说明 |
|------|------|------|
| im-api | `./im-api/cmd` | HTTP API 网关，对外 REST 接口 |
| im-msggateway | `./im-msggateway/cmd` | WebSocket 长连接网关，消息收发与在线推送 |
| im-msgtransfer | `./im-msgtransfer/cmd` | Kafka 消费者，消息持久化 / 推送分发 |
| im-push | `./im-push/cmd` | Kafka 消费者，在线推送 / 离线推送 |
| im-cron | `./im-cron/cmd` | 定时任务（消息 / 会话留存清理） |

### RPC 服务层

| 服务 | 入口 | 说明 |
|------|------|------|
| auth.rpc | `./im-rpc/auth` | 认证与 Token 管理 |
| user.rpc | `./im-rpc/user` | 用户信息与管理员 |
| relation.rpc | `./im-rpc/relation` | 好友关系 |
| group.rpc | `./im-rpc/group` | 群组与群成员管理 |
| conversation.rpc | `./im-rpc/conversation` | 会话管理 |
| msg.rpc | `./im-rpc/msg` | 消息发送、撤回、已读等 |

## 技术栈

- **语言**：Go 1.25
- **框架**：go-zero（zrpc / rest）+ gin（API 层适配）
- **通信**：gRPC + Etcd 服务发现，protobuf 定义见 [pkg/protocol](pkg/protocol)
- **存储**：MongoDB（业务数据）、Redis（缓存 / Token / 在线状态）
- **消息队列**：Kafka（segmentio/kafka-go）
- **可观测**：Prometheus 指标（各服务独立端口）、结构化日志

## 环境依赖

| 中间件 | 默认地址 |
|--------|----------|
| Redis | 127.0.0.1:6379 |
| MongoDB | 127.0.0.1:27017 |
| Etcd | 127.0.0.1:2379 |
| Kafka | 127.0.0.1:9092 |

## 快速开始

```bash
# Windows 需先安装 make：scoop install make 或 choco install make

# 一键构建全部 11 个服务到 bin/
make build

# 按依赖顺序启动全部服务（RPC → msgtransfer → push → cron → api → msggateway）
make start

# 查看运行状态 / 停止
make status
make stop
```

启动后 pid 文件在 `run/`，stdout 日志在 `logs/`。单服务操作示例：

```bash
make start-api        # 单独启动 im-api
make stop-api         # 单独停止
make logs-msg         # 跟踪 msg.rpc 日志
```

完整 Makefile 目标见文件头注释或 `make help`。

## 端口规划

| 服务 | 端口 |
|------|------|
| im-api | HTTP **18880**，Prometheus 11010 |
| im-msggateway | WebSocket **50001**，gRPC 60001，Prometheus 11030 |
| RPC 服务 | gRPC 18010-18060（auth/conversation/group/msg/relation/user），Prometheus 28010-28060 |
| 后台任务 | Prometheus 11020-11050（cron/msgtransfer/push） |

完整端口分配与规划规则见 [docs/ports.md](docs/ports.md)。

## Kafka 消息流

| Topic | 生产者 | 消费者 |
|-------|--------|--------|
| msg-transfer-topic | msg.rpc | im-msgtransfer |
| msg-persist-topic | im-msgtransfer | im-msgtransfer |
| msg-push-topic | im-msgtransfer | im-push |
| msg-offline-push-topic | im-msgtransfer | im-push |

## 目录结构

```
goim/
├── im-api/            # HTTP API 网关
├── im-msggateway/     # WebSocket 长连接网关
├── im-msgtransfer/    # Kafka 消息转移（持久化/分发）
├── im-push/           # Kafka 在线/离线推送
├── im-cron/           # 定时任务
├── im-rpc/            # 6 个 RPC 服务（auth/conversation/group/msg/relation/user）
├── pkg/               # 公共库
│   ├── a2r/           #   gin → gRPC 通用适配（Call 泛型封装）
│   ├── apiresp/       #   统一响应体与错误码（errx，支持跨 gRPC 业务码透传）
│   ├── protocol/      #   protobuf 定义与生成代码
│   ├── queue/         #   队列抽象与 Kafka 实现
│   ├── storage/       #   Mongo 模型 + Redis 缓存层（Cache-Aside + SingleFlight）
│   ├── rpcclient/     #   RPC 客户端工厂（连接失败重试）
│   ├── rpcinterceptors/ # gRPC 客户端/服务端上下文拦截器
│   ├── webhooks/      #   回调管理
│   └── ...            #   randx/timex/lock/metrics/msgdispatcher 等
├── docs/              # 文档（端口总览、数据同步、版本控制、OpenAPI）
└── Makefile           # 构建/启动/运维脚本
```

## 开发

```bash
make test     # go test ./...
make vet      # go vet ./...
make fmt      # gofmt -s -w .
make proto    # 由 pkg/protocol 下的 proto 重新生成 gRPC 代码
              # （需要 protoc / protoc-gen-go / protoc-gen-go-grpc）
```

### 关键工程约定

- 服务启动时工作目录必须为服务目录（Makefile 已处理），配置文件位于各服务 `etc/` 下。
- RPC 客户端统一使用 `pkg/rpcclient.MustNewClient` 创建，避免依赖未就绪时启动失败。
- 错误统一使用 `pkg/apiresp/errx` 定义的业务错误码，`*ErrInfo` 实现 `GRPCStatus()` 可跨 gRPC 透传原始业务码与消息，前端不会收到 `rpc error: code = ...` 框架前缀。
- 缓存遵循 Cache-Aside + SingleFlight + TTL 抖动规范，详见 `pkg/storage` 下各模型实现。

## 相关文档

- [服务端口总览](docs/ports.md)
- [数据同步](docs/data-sync.md)
- [版本控制](docs/version-control.md)
- [OpenAPI 定义](docs/openapi.json)
- [Webhooks 说明](pkg/webhooks/README.md)
