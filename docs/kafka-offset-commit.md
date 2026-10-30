# Kafka 消费位点提交：位点回退与重复消费问题

> 模块：`pkg/queue/kafka` · 涉及：`im-msgtransfer`、`im-push`
> 日期：2026-10-30

## 1. 背景

消息消费链路采用「1 个 fetchLoop + N 个 consumeLoop」的并发消费模型（key 分发 / 共享 channel 两种模式），消息处理完成后逐条提交位点。实测发现：**服务重启后会重复消费已处理过的消息**（同一 offset 再次进入 `consumeMsg`）。

## 2. 现象

```
14:08:15.266  received msg ...                     （offset 177 开始处理）
14:08:16.269  committed  offset: 177               ← 提交位置 = 178
14:08:17.275  committed  offset: 176               ← 提交位置 = 177，位点被拉回
```

此时重启，服务从位点 177 重新消费——**已处理过的 offset 177 被再次投递**。

## 3. 根因分析

位点 `P` 的语义是「下一条从 P 开始消费」。只有 `commit(P)` 成功落到 broker（`__consumer_offsets`），broker 才认为 `< P` 的消息全部消费完。重复消费有两个独立根因：

### 根因一：rebalance 窗口内提交失败

- kafka-go 同步提交内部仅重试 3 次（100ms~5s 退避，见 `reader.go` 的 `commitOffsetsWithRetry`）；
- 反复重启时，同组新旧实例交替进出 consumer group，broker 返回 `ILLEGAL_GENERATION` / `REBALANCE_IN_PROGRESS`；
- 3 次重试耗尽 → 位点原地不动 → 重启重投。失败日志是 error 级别（`Commit failed`），容易被 info 级别日志过滤忽略。

### 根因二：并发提交导致位点回退（更隐蔽）

- 并发消费下，消息**处理完成顺序 ≠ fetch 顺序**（如 176 走落库攒批路径慢了 1 秒）；
- 各 consumeLoop 独立提交 `msg.Offset + 1`，broker 保存**最后写入值**；
- 后完成的小位点覆盖先提交的大位点 → 位点回退 → 重启从回退处重复消费。
- 共享 channel 模式（如 msg-persist-topic 消费者）同样存在该隐患。

> 说明：kafka-go 成功提交写入的是 `msg.Offset + 1`。只要提交成功落到 broker，重启绝无可能重投该消息——「重启后重投 offset X」即证明 X 的提交从未成功落库。

## 4. 解决方案：按分区连续推进提交

`pkg/queue/kafka/consumer.go`，三层防御：

### 4.1 trackFetched：fetch 时落定基准位点

```go
// 同分区 fetch 严格有序，首个 fetch 到的消息位点即为本进程的最小未提交位点
func (c *KafkaConsumer) trackFetched(msg *WrappedMessage)
```

基准必须在 **fetch 时**创建，而不是首次提交时。否则若高位点先处理完（分片并发下常见），会以它为基准越过尚未处理的消息提交位点——把「重复」换成更严重的「丢失」。

### 4.2 commitMessage：只提交最大连续位置

```go
st.done[offset] = struct{}{}      // 记录已处理完成的位点
next := st.next                   // 从基准/上次提交位置向后找连续区间
for { if _, ok := st.done[next]; !ok { break }; delete(st.done, next); next++ }
// next 未推进 → 跳过提交（等待慢分片）；推进 → 提交 next
```

保证：提交位点**单调递增**（不回退）、**永不越过未处理完成的消息**（不丢）。

### 4.3 commitWithRetry：应用层退避重试

- kafka-go 内部 3 次重试失败后，应用层再重试最多 5 次（200ms 递增退避），覆盖 rebalance 窗口；
- `io.ErrClosedPipe` / `context.Canceled`（reader 已关闭）立即放弃；
- 成功打 debug 日志（含 partition/position），失败打 error 日志（含 attempt n/5），计入 `KafkaConsumerCommitErrorsTotal` 指标。

### 4.4 可测试性改造

`commitMessage` 原直接调用具体类型 `*kafka.Reader.CommitMessages`，无法 mock。抽出注入点：

```go
// KafkaConsumer
commitFn func(ctx context.Context, msgs ...kafka.Message) error  // 默认 reader.CommitMessages
```

测试注入录制器验证提交位置序列，生产行为零变化。

## 5. 方案对比

核心取舍：**位点回退换重复、位点超前换丢失**。IM 场景丢失远比重叠严重，任何方案必须守住「位点不越过未处理消息」。

| 方案 | 原理 | 优点 | 缺点 | 结论 |
|---|---|---|---|---|
| **连续推进提交**（当前） | 只提交最大连续已处理位置 | 不丢不重、保留并发、实现小 | 每条消息一次提交 RTT | ✅ 采用 |
| OrderCommit 串行 | 处理 + 提交全串行（`commitInOrder`） | 语义最简单 | 无并发，吞吐差 | 保留为可配模式 |
| `CommitInterval` 异步批量 | 间隔提交当前水位 | 提交开销最小 | 崩溃丢失整个间隔的提交；并发下同样回退 | 不采用 |
| Pause/Resume 分区 | 慢分片暂停分区等追平 | 无回退 | kafka-go 支持有限，状态机复杂 | 过度设计 |
| 手动 Assign（1 loop = 1 分区） | 每分区独占消费者，天然串行 | 模型最干净（RocketMQ queue 模式） | 并发上限 = 分区数；脱离 group 失去自动负载均衡 | 中期可选重构 |
| Kafka 事务 EOS | read_committed 事务消费 | 严格一次 | 只覆盖 Kafka→Kafka 管道，覆盖不到落库/推送 | 不适用 |
| 业务幂等兜底 | seq upsert + 客户端去重 | 任何方案的最终防线 | 非传输层解法 | ✅ 必须保留 |

## 6. 测试覆盖

`pkg/queue/kafka/consumer_test.go`（通过 `commitFn` 注入 mock）：

| 用例 | 验证点 |
|---|---|
| `TestCommitMessageContiguousOutOfOrder` | 177→176→175 乱序完成：高位点先完成不提交，连续推进后一次提交 178 |
| `TestCommitMessageInOrder` | 顺序完成逐条推进 |
| `TestCommitMessageNoRegression` | 迟到小位点不回退已提交的大位点 |
| `TestCommitMessageRetry` | 提交失败退避重试后成功 |
| `TestCommitMessageAbortOnClosedReader` | reader 关闭立即放弃 |
| `TestCommitMessagePerPartition` | 多分区独立推进 |
| `TestShardIndexStableAndInRange` / `SpreadAndNoKey` | 同 key 固定分片、均匀散列、无 key 回退 value |
| `TestModeName` | 三种消费模式启动日志 |

## 7. 残余风险与兜底

Kafka 是 at-least-once 语义，「处理完成」与「提交成功」是两个独立存储上的动作，间隙永远存在：

1. **崩溃式退出**（kill -9 / panic / 断电）：无 drain 机会；且 kafka-go 预取到 reader 内部缓冲的消息应用层未见过，无法提交，重启必重投；
2. **优雅停机残留**：drain 只覆盖已进入应用层 channel 的消息，drain 期间被强杀或 commit 仍失败时位点停留。

因此重复无法在语义上归零，由业务幂等兜底：

- `AddMsgs` 按 seq upsert——重复落库无副作用；
- 客户端按 seq 去重——重复推送对用户无感；
- `SetConversationHasReadSeq` 天然幂等（已读位点只前进不后退）。

**分工原则：传输层保证 at-least-once，业务层保证幂等。**

## 8. 排查指引

- 确认提交是否成功：debug 日志 `Kafka consumer committed ... partition: X, position: Y`；
- 提交失败：error 日志 `Kafka consumer commit failed (attempt n/5)` / `commit aborted`，关注是否集中在重启/发布时段（rebalance 窗口）；
- 正常情况应看到 `commit skipped, waiting for contiguous progress`（高位点先完成的正常等待），其后的连续推进提交会覆盖全部已处理位点；
- 指标：`KafkaConsumerCommitErrorsTotal` 持续增长说明 group 频繁 rebalance 或 broker 异常。
