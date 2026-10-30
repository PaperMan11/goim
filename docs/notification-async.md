# 通知链路异步化改造记录

> 状态：**已实施**（2026-10-29）
> 涉及：`pkg/msgdispatcher`（通知统一出口）

## 1. 问题现象

集成测试中断线/首聊场景下，观察到两类通知发送失败（contentType=1300 `ConversationChangeNotification`，自发自收同步通知）：

### 阶段一：`user not found` 误杀（已修复）

```
error  check access failed, opUserID= targetUserID=it_xxx_a_1 err=rpc error: user not found
error  require self or admin failed, err: internal error, user not found, senderID: a_1, receiverID: a_1, groupID: , sessionType: 1
```

- `senderID == receiverID` 是通知类消息的标准模式（同步给用户自己的所有设备）
- 同一 trace 下 a_1、b_2 各一条、间隔 2ms —— 一次 RPC 的循环扇出
- 根因：通知触发路径的 ctx 无 opUserID（断线清理等 detached context / 入口未带操作人），`requireSelfOrAdmin` → `isIMAdmin("")` → user.rpc 报 user not found

### 阶段二：`context canceled` 拦腰砍断（本次修复）

```
error  fail - msg.rpc/SendMsg ... duration=989.0ms - rpc error: code = Canceled desc = context canceled
error  SendMsg failed ... caller=msgdispatcher/dispatcher.go
error  send conversation change notification failed, owner: b_2 ... caller=conversation_logic.go
```

- 首聊链路：`msg.rpc SendMsg(原始消息)` → `conversation.rpc CreateSingleChatConversations`（建双方会话文档 + 版本日志）→ **循环同步发 2 条通知 SendMsg 嵌套回 msg.rpc**
- 每次嵌套 SendMsg 内含：seq 分配（Redis Lua，首聊场景 `GetMaxSeqFn` 兜底大概率 miss 打 Mongo）+ Kafka 发布
- 延迟堆叠 ≈ 989ms，逼近 SDK 客户端 ~1s 超时预算 → 客户端取消 → 取消信号沿 `HTTP → msg.rpc → conversation.rpc → 嵌套 SendMsg` 级联，通知被拦腰砍断

## 2. 根因定性

**不是** "ctx 活得太久"，而是：**best-effort 语义的通知被同步串在主请求链路里**

1. 延迟堆叠：通知 RPC 的耗时全额计入用户请求延迟，首聊场景 cache miss 率最高、延迟最大
2. 取消级联：客户端超时/断连（恰恰是最需要通知的时刻）反而把通知杀掉
3. 结构问题：通知是 fire-and-forget，调用方根本不需要等它

## 3. 修复方案（已实施）

### 3.1 opUserID 注入（2026-10-29 已上线）

`dispatcher.go` `SendNotification` 出口统一注入 `ctx = mcontext.SetOpUserIDInContext(ctx, sendID)`——通知语义上就是"以 sendID 身份发出"，下游 `requireSelfOrAdmin` 判定 `opUserID == sendID` 通过。一处修复覆盖全部通知发送点。

### 3.2 有界异步派发（本次）

`dispatcher.go` `SendNotification` 尾部：

```go
taskCtx := context.WithoutCancel(ctx)
if !d.notifyPool.TrySubmit(func() {
    if err := d.sendMsg(taskCtx, msg); err != nil {
        logx.WithContext(taskCtx).Errorf("async send notification failed, ...")
    }
}) {
    logx.WithContext(ctx).Errorf("notification queue full, reject notification, ...")
    return errx.InternalError.Wrap("notification queue full")
}
return nil
```

设计决策：

| 决策点 | 选择 | 理由 |
|---|---|---|
| 执行载体 | `pkg/utils/workerpool`（32 worker / 4096 缓冲） | 项目现成组件（conn_manager 已用）；有界并发提供背压，防通知风暴打爆 goroutine |
| 提交方式 | `TrySubmit` 非阻塞 | 队列满明确报错（`notification queue full`），不做静默降级、不阻塞调用方 |
| ctx 处理 | `context.WithoutCancel` | 保留 context value（trace/metadata 连续性不断），仅切断取消/超时传播 |
| 同步段 | 消息构建（含 `fillSenderInfo`）保持同步 | 构建轻量；RPC 取用户名是 opt-in 行为，放在提交前 |
| 顺序反转风险 | 接受 | 客户端可能先收到消息推送、后收到 1300 通知；SDK 收到消息本身会本地建/更新会话，无害 |

**语义变化**：`SendNotification` 返回 nil 仅表示"已入队"，不代表发送成功；发送失败通过异步日志暴露。所有调用点原本就需容忍通知失败（多数是 log-and-continue），语义兼容。

**覆盖范围**：全部走 dispatcher 的通知——conversation（9 个调用点）、user、group、relation、msg 内部通知，一处修复全覆盖。

## 4. 验证方法

1. 首聊场景：新用户 A 给新用户 B 发首条消息，观察 msg.rpc SendMsg 整体耗时（应从 ~1s 降到 <100ms 量级）
2. 通知存活：发消息后立即断开客户端，检查 msg.rpc 侧 1300 通知仍正常入库/推送
3. 背压观测：日志中不应出现 `notification queue full`
4. 已知历史错误不再出现：`user not found`、`context canceled`（通知路径）

## 5. 后续优化

### 5.1 已实施：首聊建会话批量写版本日志（2026-10-30）

**方向 1（保持同步、消除延迟）落地**，对象是首聊同步链 `AddMsg → CreateSingleChat/GroupChatConversations`。

**实施前的诊断修正**：原推测"seq 兜底 Mongo RTT × 嵌套次数"不成立——`AddMsg` 在建会话**之前**先 `Allocate` seq（message_logic.go:1146），seq 冷启动兜底每会话终身仅一次，且嵌套通知的 seq 分配复用已预热的 Redis key。989ms 的真实大头是 `CreateXxxConversations` 内的 Mongo 写入轮次，其中群聊为**逐成员循环 IncrVersionLog**（N 成员 = N 次 Mongo 往返）。因此 seq 预热无收益，改为**批量化版本日志**。

**改动**：

| 文件 | 内容 |
|---|---|
| `pkg/storage/mongo/versionlog/version_log.go` | 新增 `IncrVersionLogBatchDIDs(ctx, dids, eid, state)`：跨主体批量（N 个 DID × 同一 EID），每 DID 一条聚合管道（版本自增 + 去重 + 追加），合并为单次 unordered `BulkWrite`，N 次往返 → 1 次；dids 内部去重（自聊 sendID==recvID）；要求 MongoDB ≥ 4.2 |
| `pkg/storage/mongo/versionlog/cached_version_log.go` | 包装：写后逐 DID 精确失效 `CacheDelDouble` |
| `im-rpc/conversation/internal/logic/conversation_logic.go` | `CreateSingleChatConversations`：2 次 IncrVersionLog → 1 次 BatchDIDs；`CreateGroupChatConversations`：N 次循环 → 1 次 BatchDIDs |

**效果**（首聊同步链 Mongo 往返）：

| 场景 | 改造前 | 改造后 |
|---|---|---|
| 单聊首聊 | 1(批量insert) + 2(版本日志) = 3 次 | 2 次 |
| N 人群首聊 | 1 + N = N+1 次 | 2 次（与 N 无关） |

**语义注意**：unordered BulkWrite 下部分 DID 失败不自动重试，整批 err 上报（现有调用方版本日志失败均为 log-and-continue，兼容）；各 DID 的 version 独立自增，与逐个调用语义一致。

**AddMsgs / AddMsg 自动受益**：二者调用同一 Create 函数（message_logic.go:1156/1175/1229/1248）。

### 5.2 未实施

- **CreateXxxConversations 异步化评估**（方向 2）：建会话从 AddMsg 主链路剥离需重新设计"消息先于会话到达"的消费端兜底，暂缓——批量优化后同步链已与 N 无关，群聊收益递减
- **死代码清理**：`user/notification.go` `UserStatusChangedNotification` 无任何调用点——若产品预期"上下线通知订阅者"，则缺的是调用方而非删除；需产品确认
- **审计结论（2026-10-29）**：SendMsg 校验段（IsIMAdmin/IsBlack/IsFriend/群缓存）为必要同步（异步化会引入 TOCTOU 窗口）；群退群/踢人的 `SetUserConversationMaxSeq` 为会话边界一致性语义，必须保持同步；其余嵌套均为缓存读，无级联风险

## 6. 关键文件

| 文件 | 说明 |
|---|---|
| `pkg/msgdispatcher/dispatcher.go` | 通知统一出口：opUserID 注入 + 异步派发 |
| `pkg/utils/workerpool/pool.go` | 有界 workerpool（泛型） |
| `im-rpc/conversation/internal/logic/conversation_logic.go` | `CreateSingleChatConversations` 等通知触发点 |
| `im-rpc/msg/internal/logic/message_logic.go` | `SendMsg`/`requireSelfOrAdmin`、首聊建会话调用点 |
| `im-rpc/msg/internal/svc/service_context.go` | seq 分配器（`GetMaxSeqFn` 兜底） |
