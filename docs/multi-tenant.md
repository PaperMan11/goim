# GoIM 多租户（多 App）改造方案

> 状态：**设计中**（未动代码）
> 创建日期：2026-10-28

## 1. 背景与目标

当前 GoIM 是**纯独立部署模型**：一套服务 = 一个 App。所有用户、群组、消息、关系链、会话、缓存、推送路由均是全局无隔离的。

目标：**一套 GoIM 服务支持多个 App 共享部署，数据物理隔离、认证链路独立、推送不串 App**。

### 目标场景

```
goim-shared-cluster
├── App A (社交 IM)     ← 共享同一套二进制 / Mongo / Redis / Kafka
├── App B (客服 IM)
└── App C (内部协作 IM)
```

---

## 2. 现状诊断：7 个零隔离硬编码点

| # | 层 | 位置 | 当前状态 |
|---|---|---|---|
| 1 | JWT | `pkg/utils/jwtx/jwt.go` `TokenClaims` | claims 只有 `userId / uuid / platformId / deviceId`，无 appID |
| 2 | API Middleware | `im-api/internal/middlewares/auth.go` | 解析后只注入 `OpUserID / OpUserPlatform`，不设 App 上下文 |
| 3 | mcontext | `pkg/mcontext/mcontext.go` | 无 appID 的 `Get/Set` 方法 |
| 4 | Mongo Collection 名 | `pkg/storage/model/*.go` `CollectionName()` | 全局单 collection（`users`、`groups`、`msgDoc`、`conversation`…） |
| 5 | Redis Key | `pkg/storage/mongo/**/constant.go`、`pkg/storage/token/key.go` | 无 App 前缀（`token:user:{userID}`、`mongo:user:info:{userID}`） |
| 6 | conversationID 生成 | `pkg/msgprocessor/conversation.go` | `"si_" + sort(id1, id2)`，无 App 维度 |
| 7 | Admin 白名单 | IM Admin 列表（`auth.yaml`） | 全局一份管理员列表，无 App 归属 |

**结论**：隔离缺失贯穿认证、存储、路由三层，改造必须分阶段逐层推进，不能跳跃。

---

## 3. 核心架构决策（已拍板）

### 3.1 Decision A：userID 保持全局 UUID 唯一

| 方案 | 评估 | 结论 |
|---|---|---|
| ❌ 复合键 `{appID}_{userID}` 作为身份 | 所有 proto 的 `sendID / recvID / groupID` 改成复合类型；消息去重、会话生成、推送路由、关系链查询全部受影响，破坏性 ≥ 300 个文件 | 否决 |
| ✅ userID 保持 UUID 全局唯一 | 零 proto 破坏性变更。跨 App 冲突理论上不可能（UUID v4 碰撞概率可忽略） | **采纳** |

tenant 隔离靠外层三个维度：
1. JWT claims 加 `appID` → 一个 App 一条独立 Token 链
2. Mongo collection 名前缀化 → 每个 App 物理数据隔离
3. Redis key 前缀化 → 每个 App 缓存空间互不污染

`conversationID = "si_" + sort(uuid1, uuid2)` **保持不变**。因为 UUID 全局唯一，不会跨 App 冲突。

### 3.2 Decision B：共享 DB + tenant 级 Collection 前缀

```
goim_db                         ← 一个库
├── appA_users                  ← 每个 App 一套 collection
├── appA_conversations
├── appA_msg_conversation123
├── appA_groups
├── appA_group_members
├── appA_friends
├── appA_blacks
├── appA_friend_requests
├── appA_group_requests
├── appA_seq_conversations
├── appA_seq_users
├── appA_version_logs
├── appB_users
├── ...
```

vs 备选：共享 collection + `appID` 字段。

| 对比项 | Collection 前缀（采纳） | 共享 collection + appID 字段 |
|---|---|---|
| 查询性能 | 天然隔离，索引只覆盖本 App 数据 | 所有 App 挤在一张表，索引要带 appID 前缀 |
| 数据安全 | 物理隔离，脚本误操作不会跨 App | 必须靠所有查询都带 appID 条件，漏一个就是越权 |
| 迁移成本 | 一次性迁移脚本 | 存量数据逐条补 appID |
| App 数量上限 | ~20 个 collection/App × N 个 App | 单 collection 无限增长 |
| Mongo max collection 限制 | 单个 DB 默认 ≤ 12000 个 collection（MongoDB 限制），20 App × 600 = 12000，**App 数量 ≤ 200** 时安全 | 无此限制 |

> **临界上限估算**：200 个 App × (8 个基础 collection + 动态消息 collection) ≈ 200 × 20 = 4000 collection，远低于 Mongo 12000 默认上限。

### 3.3 Decision C：JWT Secret 全局共享 + Issuer 隔离

| 方案 | 评估 | 结论 |
|---|---|---|
| ❌ 每 App 独立 secret | 维护成本高（新 App 要注册 secret、rotating 要协调）；ParseToken 不知道用哪个 secret | 否决 |
| ✅ 全局共享 secret + issuer == appID | Token 解析时额外校验 issuer 是否等于当前 App；跨 App Token 即使知道 secret 也通不过 issuer 校验 | **采纳** |

```go
// jwtx/jwt.go 改造后示意
claims := NewTokenClaims(TokenTypeAccess).
    SetAppID(appID).                    // 新增
    SetIssuer(appID).                   // 复用 RegisteredClaims.Issuer
    SetUserIDStr(userID).
    ...

// 校验时
if claims.RegisteredClaims.Issuer != currentAppID {
    return ErrTokenInvalid
}
```

---

## 4. 分阶段改造计划

### 阶段依赖关系

```
Phase 1 (Token & Context)   ← 最前置，改完立刻编译通过
        │
        ├── Phase 2 (Mongo Collection 前缀化) ──┐
        │                                        ├── 可并行
        └── Phase 3 (Redis Key 前缀化) ────────┘
        │
        ├── Phase 4 (消息 & 推送路由隔离)
        │
        └── Phase 5 (租户管理控制台) ← 可选
```

---

### Phase 1：Token & Context 打底

**目标**：让 JWT 和上下文能"带 appID 跑"。**不动 proto 核心结构、不动存储、不动业务逻辑**。完成后所有现有单测继续通过。

| # | 任务 | 涉及文件 | 改动细节 |
|---|---|---|---|
| 1.1 | JWT Claims 加 AppID 字段 | `pkg/utils/jwtx/jwt.go` | `TokenClaims` 结构体加 `AppID string`；新增 `SetAppID(appID)` 链式方法 |
| 1.2 | 所有 Token 生成函数加 appID 参数 | 同上 | `GenerateAccessToken / GenerateRefreshToken / GenerateAdminToken` 签名加 `appID string` 入参；**同时**把 `SetIssuer(appID)` 写进去（利用 JWT RegisteredClaims.Issuer 天然字段做二次隔离） |
| 1.3 | parse_token 返回 AppID | `pkg/protocol/auth/auth.proto` + `im-rpc/auth/internal/logic/parse_token.go` | `ParseTokenResp` 新增 `string appID = 3;`（tag 3，已有 userID=1, expireTimeSeconds=2 不动） |
| 1.4 | 重新生成 pb.go | `pkg/protocol/gen_grpc.sh` | 执行 protoc 重新生成 `auth.pb.go` 和 `auth_grpc.pb.go` |
| 1.5 | constant 加 `OpAppID` key | `pkg/protocol/constant/context.go` | `OpAppID = "opAppID"`（与已有 `OpUserID / OpUserPlatform` 对齐） |
| 1.6 | mcontext 加 AppID 读写 | `pkg/mcontext/mcontext.go` | 新增 `GetOpAppIDFromContext(ctx)` / `SetOpAppIDInContext(ctx, appID)` |
| 1.7 | API Middleware 注入 AppID | `im-api/internal/middlewares/auth.go` | 从 `ParseTokenResp.AppID` 取值，`c.Set(constant.OpAppID, resp.AppID)` 写入 gin context |
| 1.8 | Admin Token 链同步改造 | `im-rpc/auth/internal/logic/get_admin_token.go` | `GetAdminToken` 的 `req` 需带 `appID`（proto 扩展），JWT 生成时写入 |
| 1.9 | get_user_token 链同步改造 | `im-rpc/auth/internal/logic/get_user_token.go` | 同上 |

**Proto 改动量**：仅 `auth.proto` 的 `ParseTokenResp` 加一个字段。**零破坏性**（proto3 新增字段向后兼容）。

**估算文件数**：~15 个。

**验收标准**：
- `go build ./...` 通过
- 所有现有单测通过
- 手动发一个 JWT token，解出来能看到 `appID` 和 `issuer`
- API middleware parse 后 `gin.Context` 能取到 `OpAppID`

---

### Phase 2：Mongo Collection 前缀化

**目标**：MongoDB 层真正物理隔离。改造所有 Model 的 Collection 选择逻辑。

| # | 任务 | 涉及文件 | 改动细节 |
|---|---|---|---|
| 2.1 | CollectionName 改签名 | `pkg/storage/model/*.go`（user.go, conversation.go, friend.go, group.go, request.go, seq.go, message.go, version.go, webhook.go） | 所有 `CollectionName()` 改为 `CollectionName(appID string) string { return appID + "_" + baseName }`。`baseName` 保留当前全局名（`users`, `groups`, `msgDoc` …） |
| 2.2 | Model 方法统一加 appID | `pkg/storage/mongo/*/*.go` | 所有 Model 方法签名加 `ctx, appID string, ...`。内部调用 `CollectionName(appID)` 取 collection |
| 2.3 | 所有业务调用点传 appID | `im-rpc/*/internal/logic/*.go` | 每个 logic 方法入口 `appID := mcontext.GetOpAppIDFromContext(ctx)`，传入下游 |
| 2.4 | MsgDoc 动态 collection | `pkg/storage/model/message.go` + `pkg/storage/mongo/msg/msg.go` | `msg_{conversationID}` → `{appID}_msg_{conversationID}`。按 prefix 构造 |
| 2.5 | 升级脚本 | `scripts/mongo_migrate.go`（新增） | 给存量 collection 批量 rename 加 `{appID}_` 前缀。支持 dry-run、按前缀 rename |
| 2.6 | Service 启动时预建 collection（可选） | `im-rpc/*/internal/svc/service_context.go` | 首次启动时 `db.CreateCollection` 确保 collection 存在（Mongo 也可首次写入自动创建，但预建能及早暴露权限问题） |

**MsgDoc 特殊性**：当前每个 conversation 一个独立 collection（`msg_{convID}`），数量随会话数增长。加前缀后变成 `{appID}_msg_{convID}`，同一 App 的所有动态 collection 前缀相同，迁移脚本可以按前缀批量 rename。

**估算文件数**：~30 个。

**验收标准**：
- 两个 App 各跑一次注册 + 发消息 + 拉消息
- Mongo 里能看到 `appA_*` 和 `appB_*` 两套 collection
- App A 的消息不出现在 App B 的 collection 里
- 升级脚本能正确 rename 存量 collection

---

### Phase 3：Redis Key 前缀化

**目标**：Redis 缓存 / Token / 分配器 key 全部加 App 前缀，与 Phase 2 同构，可并行。

| # | 任务 | 涉及文件 | 改动细节 |
|---|---|---|---|
| 3.1 | Key 常量加占位 | `pkg/storage/mongo/**/constant.go`（8 个文件） | 所有 key 格式串第一个 `%s` 改为 appID。例：`"mongo:user:info:%s"` → `"mongo:%s:user:info:%s"` |
| 3.2 | Key 构造函数改签名 | 同上 + `pkg/storage/token/key.go` + `pkg/storage/redis/allocator/keys.go` | `func KeyUserInfo(appID, userID string) string` |
| 3.3 | Cache 读写传 appID | `pkg/storage/redis/cache.go` + 所有 `cached_*_model.go` | `CacheGet(ctx, appID, key, dest)` / `CacheSet(ctx, appID, key, value, ttl)` |
| 3.4 | Token Redis Store 隔离 | `pkg/storage/token/redis_store.go` | key 统一加 appID 前缀（`token:%s:user:%s:tokens`）。踢出、查询、delete channel 都要隔离 |
| 3.5 | Seq Allocator 隔离 | `pkg/storage/redis/allocator/seq_allocator.go` + Lua 脚本 | 分配器 key 加 appID，`allocate_seq.lua` / `commit_seq.lua` 入参加 appID |
| 3.6 | Redis Cluster 跨 slot 保护 | 同上 | 同一 operation 涉及的多 key 必须用相同 hash tag：`{appA}:mongo:user:info:{uid}`、`{appA}:token:user:{uid}` → Redis Cluster 会将带相同 hash tag 的 key 落同一 slot |

**Redis Cluster 关键约束**：之前项目经验（Lessons Learned：Lua 脚本在 Redis Cluster 必须操作同一 key，否则 CROSSSLOT），加 appID 后多 key 操作要注意 hash tag 一致性。

**估算文件数**：~20 个。

**验收标准**：
- 两个 App 各自操作 Redis，KEYS 能看到 `mongo:appA:*` 和 `mongo:appB:*` 两套
- 执行 Lua 脚本不触发 CROSSSLOT 错误
- Token 踢下线只影响同 App 用户

---

### Phase 4：消息 & 推送路由隔离

**目标**：确保 WebSocket 连接注册、消息推送、离线推送都不会跨 App。

| # | 任务 | 涉及文件 | 改动细节 |
|---|---|---|---|
| 4.1 | MsgData 加 appID | `pkg/protocol/sdkws/sdkws.proto` | `MsgData` 新增 `string appID = 100;`（用高位 tag，避免破坏现有字段解析顺序） |
| 4.2 | MsgData pb.go 重新生成 | 同上 | protoc 生成新的 `sdkws.pb.go` |
| 4.3 | 消息入口注入 appID | `im-rpc/msg/internal/logic/message_logic.go` | 发送消息时从 context 取 appID，写入 `MsgData.AppID` |
| 4.4 | 消息入库携带 appID | `im-msgtransfer/handler.go` | 从 Kafka 消费的 MsgData 应已带 appID；落 Mongo 时 appID 用在 collection 名（Phase 2 已覆盖） |
| 4.5 | WebSocket 连接注册隔离 | `im-msggateway/internal/conn_manager.go` + `ws_connection.go` | 当前连接标识是 `userID + platformID`；改为 **`appID + userID + platformID`** 三元组 key |
| 4.6 | 推送分发校验 appID | `im-msggateway/internal/hub_server.go` + `im-push/internal/handler.go` | 推送时从 MsgData 取 appID，与目标连接的 appID 比对，不一致则跳过（防止串 App） |
| 4.7 | SeqUser / SeqConversation 隔离 | `pkg/storage/mongo/sequser/` + `seqconversation/` | Phase 2 已用 collection 前缀天然隔离；确认 model 内部 doc 不需要额外 appID 字段 |
| 4.8 | conversationID 不变 | `pkg/msgprocessor/conversation.go` | **保持 `"si_" + sort(uuid1, uuid2)` 不变**。UUID 全局唯一，不需要 App 维度 |

**WebSocket 连接注册特别注意**：多端登录时，同一 userID 在不同 App 会各开各的连接表，互不干扰。踢下线操作也只踢对应 App 下的连接。

**估算文件数**：~15 个 + 1 proto。

**验收标准**：
- App A 的 user 登录 WebSocket，App B 发消息推不过来
- App A 同一 user 在 Android / iOS / Web 三端登录，三端都能收到 App A 的消息
- 踢 App A 的 user，App B 的同 userID 不受影响

---

### Phase 5：租户管理控制台（可选）

**目标**：运维层能创建 App、配置管理员、设置 per-App Admin Secret。

| # | 任务 | 说明 |
|---|---|---|
| 5.1 | App 注册表 | Mongo 新增 collection `apps`：`{appID string, appName string, adminUserIDs []string, createdAt int64, status int}` |
| 5.2 | Admin Token 链 per-App Secret | Phase 1 是全局共享 secret，这里可以升级：`apps` 表加 `adminSecret` 字段；`get_admin_token` 按 appID 查对应 secret 校验 |
| 5.3 | requireUserIsAdminOfApp | 当前 `requireUserIsAdmin` 查全局 admin 列表；改为查 apps 表里 appID 对应的 adminUserIDs |
| 5.4 | 管理 HTTP API | 新增一组 internal endpoint（只对 Admin App 开放）：`POST /internal/app/create`、`GET /internal/app/list`、`POST /internal/app/add_admin`、`POST /internal/app/delete` |
| 5.5 | 配置面板（可选） | 后续做 Web UI，非必需 |

**估算文件数**：~10 个（可选）。

---

## 5. 改造影响范围总结

```
改动量金字塔
│                    ┌─────────────────┐
│                    │ Phase 5 控制台   │  ~10 文件（可选）
│                   ──┼─────────────────┼──
│                  ── │ Phase 4 消息隔离 │  ~15 文件 + 1 proto
│                 ──  ┼─────────────────┼──
│                ──   │ Phase 2+3 存储层 │  ~50 文件（核心工作量）
│               ──    ┼─────────────────┼──
│              ──     │ Phase 1 Token层  │  ~15 文件（最前置）
│─────────────────────────────────────────────
```

### Proto 改动汇总（仅 3 处）

| Proto | 改动 | 兼容性 |
|---|---|---|
| `auth.proto` → `ParseTokenResp` | 新增 `string appID = 3;` | ✅ 向后兼容（proto3 新增字段） |
| `auth.proto` → `GetAdminTokenReq` | 新增 `string appID = 3;` | ✅ 向后兼容 |
| `sdkws.proto` → `MsgData` | 新增 `string appID = 100;` | ✅ 向后兼容（高位 tag 不影响现有解析） |

**关键原则**：所有 proto 改动都是**新增字段**，不重命名、不删除、不调整已有 tag 号。

### 不改动的东西

- `userID` / `groupID` 生成逻辑（保持 UUID）
- `conversationID` 拼接逻辑（`"si_" + sort(id1,id2)` 不变）
- 业务接口 proto 的入参结构（所有 `/api/v1/*` 入参不变）
- Kafka topic 结构（共享 Kafka 按 appID 做消息过滤即可；如需物理隔离可再拆 topic）
- API 路由结构（不新增路由前缀，App 归属从 Token 解析得出）

---

## 6. 风险点与防护

| # | 风险 | 影响 | 防护措施 |
|---|---|---|---|
| 1 | **存量数据迁移**：线上已有数据，Mongo 需要 rename collection | 数据不可用 | ① 先测试环境演练；② 写支持 dry-run 的 rename 脚本；③ 低峰期执行；④ 执行前全量备份 |
| 2 | **MsgDoc 动态 collection 膨胀**：每个会话一个 collection，加 App 前缀后 App 数量 × 会话数 | Mongo 连接/内存压力 | ① 上线前评估每个 App 的会话数上限；② 可选：高频冷会话定期归档到 `{appID}_msg_archive_{convID}` |
| 3 | **Redis Cluster CROSSSLOT**：改造后同一逻辑操作的 key 可能跨 slot | Lua 脚本执行失败 | 所有 multi-key 操作加 `{appID}` hash tag 强制同 slot（Redis Cluster 标准做法） |
| 4 | **WebSocket 连接切换**：客户端换 App 登录忘记断旧连接 | 旧连接继续收/发消息 | SDK 层保证：换 App 调用必须先 close 旧连接再 connect 新 App |
| 5 | **Token 跨 App 使用**：攻击者拿到 App A 的 Token 试图访问 App B | 认证绕过 | JWT 校验 issuer == 当前 App；中间件层显式 reject |
| 6 | **appID 缺失 panic**：某个 logic 忘记从 context 取 appID，拿到空字符串 | collection 名变成 `_users` 污染全局 | ① ServiceContext 加 debug 模式断言 appID 非空；② 单测 mock context 必须注入 appID |
| 7 | **Admin Secret 每 App 独立的密钥管理**（Phase 5） | 运维复杂度 | 初期共享一个 secret 靠 issuer 隔离够用；后期引入 Vault / KMS 管理 per-App secret |

---

## 7. 里程碑 & 验收 Checklist

### Phase 1 完成标志

- [ ] `jwtx.Generate*Token` 全部加 appID 参数
- [ ] `ParseTokenResp` proto 返回 appID
- [ ] API middleware 自动注入 `OpAppID` 到 context
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 通过

### Phase 2 完成标志

- [ ] Mongo 新写入数据落 `{appID}_*` collection
- [ ] 升级脚本能 rename 存量 collection
- [ ] 两个 App 各注册 1 用户 + 互发消息，Mongo 里数据完全隔离

### Phase 3 完成标志

- [ ] 所有 Redis key 带 appID 前缀
- [ ] Lua 脚本在 Redis Cluster 执行不报 CROSSSLOT
- [ ] 踢 App A token，App B 同 user 不受影响

### Phase 4 完成标志

- [ ] WebSocket 连接注册三元组（appID + userID + platformID）
- [ ] App A user 完全收不到 App B 的推送
- [ ] App A 三端登录都能收到 App A 消息

### Phase 5 完成标志（可选）

- [ ] 创建 App 能正确初始化 collection 和 admin
- [ ] per-App Admin Secret 校验生效
- [ ] 运维面板能列出所有 App 及其统计

---

## 8. 附录：关键文件速查

| 主题 | 文件 |
|---|---|
| JWT Claims | `pkg/utils/jwtx/jwt.go` |
| API 认证中间件 | `im-api/internal/middlewares/auth.go` |
| Context 读写 | `pkg/mcontext/mcontext.go` |
| Proto：Auth | `pkg/protocol/auth/auth.proto` |
| Proto：MsgData | `pkg/protocol/sdkws/sdkws.proto` |
| Mongo Model CollectionName | `pkg/storage/model/*.go` |
| Mongo 各 Model 实现 | `pkg/storage/mongo/*/*.go` |
| Redis Key 常量 | `pkg/storage/mongo/**/constant.go` |
| Token Key | `pkg/storage/token/key.go` |
| Seq Allocator | `pkg/storage/redis/allocator/seq_allocator.go` |
| conversationID 生成 | `pkg/msgprocessor/conversation.go` |
| MsgGateway 连接管理 | `im-msggateway/internal/conn_manager.go` |
| MsgGateway 推送分发 | `im-msggateway/internal/hub_server.go` |
| Admin Token 逻辑 | `im-rpc/auth/internal/logic/get_admin_token.go` |
| IM Admin 校验 | `im-rpc/auth/internal/logic/logic.go` 的 `requireUserIsAdmin` |
