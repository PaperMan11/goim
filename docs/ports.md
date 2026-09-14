# GoIM 服务端口总览

> 自动汇总自各服务 `etc/*.yaml` 配置文件，便于本地开发与部署时快速查阅。

## 端口分配一览

### API / 网关层

| 服务 | 配置文件 | HTTP/API 端口 | WebSocket 端口 | Prometheus 端口 |
|------|----------|--------------|----------------|-----------------|
| im-api | `im-api/etc/api.yml` | **18080** | - | 11010 |
| im-msggateway | `im-msggateway/etc/msggateway.yml` | - (RPC: 8070) | **50001** | 11030 |

### RPC 服务层

| 服务 | Etcd Key | gRPC 端口 (ListenOn) | Prometheus 端口 |
|------|----------|---------------------|-----------------|
| auth.rpc | `auth.rpc` | **18010** | 28010 |
| conversation.rpc | `conversation.rpc` | **18020** | 28020 |
| group.rpc | `group.rpc` | **18030** | 28030 |
| msg.rpc | `msg.rpc` | **18040** | 28040 |
| relation.rpc | `relation.rpc` | **18050** | 28050 |
| user.rpc | `user.rpc` | **18060** | 28060 |

### 后台任务服务（无对外端口）

| 服务 | 配置文件 | Prometheus 端口 | 说明 |
|------|----------|-----------------|------|
| im-cron | `im-cron/etc/cron.yml` | 11020 | 定时任务（消息/会话留存清理） |
| im-msgtransfer | `im-msgtransfer/etc/msgtransfer.yml` | 11040 | Kafka 消费者（消息持久化/推送分发） |
| im-push | `im-push/etc/push.yml` | 11050 | Kafka 消费者（在线推送/离线推送） |

## 端口规划规则

| 端口段 | 用途 |
|--------|------|
| 11000-11099 | API/网关层 Prometheus |
| 18000-18099 | RPC 服务 gRPC 端口 |
| 28000-28099 | RPC 服务 Prometheus |
| 50001 | WebSocket 长连接端口 |
| 8070 | msggateway gRPC（ListenOn） |

## 中间件依赖

| 中间件 | 地址 |
|--------|------|
| Redis | 127.0.0.1:6379 |
| MongoDB | 127.0.0.1:27017 |
| Etcd | 127.0.0.1:2379 |
| Kafka | 127.0.0.1:9092 |

## Kafka Topic 一览

| Topic | 生产者 | 消费者 |
|-------|--------|--------|
| msg-transfer-topic | msg.rpc | im-msgtransfer |
| msg-persist-topic | im-msgtransfer | im-msgtransfer |
| msg-push-topic | im-msgtransfer | im-push |
| msg-offline-push-topic | im-msgtransfer | im-push |
