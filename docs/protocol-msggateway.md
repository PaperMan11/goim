# 客户端与 MsgGateway 通信协议

本文档描述客户端（SDK）与 `im-msggateway` 服务之间的 WebSocket 长连接通信协议，包括连接建立、传输编码、消息封装、业务指令（reqIdentifier）、推送与踢下线机制、心跳保活和错误码。

协议定义代码位置：

- 指令枚举：[im-msggateway/internal/protocol.go](file:///c:/Users/Administrator/Desktop/goim/im-msggateway/internal/protocol.go)
- 请求/响应封装：[im-msggateway/internal/types.go](file:///c:/Users/Administrator/Desktop/goim/im-msggateway/internal/types.go)
- 握手参数：[im-msggateway/internal/conn_context.go](file:///c:/Users/Administrator/Desktop/goim/im-msggateway/internal/conn_context.go)
- 连接读写与心跳：[im-msggateway/internal/ws_connection.go](file:///c:/Users/Administrator/Desktop/goim/im-msggateway/internal/ws_connection.go)
- 业务分发：[im-msggateway/internal/message_handler.go](file:///c:/Users/Administrator/Desktop/goim/im-msggateway/internal/message_handler.go)
- 推送入口：[im-msggateway/internal/hub_server.go](file:///c:/Users/Administrator/Desktop/goim/im-msggateway/internal/hub_server.go)

---

## 1. 连接建立（握手）

### 1.1 接入地址

```
ws://<host>:<port>/ws
wss://<host>:<port>/ws   // 启用 TLS 时
```

### 1.2 握手参数（Query String）

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `userID` | string | 是 | 登录用户 ID，必须与 token 解析出的用户一致 |
| `token` | string | 是 | 登录令牌（由 auth 服务签发） |
| `platformID` | int | 是 | 平台 ID，取值见附录 A |
| `deviceID` | string | 否 | 设备唯一标识，用于在线状态管理 |
| `sendResponse` | bool | 否 | 握手/连接级错误是否通过连接回写 JSON 错误体 |
| `isBackground` | bool | 否 | 是否为后台连接（App 切后台时置 true） |
| `operationID` | string | 否 | 操作 ID，链路追踪 |
| `compression` | string | 否 | 压缩算法：`none`（默认）/ `gzip` / `zlib` |
| `sdkType` | string | 否 | SDK 类型，决定消息编解码器；未注册的类型回退到默认 JSON 编码 |
| `sdkVersion` | string | 否 | SDK 版本号（记录用） |

示例：

```
ws://127.0.0.1:10014/ws?userID=user_001&token=eyJhbGciOi...&platformID=5&sdkType=js&compression=gzip&operationID=op-123
```

### 1.3 认证流程

1. 服务端解析握手参数（`ParseHandshakeRequest`）。
2. 当 `EnableAuth=true` 时，调用 auth RPC `ParseToken` 校验 token：
   - token 解析失败 → 拒绝连接；
   - token 中的 `userID` 与握手参数 `userID` 不一致 → 拒绝连接；
   - token 中的 `platformID` 与握手参数 `platformID` 不一致 → 拒绝连接。
3. 校验通过后执行 HTTP → WebSocket Upgrade。
4. Upgrade 成功后连接加入连接管理器（ConnManager），按多端登录策略处理同用户/同平台旧连接（见 §6）。

握手失败时返回 HTTP 错误响应；若 `sendResponse=true` 且连接已建立，错误体会以 JSON 形式通过连接回写：

```json
{"errCode":10016,"errMsg":"handshake error: ..."}
```

### 1.4 连接数限制

当全局连接数达到 `MaxConns` 上限时，新握手请求被拒绝（`ConnOverMaxNumLimit`）。

---

## 2. 传输层

### 2.1 帧类型

| 方向 | 帧类型 | 说明 |
|---|---|---|
| 客户端 → 服务端 | Text / Binary | 业务消息（Request） |
| 服务端 → 客户端 | Binary | 业务消息（Response / Push） |
| 双向 | Ping / Pong | WebSocket 协议层心跳 |

### 2.2 消息处理流水线

接收方向（客户端 → 服务端）：

```
WS Frame → 解压（按 compression）→ 反序列化为 Request（按 sdkType 选择编码器）→ 业务分发（按 reqIdentifier）
```

发送方向（服务端 → 客户端）：

```
Response → 序列化（按 sdkType 选择编码器）→ 压缩（按 compression）→ Binary Frame
```

### 2.3 编码器（Encoder）

| sdkType | 编码 | 说明 |
|---|---|---|
| `normal`（默认） | JSON (`encoding/json`) | 默认编码器，未匹配的 sdkType 均回退到此 |
| 其他 | 可通过 `RegisterEncoder` 扩展（如 gob） | 当前内置 JSON 实现 |

> 注意：Request/Response 中 `data` 字段类型为 `[]byte`。在 JSON 编码下，`data` 以 **Base64 字符串** 形式出现在报文中，其原始字节为 **protobuf 二进制**（具体结构见 §4 各指令）。

### 2.4 压缩器（Compressor）

| compression | 算法 |
|---|---|
| 未指定 / `none` | 不压缩（默认） |
| `gzip` | gzip |
| `zlib` | zlib |

压缩作用于整条已编码的 Request/Response 报文。

### 2.5 大小与超时限制

| 项 | 值 | 说明 |
|---|---|---|
| 读大小限制 | `MaxMsgSize`（服务端配置） | 超过则连接被关闭 |
| 写超时 | 10s | 单次写帧超时 |
| 读超时（pongWait） | 60s | 60s 内未收到 Pong/消息则判定死连接 |
| 服务端 Ping 周期 | 54s（pongWait × 0.9） | 服务端主动 Ping，客户端必须回 Pong |
| 发送队列 | 256 条/连接 | Response 写入超时 1s 判失败 |

---

## 3. 消息封装

### 3.1 Request（客户端 → 服务端）

```json
{
  "reqIdentifier": 1003,
  "token": "eyJhbGciOi...",
  "senderID": "user_001",
  "operationID": "op-20261016-0001",
  "msgIncr": "42",
  "data": "<Base64(protobuf 二进制)>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `reqIdentifier` | int | 业务指令码，见 §4 |
| `token` | string | 用户令牌 |
| `senderID` | string | 发送者用户 ID；`EnableAuth=true` 时必须与握手 userID 一致，否则返回错误并断开 |
| `operationID` | string | 操作 ID，链路追踪，响应中原样回传 |
| `msgIncr` | string | 客户端请求自增标识，用于请求-响应配对，响应中原样回传 |
| `data` | []byte | 业务数据（protobuf 二进制；JSON 编码下为 Base64 字符串） |

### 3.2 Response（服务端 → 客户端）

```json
{
  "reqIdentifier": 1003,
  "msgIncr": "42",
  "operationID": "op-20261016-0001",
  "errCode": 0,
  "errMsg": "",
  "data": "<Base64(protobuf 二进制)>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `reqIdentifier` | int | 与请求一致；服务端主动推送时为推送类指令码（2001/2002 等） |
| `msgIncr` | string | 与请求一致（推送消息为空） |
| `operationID` | string | 与请求一致（推送消息为服务端生成） |
| `errCode` | int | 0 表示成功；非 0 见 §7 错误码 |
| `errMsg` | string | 错误描述（已剥离 gRPC 前缀的原始错误信息） |
| `data` | []byte | 业务数据（protobuf 二进制；JSON 编码下为 Base64 字符串） |

---

## 4. 业务指令（reqIdentifier）

### 4.1 指令总表

| 指令码 | 名称 | 方向 | 说明 |
|---|---|---|---|
| 1001 | WSGetNewestSeq | C→S | 获取各会话最新最大/最小 seq |
| 1002 | WSPullMsgBySeqList | C→S | 按 seq 范围拉取历史消息 |
| 1003 | WSSendMsg | C→S | 发送消息 |
| 1004 | WSSendSignalMsg | C→S | 发送信令消息（RTC 等） |
| 1005 | WSPullMsg | C→S | 按会话 seq 区间拉取消息 |
| 1006 | WSGetConvMaxReadSeq | C→S | 获取会话最大已读 seq |
| 1007 | WsPullConvLastMessage | C→S | 拉取会话最后一条消息 |
| 2001 | WSPushMsg | S→C | 服务端主动推送消息 |
| 2002 | WSKickOnlineMsg | S→C | 踢下线通知 |
| 2003 | WSLogoutMsg | C→S | 登出 |
| 2004 | WSSetBackgroundStatus | C→S | 设置前后台状态 |
| 2005 | WSSubUserOnlineStatus | C→S | 订阅用户在线状态 |
| 3001 | WSDataError | S→C | 数据错误通知 |

所有 `data` 均为 protobuf 二进制（`pkg/protocol/sdkws/sdkws.proto` 与 `pkg/protocol/msg/msg.proto` 中定义），下表给出各指令对应的 proto 结构。

### 4.2 请求类指令

#### 1001 WSGetNewestSeq — 获取最新 seq

- Request.data：`sdkws.GetMaxSeqReq`

```protobuf
message GetMaxSeqReq { string userID = 1; }
```

- Response.data：`sdkws.GetMaxSeqResp`

```protobuf
message GetMaxSeqResp {
  map<string, int64> maxSeqs = 1;  // 会话ID -> 最大seq
  map<string, int64> minSeqs = 2;  // 会话ID -> 最小seq
}
```

#### 1002 WSPullMsgBySeqList — 按 seq 范围拉取消息

- Request.data：`sdkws.PullMessageBySeqsReq`

```protobuf
message PullMessageBySeqsReq {
  string userID = 1;
  repeated SeqRange seqRanges = 2;
  PullOrder order = 3;              // 0-升序 1-降序
}
message SeqRange {
  string conversationID = 1;
  int64 begin = 2;
  int64 end = 3;
  int64 num = 4;                    // 数量限制
}
```

- Response.data：`sdkws.PullMessageBySeqsResp`

```protobuf
message PullMessageBySeqsResp {
  map<string, PullMsgs> msgs = 1;             // 普通消息：会话ID -> 消息集合
  map<string, PullMsgs> notificationMsgs = 2; // 通知消息：会话ID -> 消息集合
}
message PullMsgs {
  repeated MsgData Msgs = 1;
  bool isEnd = 2;                   // 是否拉完
  int64 endSeq = 3;
}
```

#### 1003 WSSendMsg — 发送消息

- Request.data：`sdkws.MsgData`（完整字段见《消息协议文档》）
- Response.data：`sdkws.UserSendMsgResp`

```protobuf
message UserSendMsgResp {
  string serverMsgID = 1;
  string clientMsgID = 2;
  int64 sendTime = 3;
}
```

#### 1004 WSSendSignalMsg — 发送信令消息

同 1003，用于 RTC 信令等场景，Request.data 为 `sdkws.MsgData`（`contentType=202 SignalMsg`），Response.data 为 `sdkws.UserSendMsgResp`。

#### 1005 WSPullMsg — 按会话区间拉取消息

- Request.data：`msg.GetSeqMessageReq`
- Response.data：`msg.GetSeqMessageResp`

#### 1006 WSGetConvMaxReadSeq — 获取会话已读/最大 seq

- Request.data：`msg.GetConversationsHasReadAndMaxSeqReq`

```protobuf
message GetConversationsHasReadAndMaxSeqReq {
  string userID = 1;
  repeated string conversationIDs = 2;
  bool returnPinned = 3;
}
```

- Response.data：`msg.GetConversationsHasReadAndMaxSeqResp`

#### 1007 WsPullConvLastMessage — 拉取会话最后一条消息

- Request.data：`msg.GetLastMessageReq`
- Response.data：`msg.GetLastMessageResp`

#### 2003 WSLogoutMsg — 登出

服务端回响应后立即关闭连接（连接以 LogoutError 关闭，不再触发踢下线广播与额外错误日志）。客户端以此指令实现主动登出语义。

#### 2004 WSSetBackgroundStatus — 设置前后台状态

- Request.data：`sdkws.SetAppBackgroundStatusReq`

```protobuf
message SetAppBackgroundStatusReq {
  string userID = 1;
  bool isBackground = 2;
}
```

- Response.data：`sdkws.SetAppBackgroundStatusResp`（空）

#### 2005 WSSubUserOnlineStatus — 订阅用户在线状态

- Request.data：`sdkws.SubUserOnlineStatus`

```protobuf
message SubUserOnlineStatus {
  repeated string subscribeUserID = 1;    // 订阅列表
  repeated string unsubscribeUserID = 2;  // 取消订阅列表
}
```

- 订阅结果通过 2001 推送下发（`SubUserOnlineStatusTips`，见《消息协议文档》）。

### 4.3 服务端推送类指令

#### 2001 WSPushMsg — 消息推送

服务端收到新消息/通知后，通过 msggateway RPC 向目标用户的所有在线连接推送：

```json
{
  "reqIdentifier": 2001,
  "msgIncr": "",
  "operationID": "msgtransfer:xxxx",
  "errCode": 0,
  "errMsg": "",
  "data": "<Base64(sdkws.PushMessages)>"
}
```

`PushMessages` 结构：

```protobuf
message PushMessages {
  map<string, PullMsgs> msgs = 1;             // 普通消息：会话ID -> 消息集合
  map<string, PullMsgs> notificationMsgs = 2; // 通知消息：会话ID -> 消息集合
}
```

- 普通消息放入 `msgs`，通知消息（会话 ID 前缀 `n_`）放入 `notificationMsgs`；
- 每条推送可携带多条 `MsgData`；
- 推送无需客户端应答，客户端按本地 seq 校验缺口后触发 1002 拉取补齐。

#### 2002 WSKickOnlineMsg — 踢下线

当同用户/同平台新连接登录触发互踢策略，或管理员调用 `KickUserOffline` 时，服务端向被踢连接发送：

```json
{"reqIdentifier":2002,"msgIncr":"","operationID":"","errCode":0,"errMsg":"","data":null}
```

发送后服务端立即移除该连接。客户端收到后应清理本地登录态并停止重连（或走重新登录流程）。

#### 3001 WSDataError — 数据错误

数据异常场景的通知指令（保留）。

---

## 5. 消息处理时序

### 5.1 发送消息（1003）

```
Client A                MsgGateway              Msg RPC              MsgTransfer/MQ
   │  Request(1003,MsgData) │                      │                      │
   │ ─────────────────────> │  proto.Unmarshal     │                      │
   │                        │ ─── SendMsg ───────> │ 分配seq/落库         │
   │                        │ <── SendMsgResp ──── │                      │
   │ Response(1003, UserSendMsgResp)               │                      │
   │ <───────────────────── │                      │ ── MQ ─────────────> │ 推送给接收方网关
```

### 5.2 接收推送（2001）

```
MsgGateway              Client B
   │ Response(2001, PushMessages{msgs:{convID:[MsgData...]}})
   │ ────────────────────> │ 校验 seq 连续性
   │                       │ 有缺口 → 发起 1002 WSPullMsgBySeqList 补齐
```

### 5.3 请求-响应配对

客户端为每个请求分配唯一 `msgIncr`，服务端响应原样带回；同一连接上允许并发多个在途请求，客户端按 `msgIncr` 匹配响应。

---

## 6. 多端登录策略

由服务端配置 `LoginStrategy` 决定，同用户多连接时的处理规则：

| 策略 | 值 | 行为 |
|---|---|---|
| `allow_multi` | 允许全端登录，同端互斥 | 不同平台共存；同平台超出数量限制时踢掉最旧连接 |
| `single` | 单端登录 | 新连接登录即踢掉该用户所有旧连接 |
| `replace` | 替换登录 | 新连接替换所有旧连接 |
| `replace_same_platform` | 同平台替换 | 仅踢掉相同 platformID 的旧连接 |

被踢连接收到 `2002 WSKickOnlineMsg` 后被移除。管理端也可通过 msggateway RPC `KickUserOffline`（可指定 platformID，0 表示全部平台）主动踢人。

平台互斥分组见附录 A（Mobile / PC / Pad 等终端类型）。

---

## 7. 错误码

响应 `errCode` 为 0 表示成功。常见错误码（完整定义见 [pkg/apiresp/errx/errx_code.go](file:///c:/Users/Administrator/Desktop/goim/pkg/apiresp/errx/errx_code.go)）：

| errCode | 常量 | 说明 |
|---|---|---|
| 1000 | ErrCodeUnknown | 未知错误 |
| 1001 | ErrCodeArgsError | 参数错误 |
| 1002 | ErrCodeNoPermissionError | 无权限 |
| 1004 | ErrCodeRecordNotFoundError | 记录不存在 |
| 1005 | ErrCodeUnknownMessageError | 未知 reqIdentifier |
| 1101 | ErrCodeUserIDNotFoundError | 用户不存在 |
| 1201 | ErrCodeGroupIDNotFoundError | 群不存在 |
| 1203 | ErrCodeNotInGroupYetError | 不在群中 |
| 1208 | ErrCodeGroupMutedError | 群已禁言 |
| 1209 | ErrCodeGroupMemberMutedError | 成员已被禁言 |
| 1302 | ErrCodeBlockedByPeer | 被对方拉黑 |
| 1303 | ErrCodeNotPeersFriend | 非对方好友 |
| 10005 | ErrCodeGetIMTokenError | 获取 IM token 失败 |
| 10015 | ErrCodeLogoutError | 已登出 |
| 10016 | ErrCodeHandshakeError | 握手错误 |
| 90001 | ErrCodeInternalError | 服务内部错误 |
| 90002 | ErrCodeDatabaseError | 存储层错误 |
| 90007 | ErrCodeDataError | 数据错误（如反序列化失败、senderID 不匹配） |

错误信息约定：`errMsg` 已剥离 `rpc error: code = ... desc =` 前缀，直接返回原始业务错误描述。

---

## 8. 连接关闭与在线状态

- 客户端主动断开、心跳超时、读写错误、登出（2003）均会导致连接移除；
- 连接建立时，msggateway 调用 user RPC `SetUserOnlineStatus` 上报 `(userID, platformID, connID, deviceID)` 上线，并触发 `user_online` webhook；
- 连接移除时上报下线（并调用 auth RPC `KickTokens` 注销本次 token），触发 `user_offline` webhook；
- 静默断开（Disconnect）不发送踢下线消息；主动踢人（Kick）先发送 2002 再断开。

---

## 附录 A. 平台 ID

| platformID | 名称 | 终端类型 |
|---|---|---|
| 1 | IOS | Mobile |
| 2 | Android | Mobile |
| 3 | Windows | PC |
| 4 | OSX | PC |
| 5 | Web | Web |
| 6 | MiniWeb | MiniWeb |
| 7 | Linux | PC |
| 8 | APad | Pad |
| 9 | IPad | Pad |
| 10 | Admin | Admin |
| 11 | HarmonyOS | HarmonyOS |
| 12 | Bot | Bot |

## 附录 B. 一次完整报文示例（JSON 编码 + 无压缩）

发送文本消息请求（客户端 → 服务端）：

```json
{
  "reqIdentifier": 1003,
  "token": "eyJhbGciOi...",
  "senderID": "user_001",
  "operationID": "op-0001",
  "msgIncr": "1",
  "data": "CgZ1c2VyXzAwMRIJdXNlcl8wMDIaBGhiZ2ki...=="
}
```

其中 `data` 解码后为 `sdkws.MsgData` 的 protobuf 二进制，其 JSON 等价为：

```json
{
  "sendID": "user_001",
  "recvID": "user_002",
  "clientMsgID": "client-uuid-0001",
  "senderPlatformID": 5,
  "sessionType": 1,
  "msgFrom": 100,
  "contentType": 101,
  "content": "{\"text\":\"Hello\",\"atUsers\":[]}",
  "options": {"history":true,"persistent":true,"offlinePush":true,"unreadCount":true,"conversationUpdate":true,"senderSync":true}
}
```

服务端响应：

```json
{
  "reqIdentifier": 1003,
  "msgIncr": "1",
  "operationID": "op-0001",
  "errCode": 0,
  "errMsg": "",
  "data": "CiBzZXJ2ZXItbXNnLWlkEg1jbGllbnQtdXVpZC0wMDAxG..."
}
```

`data` 解码后为 `sdkws.UserSendMsgResp{serverMsgID, clientMsgID, sendTime}`。
