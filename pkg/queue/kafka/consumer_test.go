package kafka

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/segmentio/kafka-go"

	"github.com/PaperMan11/goim/pkg/queue"
)

// commitRecorder 记录注入的提交动作：前 failures 次返回 err，之后成功并记录
// 提交位置（position = offset + 1，即"下一条待消费位点"）。
type commitRecorder struct {
	calls     int
	failures  int
	err       error
	positions []int64
}

func (r *commitRecorder) commit(_ context.Context, msgs ...kafka.Message) error {
	r.calls++
	if r.failures > 0 {
		r.failures--
		return r.err
	}
	for _, m := range msgs {
		r.positions = append(r.positions, m.Offset+1)
	}
	return nil
}

func newTestConsumer(fn func(ctx context.Context, msgs ...kafka.Message) error) *KafkaConsumer {
	return &KafkaConsumer{
		cfg:      KafkaConfig{Topic: "test-topic", GroupID: "test-group"},
		ctx:      context.Background(),
		commitFn: fn,
	}
}

func wrappedMessage(partition int, offset int64) *WrappedMessage {
	return &WrappedMessage{
		Message: kafka.Message{Topic: "test-topic", Partition: partition, Offset: offset},
	}
}

// fetch 模拟：同分区按 fetch 顺序落定基准位点
func fetch(c *KafkaConsumer, partition int, offsets ...int64) {
	for _, offset := range offsets {
		c.trackFetched(wrappedMessage(partition, offset))
	}
}

// 乱序完成（分片并发下 177 先于 175/176 处理完）：
// 高位点不得越过未处理消息提交，位点必须等待连续推进后一次性提交。
func TestCommitMessageContiguousOutOfOrder(t *testing.T) {
	rec := &commitRecorder{}
	c := newTestConsumer(rec.commit)
	fetch(c, 0, 175, 176, 177)

	ctx := context.Background()
	c.commitMessage(ctx, wrappedMessage(0, 177))
	c.commitMessage(ctx, wrappedMessage(0, 176))
	if len(rec.positions) != 0 {
		t.Fatalf("高位点先完成时不应提交任何位点, got %v", rec.positions)
	}

	c.commitMessage(ctx, wrappedMessage(0, 175))
	if len(rec.positions) != 1 || rec.positions[0] != 178 {
		t.Fatalf("连续推进后应一次性提交 178, got %v", rec.positions)
	}
}

// 顺序完成：每条消息处理完推进一个位点。
func TestCommitMessageInOrder(t *testing.T) {
	rec := &commitRecorder{}
	c := newTestConsumer(rec.commit)
	fetch(c, 0, 10, 11)

	ctx := context.Background()
	c.commitMessage(ctx, wrappedMessage(0, 10))
	c.commitMessage(ctx, wrappedMessage(0, 11))

	want := []int64{11, 12}
	if len(rec.positions) != len(want) {
		t.Fatalf("expected commits %v, got %v", want, rec.positions)
	}
	for i := range want {
		if rec.positions[i] != want[i] {
			t.Fatalf("expected commits %v, got %v", want, rec.positions)
		}
	}
}

// 迟到的小位点不得把已提交的大位点拉回（位点回退会导致重启后重复消费）。
func TestCommitMessageNoRegression(t *testing.T) {
	rec := &commitRecorder{}
	c := newTestConsumer(rec.commit)
	fetch(c, 0, 176, 177, 178)

	ctx := context.Background()
	c.commitMessage(ctx, wrappedMessage(0, 178)) // 无连续推进，跳过
	c.commitMessage(ctx, wrappedMessage(0, 177)) // 无连续推进，跳过
	c.commitMessage(ctx, wrappedMessage(0, 176)) // 连续推进到 179，一次提交

	if len(rec.positions) != 1 || rec.positions[0] != 179 {
		t.Fatalf("最终应只提交一次且位点为 179, got %v", rec.positions)
	}
}

// 提交失败时按退避重试，成功后位点正确；重试期间不影响状态推进。
func TestCommitMessageRetry(t *testing.T) {
	rec := &commitRecorder{failures: 2, err: errors.New("ILLEGAL_GENERATION")}
	c := newTestConsumer(rec.commit)
	fetch(c, 0, 10)

	c.commitMessage(context.Background(), wrappedMessage(0, 10))

	if rec.calls != 3 {
		t.Fatalf("expected 3 commit attempts (2 failures + 1 success), got %d", rec.calls)
	}
	if len(rec.positions) != 1 || rec.positions[0] != 11 {
		t.Fatalf("重试成功后应提交位点 11, got %v", rec.positions)
	}
}

// reader 关闭（io.ErrClosedPipe）时立即放弃重试，避免停机时无谓阻塞。
func TestCommitMessageAbortOnClosedReader(t *testing.T) {
	rec := &commitRecorder{failures: 99, err: io.ErrClosedPipe}
	c := newTestConsumer(rec.commit)
	fetch(c, 0, 10)

	c.commitMessage(context.Background(), wrappedMessage(0, 10))

	if rec.calls != 1 {
		t.Fatalf("reader 已关闭时应只尝试 1 次, got %d", rec.calls)
	}
	if len(rec.positions) != 0 {
		t.Fatalf("不应有成功提交, got %v", rec.positions)
	}
}

// 多分区独立维护连续进度，互不干扰。
func TestCommitMessagePerPartition(t *testing.T) {
	rec := &commitRecorder{}
	c := newTestConsumer(rec.commit)
	fetch(c, 0, 10)
	fetch(c, 1, 20)

	ctx := context.Background()
	c.commitMessage(ctx, wrappedMessage(1, 20))
	c.commitMessage(ctx, wrappedMessage(0, 10))

	want := []int64{21, 11}
	if len(rec.positions) != 2 || rec.positions[0] != want[0] || rec.positions[1] != want[1] {
		t.Fatalf("expected commits %v, got %v", want, rec.positions)
	}
}

// 同 key 必须路由到同一分片（保证同会话串行），且结果稳定、落在分片范围内。
func TestShardIndexStableAndInRange(t *testing.T) {
	c := &KafkaConsumer{shardChans: make([]chan queue.Message, 4)}

	msg := &WrappedMessage{Message: kafka.Message{Key: []byte("si_it_a_1_it_b_2")}}
	first := c.shardIndex(msg)
	if first < 0 || first >= 4 {
		t.Fatalf("shard index out of range: %d", first)
	}
	for i := 0; i < 100; i++ {
		if got := c.shardIndex(msg); got != first {
			t.Fatalf("same key must map to same shard, want %d got %d", first, got)
		}
	}
}

// 不同 key 应散列到多个分片，无 key 消息回退按 value 散列且不越界。
func TestShardIndexSpreadAndNoKey(t *testing.T) {
	const shards = 4
	c := &KafkaConsumer{shardChans: make([]chan queue.Message, shards)}

	seen := make(map[int]struct{})
	for i := 0; i < 50; i++ {
		msg := &WrappedMessage{Message: kafka.Message{Key: []byte("conv-" + string(rune('a'+i%26)) + string(rune('a'+i)))}}
		idx := c.shardIndex(msg)
		if idx < 0 || idx >= shards {
			t.Fatalf("shard index out of range: %d", idx)
		}
		seen[idx] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("keys should spread across shards, got %v", seen)
	}

	noKey := &WrappedMessage{Message: kafka.Message{Value: []byte("payload")}}
	if idx := c.shardIndex(noKey); idx < 0 || idx >= shards {
		t.Fatalf("no-key message shard index out of range: %d", idx)
	}
}

func TestModeName(t *testing.T) {
	c := &KafkaConsumer{cfg: KafkaConfig{Consumers: 2, Processors: 4}}
	if got := c.modeName(); !strings.Contains(got, "shared-channel") || !strings.Contains(got, "processors: 4") {
		t.Fatalf("unexpected mode name: %s", got)
	}

	c.cfg.KeySharding = true
	if got := c.modeName(); !strings.Contains(got, "key-sharding") || !strings.Contains(got, "shards: 4") {
		t.Fatalf("unexpected mode name: %s", got)
	}

	c.cfg = KafkaConfig{OrderCommit: true}
	if got := c.modeName(); !strings.Contains(got, "order-commit") {
		t.Fatalf("unexpected mode name: %s", got)
	}
}
