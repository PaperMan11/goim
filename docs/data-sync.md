# GoIM 客户端与服务端数据同步机制

---

## 一、同步体系总览

GoIM 的数据同步分为两大维度，各自使用不同的同步策略：

| 维度 | 同步策略 | 版本载体 | 核心接口 |
|------|---------|---------|---------|
| **关系数据同步**（群成员、好友、黑名单、会话） | VersionLog 增量同步 + 全量回退 | `version` + `versionID` | `FindChangeLog` / `IncrVersionLogBatch` |
| **消息数据同步** | Seq 序列号拉取 | `seq` + `maxSeq` + `hasReadSeq` | `GetMaxSeq` / `PullMessageBySeqs` |

关系数据同步覆盖 5 类实体，各自独立的 DID 命名空间：

| 实体 | DID | RPC 方法 | 使用场景 |
|------|-----|---------|---------|
| 群成员 | `groupID`（无前缀） | `GetIncrementalGroupMember` | 群成员变更同步 |
| 加入群 | `joingroup: + userID` | `GetIncrementalJoinGroup` | 用户加入的群列表同步 |
| 好友 | `friend: + userID` | `GetIncrementalFriends` | 好友列表同步 |
| 黑名单 | `black: + userID` | `GetIncrementalBlacks` | 黑名单同步 |
| 会话 | `conv: + userID` | `GetIncrementalConversation` | 会话列表同步 |

---

## 二、同步流程图

### 2.1 关系数据同步（VersionLog 机制）

```
┌─────────────────────────────────────────────────────────────────────┐
│                           客户端                                     │
│  本地状态: version=42, versionID="65a1b2c3...", sortVersion=40      │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               │ ① 收到 WebSocket 通知
                               │   (携带 friendVersion=43, versionID="65a1b2c4...")
                               ▼
              ┌────────────────────────────────┐
              │  ② 判断是否需要同步             │
              │  通知 version(43) > 本地(42)?   │
              │  → 是: 发起增量同步              │
              │  → 否: 跳过                     │
              └───────────────┬────────────────┘
                              │
                              │ ③ GetIncrementalGroupMember
                              │   (groupID, version=42, versionID="65a1b2c3...")
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        服务端 (group.rpc)                            │
│                                                                     │
│  ④ 权限校验: 群存在 + 未解散 + 调用者是群成员                          │
│                                                                     │
│  ⑤ FindChangeLog(groupID, clientVersion=42, limit=200)             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ MongoDB 聚合 Pipeline:                                      │   │
│  │                                                             │   │
│  │ Stage 1: $match 定位 DID 文档                               │   │
│  │ Stage 2: 兼容性校验                                           │   │
│  │   clientVersion(42) >= deleted(0)? 否                        │   │
│  │   clientVersion(42) >  version(43)? 否                      │   │
│  │   → 校验通过, 保留 logs                                      │   │
│  │ Stage 3: $filter → version > 42 的条目                       │   │
│  │ Stage 4: $size → 变更数                                      │   │
│  │ Stage 5: (limit=200) 变更数 ≤ 200? 是 → 保留 logs            │   │
│  │ Stage 6: $project 最终投影                                   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ⑥ 判断分支                                                        │
│  ┌──────────────────────┐    ┌──────────────────────────────────┐  │
│  │ Logs 非空            │    │ Logs 为空                        │  │
│  │ 且 versionID 匹配    │    │ 或 versionID 不匹配              │  │
│  │ → 增量同步           │    │ → 全量同步                        │  │
│  └──────────┬───────────┘    └──────────────┬───────────────────┘  │
│             │                               │                      │
│  ⑦ 增量同步  │                               │  ⑧ 全量同步          │
│  ClassifyIncrementalLogs                   │  FindMembersByGroup   │
│  ├ InsertIDs → 拉取成员详情                │  返回全部成员 + Full=true│
│  ├ UpdateIDs → 拉取成员详情                │  + 当前 version/ID     │
│  ├ DeleteIDs → 直接返回                    │                       │
│  ├ SortChanged → SortVersion              │                       │
│  └ GroupChanged → 附带群信息              │                       │
│             │                               │                      │
│  返回: Full=false                        返回: Full=true           │
│       Insert/Update/Delete 成员                  Insert=全部成员    │
│       Version=43, SortVersion=43               Version=43          │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               │ ⑨ 客户端应用变更
                               ▼
              ┌────────────────────────────────┐
              │  增量同步 (Full=false):         │
              │  本地新增 Insert 成员           │
              │  本地更新 Update 成员           │
              │  本地删除 Delete 成员           │
              │  本地 version = 43             │
              │  本地 versionID = 新ID         │
              │                                │
              │  全量同步 (Full=true):          │
              │  本地列表 = 服务端全量列表      │
              │  本地 version = 43             │
              └────────────────────────────────┘
```

### 2.2 消息数据同步（Seq 机制）

```
┌─────────────────────────────────────────────────────────────────────┐
│                           客户端                                     │
│  本地状态: maxSeq=100, hasReadSeq=95                                │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               │ ① GetMaxSeq(userID)
                               │   获取所有会话的最大 seq
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        服务端 (msg.rpc)                             │
│                                                                     │
│  ② SeqUserModel.FindAllUserSeqs(userID)                            │
│     → 返回 map[conversationID] → {MaxSeq, MinSeq}                  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               │ ③ 客户端比对
                               │   服务端 maxSeq=105 > 本地 100?
                               │   → 是: 需要拉取
                               ▼
              ┌────────────────────────────────┐
              │  ④ PullMessageBySeqs           │
              │  SeqRanges: [{convID,          │
              │    Begin=101, End=105, Num=5}]  │
              └───────────────┬────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        服务端 (msg.rpc)                             │
│                                                                     │
│  ⑤ 遍历 SeqRanges, 逐会话调用 pullMessage()                        │
│     → 从 MongoDB 按 seq 范围查询消息文档                            │
│     → 分离普通消息 vs 通知消息 (isNotification)                      │
│                                                                     │
│  返回: map[convID] → {MsgList, NotificationMsgList}                │
└──────────────────────────────┬──────────────────────────────────────┘
                               │
                               │ ⑥ 客户端追加消息到本地
                               ▼
              ┌────────────────────────────────┐
              │  本地 maxSeq = 105             │
              │  新消息已写入本地数据库         │
              └────────────────────────────────┘
```

### 2.3 通知触发同步

```
┌──────────────┐     WebSocket      ┌──────────────┐
│  服务端 RPC   │ ──────────────────▶ │   客户端      │
│              │  通知携带版本号      │              │
│  IncrVersion │  friendVersion=43   │  收到通知     │
│  LogBatch()  │  versionID=xxx     │  比对版本     │
│  推进版本    │  groupMemberVer=43 │  发起同步     │
│              │ ──────────────────▶ │              │
└──────────────┘                     └──────────────┘
```

---

## 三、大群同步痛点与解决方案

### 3.1 痛点

| 痛点 | 说明 |
|------|------|
| **全量同步代价大** | 万人群全量同步需返回 1 万条成员详情（nickname/avatar/role 等），网络/内存/序列化开销巨大 |
| **Logs 数组膨胀** | 高频变更群（频繁加退成员）的 logs 数组会无限增长，单文档体积膨胀 |
| **DB 查询压力** | 每次同步都需查询 MongoDB，大群场景下 FindMembersByGroup 全量扫描代价高 |
| **重复变更冗余** | 同一用户先加群又退群，logs 中会产生两条记录，客户端需合并处理 |
| **落后太多导致雪崩** | 客户端长时间离线后 version 落后很多，增量同步反而比全量更重 |

### 3.2 本项目的解决方案

#### 方案 1：全有或全无策略（SyncLimit=200）

[group_logic.go:1633](file:///c:/Users/Administrator/Desktop/goim/im-rpc/group/internal/logic/group_logic.go#L1633) — [version_log.go:237-314](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/mongo/versionlog/version_log.go#L237-314)

```
变更数 ≤ 200  → 返回全部增量变更（只传差异，轻量）
变更数 > 200  → 返回空数组，客户端回退全量同步
```

此策略避免了"增量比全量更重"的倒挂问题：如果变更数过大，增量传输的数据量可能超过全量，不如直接全量。

#### 方案 2：Logs 数组去重 — 每实体只保留最新变更

[version_log.go:98-108](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/mongo/versionlog/version_log.go#L98-108)

`IncrVersionLogBatch` 在 Pipeline Stage 4 使用 `$filter` 在写入前剔除同 EID 旧条目：

```
写入前 logs = [user_A:insert@v10, user_B:delete@v15, user_A:delete@v30]
本次写入   = IncrVersionLogBatch(group_1, ["user_A","user_C"], Insert)

$filter 后:  [user_B:delete@v15]              ← user_A 的两条旧记录被剔除
追加后:      [user_B:delete@v15, user_A:insert@v43, user_C:insert@v43]
```

同一实体在 logs 中永远只保留最新一条状态，避免数组无限膨胀。

#### 方案 3：ClassifyIncrementalLogs — 多次操作合并为最终态

[version.go:131-159](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/model/version.go#L131-159)

即使 logs 中因并发等原因出现同一 EID 的多条记录，`ClassifyIncrementalLogs` 在分类时使用 `seen` map 做覆盖：

```go
seen := make(map[string]int32) // EID → 最后一次 State（后者覆盖前者）
for _, log := range logs {
    seen[log.EID] = log.State
}
```

同一用户先 insert@v10 再 delete@v30 → 最终归入 DeleteIDs，客户端只需执行一次删除操作。

#### 方案 4：Redis + SingleFlight 缓存层

[service_context.go:58](file:///c:/Users/Administrator/Desktop/goim/im-rpc/group/internal/svc/service_context.go#L58)

```go
versionLogModel := versionLogModel.NewCachedVersionLogModelFromMongo(versionMongo, redisCli, singleFlight)
```

VersionLog 查询走 `MongoDB + Redis 缓存 + SingleFlight` 三级链路：
- **Redis 缓存**：`GetVersionLog` / `FindChangeLog` 命中缓存时直接返回，不查 MongoDB
- **SingleFlight**：缓存 miss 时，同一 DID 的并发请求合并为一次 MongoDB 查询，避免缓存击穿
- **CAS 写入**：缓存更新使用 `CacheSetCAS`（版本号乐观锁），防止旧版本缓存覆盖新版本

#### 方案 5：哈希校验 — 轻量判断是否需要全量

[group_logic.go:1833-1858](file:///c:/Users/Administrator/Desktop/goim/im-rpc/group/internal/logic/group_logic.go#L1833-1858)

`GetFullGroupMemberUserIDs` 返回成员 ID 列表的哈希值，客户端比对本地哈希：

```go
curHash := hash.HashStringSet(userIDs)
resp.Equal = req.IdHash != 0 && req.IdHash == curHash
```

- `Equal=true` → 客户端本地 ID 列表与服务端一致，无需同步
- `Equal=false` → 需要进一步走增量/全量同步

此方案避免了每次都拉取全量成员详情，只传 ID 列表 + 哈希即可判断是否一致。

#### 方案 6：批量增量同步 — 减少网络往返

[group_logic.go:1720-1737](file:///c:/Users/Administrator/Desktop/goim/im-rpc/group/internal/logic/group_logic.go#L1720-1737)

`BatchGetIncrementalGroupMember` 支持一次请求同步多个群的成员变更，客户端登录后可批量拉取所有加入群的增量，减少 RPC 往返次数。

---

## 四、增量同步与全量同步的判定逻辑

### 4.1 判定流程

以 [GetIncrementalGroupMember](file:///c:/Users/Administrator/Desktop/goim/im-rpc/group/internal/logic/group_logic.go#L1612) 为例：

```
FindChangeLog(did, clientVersion, limit=200)
        │
        ├─ 文档不存在 → 自动初始化空文档 → Logs=[] → 全量同步
        │
        ├─ clientVersion >= deleted → 旧日志已清理 → Logs=[] → 全量同步
        │
        ├─ clientVersion > version → 客户端版本超前(异常) → Logs=[] → 全量同步
        │
        ├─ 变更数 > 200 → 落后太多 → Logs=[] → 全量同步
        │
        └─ Logs 非空 + versionID 匹配 → 增量同步
             │
             └─ Logs 非空但 versionID 不匹配 → 全量同步
```

### 4.2 versionID 的作用

`versionID` 是 MongoDB 文档的 ObjectID hex 字符串，在 `IncrVersionLogBatch` 写入后返回。客户端需要同时保存 `version`（数字版本号）和 `versionID`（文档 ID）：

- `version` 用于 `FindChangeLog` 过滤旧变更
- `versionID` 用于校验文档一致性 — 如果服务端 version log 文档被重建（如群解散后重新创建），versionID 会变化，即使 version 数字相同也需要全量同步

### 4.3 全量同步响应

```go
// fullGroupMemberResp
return &pbgroup.GetIncrementalGroupMemberResp{
    Version:     curVersion,  // 当前最新版本号
    VersionID:   groupID,     // DID
    Full:        true,         // 标记全量
    Insert:      inserts,      // 全量成员列表
    SortVersion: curVersion,   // 排序版本 = 当前版本
}
```

### 4.4 增量同步响应

```go
// GetIncrementalGroupMember 增量分支
return &pbgroup.GetIncrementalGroupMemberResp{
    Version:     uint64(verLog.Version),  // 最新版本号
    VersionID:   groupID,
    Full:        false,                    // 标记增量
    Insert:      insertMembers,            // 新增成员详情
    Update:      updateMembers,            // 更新成员详情
    Delete:      c.DeleteIDs,              // 删除成员 ID 列表
    SortVersion: c.SortVersion,            // 排序版本(仅变更时)
    Group:       groupInfo,               // 群信息(仅变更时)
}
```

---

## 五、各实体同步接口对照

### 5.1 关系数据同步

| 实体 | DID | 增量接口 | 全量接口 | 哈希校验接口 |
|------|-----|---------|---------|-------------|
| 群成员 | `groupID` | `GetIncrementalGroupMember` | `fullGroupMemberResp`（内部） | `GetFullGroupMemberUserIDs` |
| 加入群 | `joingroup:userID` | `GetIncrementalJoinGroup` | `fullJoinGroupResp`（内部） | `GetFullJoinGroupIDs` |
| 好友 | `friend:userID` | `GetIncrementalFriends` | 全量分支（内部） | `GetFullFriendUserIDs` |
| 黑名单 | `black:userID` | `GetIncrementalBlacks` | 全量分支（内部） | — |
| 会话 | `conv:userID` | `GetIncrementalConversation` | 全量分支（内部） | — |

### 5.2 消息数据同步

| 接口 | 功能 | 输入 | 输出 |
|------|------|------|------|
| `GetMaxSeq` | 获取所有会话最大 seq | `userID` | `map[convID]→{MaxSeq, MinSeq}` |
| `PullMessageBySeqs` | 按 seq 范围拉取消息 | `userID` + `SeqRanges[{convID, Begin, End, Num}]` | `map[convID]→{MsgList, NotificationMsgList}` |
| `GetConversationsHasReadAndMaxSeq` | 获取已读 seq + 最大 seq | `userID` | `map[convID]→{MaxSeq, HasReadSeq}` |
| `SetConversationHasReadSeq` | 设置已读 seq | `userID` + `convID` + `hasReadSeq` | — |

### 5.3 API 路由对照

| 路由 | RPC 方法 | 同步类型 |
|------|---------|---------|
| `GET /newest_seq` | `GetMaxSeq` | 消息同步 |
| `POST /pull_msg_by_seq` | `PullMessageBySeqs` | 消息同步 |
| `GET /get_conversations_has_read_and_max_seq` | `GetConversationsHasReadAndMaxSeq` | 消息同步 |
| `POST /set_conversation_has_read_seq` | `SetConversationHasReadSeq` | 消息同步 |
| `POST /get_incremental_join_groups` | `GetIncrementalJoinGroup` | 关系同步 |
| `POST /get_incremental_group_members` | `GetIncrementalGroupMember` | 关系同步 |
| `POST /get_incremental_group_members_batch` | `BatchGetIncrementalGroupMember` | 关系同步 |
| `POST /get_full_group_member_user_ids` | `GetFullGroupMemberUserIDs` | 哈希校验 |
| `POST /get_full_join_group_ids` | `GetFullJoinGroupIDs` | 哈希校验 |
| `POST /get_incremental_friends` | `GetIncrementalFriends` | 关系同步 |
| `POST /get_incremental_blacks` | `GetIncrementalBlacks` | 关系同步 |
| `POST /get_full_friend_user_ids` | `GetFullFriendUserIDs` | 哈希校验 |
