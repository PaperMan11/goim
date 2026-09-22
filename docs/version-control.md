# GoIM 版本控制机制

> 本文档详细描述 VersionLog 数据增量同步版本控制机制及其完整流程。

---

## 一、数据结构

**核心文件：**
- [version.go](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/model/version.go) — 数据模型
- [version_log.go](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/mongo/versionlog/version_log.go) — MongoDB 存储实现

### VersionLogTable

每条文档对应一个 DID（日志主体），记录该主体从首次创建以来的全部变更链。

```go
type VersionLogTable struct {
    DID        string            // 日志主体ID
    Logs       []VersionLogElem  // 变更日志链
    Version    uint              // 当前最新版本号（单调递增）
    Deleted    uint              // 已清理水位线（低于此值的旧日志已被删除）
    LastUpdate time.Time         // 最后更新时间
}

type VersionLogElem struct {
    EID        string    // 实体ID（成员变更=用户ID，群变更=__GROUP_CHANGE__）
    State      int32     // 1=新增, 2=删除, 3=更新
    Version    uint      // 该条变更对应的版本号
    LastUpdate time.Time // 该条变更的时间
}
```

### 版本状态常量

```go
const (
    VersionStateInsert = iota + 1 // 1 新增
    VersionStateDelete            // 2 删除
    VersionStateUpdate            // 3 更新
)

const FirstVersion         = 1  // 首版本号
const DefaultDeleteVersion = 0  // 初始水位线
```

### DID 命名空间

不同类型的数据使用前缀区分，避免 logs 数组混杂不同 EID：

| 前缀 | 用途 | DID 构造 |
|------|------|---------|
| `friend:` | 好友版本日志 | `friend: + userID` |
| `black:` | 黑名单版本日志 | `black: + userID` |
| `group_member:` | 群成员版本日志 | `group_member: + userID` |
| `join_group:` | 加入群版本日志 | `join_group: + userID` |
| `conversation:` | 会话版本日志 | `conversation: + userID` |
| (无前缀) | 群成员同步 | `groupID` |

### Proto 字段

relation / group / conversation 的 proto 中使用 `versionID`（string）+ `sortVersion`（uint64）双版本字段：

```protobuf
// relation.proto / group.proto / conversation.proto
string versionID = 2;      // 版本ID（MongoDB ObjectID hex）
uint64 sortVersion = 7;    // 排序版本号

// sdkws.proto
uint64 groupMemberVersion = 6;  // 群成员版本号
uint64 groupSortVersion = 7;    // 群排序版本号
uint64 friendVersion = 3;      // 好友版本号
uint64 friendSortVersion = 4;  // 好友排序版本号
```

---

## 二、版本推进机制 — IncrVersionLogBatch

[version_log.go:50-168](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/mongo/versionlog/version_log.go#L50-168)

使用单条 `FindOneAndUpdate` 聚合 Pipeline 原子完成 **版本自增 + 旧条目去重 + 新条目追加**，避免两步操作（先自增再 push）的一致性问题和批量场景下产生 N 个不同 version。

### Pipeline 各 Stage

```
Stage 1: $addFields — 把 eids 暂存为临时字段 delete_e_ids
Stage 2: $set — 对 upsert 文档补零值（version=1, deleted=0, logs=[]）
Stage 3: $set — version += 1，last_update = now
Stage 4: $set — $filter 剔除 logs 中与本次 eids 相同的旧条目
           （每实体只保留最新一次变更，避免历史膨胀）
Stage 5: $set — $concatArrays 追加新条目（version 取 Stage 3 计算后的 $version）
Stage 6: $unset — 移除临时字段 delete_e_ids
```

### 去重示例

```
DB 当前 logs = [user_A:insert@v10, user_B:delete@v15, user_A:delete@v30]
本次调用    = IncrVersionLogBatch(ctx, "group_1", ["user_A","user_C"], Insert)

Stage 4 $filter 后:
  logs = [user_B:delete@v15]              ← user_A 的两条旧记录被剔除

Stage 5 $concatArrays 后:
  logs = [user_B:delete@v15,
          user_A:insert@v43,              ← 新追加，version=43
          user_C:insert@v43]              ← 新追加，version=43
```

### 投影优化

`FindOneAndUpdate` 设置 `SetProjection({logs: 0})`，只返回 version/deleted 等元字段，不把整个 logs 数组拉回客户端。调用方在内存中拼装本次新增的 N 条 elem。

---

## 三、增量同步查询 — FindChangeLog

[version_log.go:237-314](file:///c:/Users/Administrator/Desktop/goim/pkg/storage/mongo/versionlog/version_log.go#L237-314)

客户端携带本地 `clientVersion` 发起增量同步请求，服务端返回 `version > clientVersion` 的变更条目。

### 全有或全无策略

```
变更数 ≤ limit → 返回全部变更条目（增量同步）
变更数 > limit → 返回空数组（客户端应走全量同步）
```

### 水位线兼容性校验

```
clientVersion >= deleted   → 该版本之前的日志已被清理，返回空（走全量）
clientVersion >  version   → 客户端版本超前于服务端，返回空（走全量）
```

### 查询 Pipeline

```
Stage 1: $match — 定位 DID 文档
Stage 2: $addFields — 兼容性校验，不满足则清空 logs
Stage 3: $addFields — $filter 过滤出 version > clientVersion 的条目
Stage 4: $addFields — $size 计算过滤后条数
Stage 5: $addFields — (仅 limit > 0 时) 超限则清空 logs
Stage 6: $project — 最终投影
```

### 查询示例

```
DB logs = [v11:insert@A, v15:update@B, v20:delete@C, v50:update@A]

FindChangeLog(ctx, "group_1", clientVersion=10, limit=2)
  → 变更数=4 > limit=2 → 返回 Logs=[]（空数组，走全量）

FindChangeLog(ctx, "group_1", clientVersion=10, limit=10)
  → 变更数=4 ≤ limit=10 → 返回 Logs=[v11, v15, v20, v50]，Version=50
```

### Fast Path

`clientVersion==0 && limit==0` 时直接走 `FindOne`，跳过聚合管道。

---

## 四、版本控制完整流程

```
┌──────────┐
│  客户端   │  持有本地 version（初始=0）+ versionID
└────┬─────┘
     │
     │  ① 首次同步（clientVersion=0）
     ▼
┌──────────────────────────────────────────────────┐
│  FindChangeLog(did, clientVersion=0, limit=0)    │
│  → Fast path: 直接 FindOne 返回完整 VersionLog   │
│  → 客户端获得全量数据 + 最新 version/versionID   │
└──────────────────────────┬──────────────────────┘
                           │
                           │  客户端本地版本 = version
                           ▼
              ┌────────────────────────┐
              │   服务端数据发生变更     │
              │   (加好友/退群/改备注)  │
              └───────────┬────────────┘
                          │
                          │  ② IncrVersionLogBatch(did, eid, state)
                          ▼
┌──────────────────────────────────────────────────────────┐
│  FindOneAndUpdate 聚合 Pipeline（原子操作）              │
│                                                          │
│  Stage 1: 暂存 eids                                      │
│  Stage 2: 补零值（新文档初始化 version=1）                │
│  Stage 3: version += 1                                   │
│  Stage 4: $filter 剔除同 EID 旧条目（去重）               │
│  Stage 5: $concatArrays 追加新条目（version=新值）       │
│  Stage 6: $unset 移除临时字段                             │
│                                                          │
│  返回: version=43, versionID=ObjectId(...)               │
└──────────────────────────┬───────────────────────────────┘
                           │
                           │  ③ 通知推送（携带 version + versionID）
                           ▼
              ┌────────────────────────┐
              │  WebSocket / 通知       │
              │  friendVersion=43       │
              │  friendSortVersion=N    │
              │  versionID=xxx          │
              └───────────┬────────────┘
                          │
                          │  ④ 客户端收到通知，发起增量同步
                          ▼
┌──────────────────────────────────────────────────────────┐
│  FindChangeLog(did, clientVersion=42, limit=100)        │
│                                                          │
│  Stage 2: 兼容性校验                                      │
│    42 >= deleted(0)?  否                                  │
│    42 >  version(43)? 否                                  │
│  Stage 3: $filter → version > 42 的条目                   │
│  Stage 4: $size = 变更数                                  │
│  Stage 5: 变更数 ≤ 100 → 返回增量条目                     │
│           变更数 > 100 → 返回空数组（走全量）              │
│                                                          │
│  返回: Logs=[{EID, State, Version=43}], Version=43       │
└──────────────────────────┬───────────────────────────────┘
                           │
                           │  ⑤ 客户端应用增量变更
                           ▼
              ┌────────────────────────┐
              │  客户端更新本地数据      │
              │  本地 version = 43      │
              │  本地 versionID = xxx  │
              └────────────────────────┘
```

### 流程说明

| 步骤 | 操作 | 触发方 |
|------|------|--------|
| ① | 首次全量同步（clientVersion=0，走 Fast path） | 客户端 |
| ② | 数据变更时原子推进版本号 + 追加变更日志 | 服务端 RPC |
| ③ | 通过 WebSocket 推送通知，携带新 version + versionID | 服务端 |
| ④ | 客户端收到通知后，携带本地 clientVersion 发起增量同步 | 客户端 |
| ⑤ | 客户端应用增量变更，更新本地版本号 | 客户端 |

### 全量同步回退条件

以下任一条件满足时，`FindChangeLog` 返回空数组，客户端回退到全量同步：

| 条件 | 含义 |
|------|------|
| `clientVersion >= deleted` | 客户端版本落在已清理范围，旧日志已被删除 |
| `clientVersion > version` | 客户端版本超前于服务端（数据异常） |
| 变更数 > limit（limit > 0 时） | 客户端落后太多，增量同步代价大于全量 |

### 使用场景

| 服务 | 文件 | 说明 |
|------|------|------|
| **relation.rpc** | [relation_logic.go:519-592](file:///c:/Users/Administrator/Desktop/goim/im-rpc/relation/internal/logic/relation_logic.go#L519-592) | 好友/黑名单增量同步：客户端传 `version` + `versionID` 查询变更 |
| **group.rpc** | [group_logic.go:1082-1084](file:///c:/Users/Administrator/Desktop/goim/im-rpc/group/internal/logic/group_logic.go#L1082-1084) | 群成员变更后调用 `IncrVersionLog` 推进版本，通知携带 `sortVersion` + `version` + `versionID` |
| **conversation.rpc** | [conversation_logic.go:593](file:///c:/Users/Administrator/Desktop/goim/im-rpc/conversation/internal/logic/conversation_logic.go#L593) | 会话 upsert 后推进版本日志（Insert 或 Update） |
