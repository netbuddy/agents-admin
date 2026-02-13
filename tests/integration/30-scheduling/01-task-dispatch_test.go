// Package integration 调度分配集成测试
//
// 测试范围：调度器将 queued 状态的 Run 分配到可用的 Node，并通知节点管理器
//
// 测试用例（对应文档 TC-SCHEDULE-001 ~ TC-SCHEDULE-009）：
//   - TC-SCHEDULE-001: 基本调度
//   - TC-SCHEDULE-002: 无可用节点
//   - TC-SCHEDULE-003: 标签匹配
//   - TC-SCHEDULE-004: 标签不匹配
//   - TC-SCHEDULE-005: 负载均衡
//   - TC-SCHEDULE-008: 幂等性
//   - TC-SCHEDULE-009: Redis 消息确认
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"agents-admin/internal/apiserver/scheduler"
	"agents-admin/internal/config"
	"agents-admin/internal/shared/infra"
	"agents-admin/internal/shared/model"
	"agents-admin/internal/shared/queue"
	"agents-admin/internal/shared/storage"
)

var (
	testStore *storage.PostgresStore
	testRedis *infra.RedisInfra
	idSeq     uint32
)

func mustLabelsJSON(t *testing.T, m map[string]string) json.RawMessage {
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal labels failed: %v", err)
	}
	return b
}

func uniqueID(prefix string) string {
	seq := atomic.AddUint32(&idSeq, 1) % 1000
	base := prefix
	base = strings.ReplaceAll(base, "_", "-")
	if len(base) > 8 {
		base = base[:8]
	}
	base = strings.Trim(base, "-")
	if base == "" {
		base = "id"
	}
	return fmt.Sprintf("%s-%s-%03d", base, time.Now().Format("150405"), seq)
}

func startScheduler(ctx context.Context, s *scheduler.Scheduler, timeout time.Duration) (context.CancelFunc, <-chan struct{}) {
	schedCtx, cancel := context.WithTimeout(ctx, timeout)
	done := make(chan struct{})
	go func() {
		s.Start(schedCtx)
		close(done)
	}()
	return cancel, done
}

func stopScheduler(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	cancel()
	select {
	case <-done:
		return
	case <-time.After(8 * time.Second):
		t.Fatalf("scheduler did not stop in time")
	}
}

func newTestScheduler(store storage.PersistentStore, schedulerQueue queue.SchedulerQueue, nodeQueue queue.NodeRunQueue) *scheduler.Scheduler {
	cfg := scheduler.DefaultConfig()
	cfg.NodeID = uniqueID("schedule")
	cfg.Redis.ReadTimeout = 200 * time.Millisecond
	cfg.Redis.ReadCount = 10
	cfg.Fallback.Interval = 200 * time.Millisecond
	cfg.Fallback.StaleThreshold = 24 * time.Hour
	return scheduler.NewSchedulerWithConfig(store, schedulerQueue, nodeQueue, cfg)
}

func newTestSchedulerWithChain(store storage.PersistentStore, schedulerQueue queue.SchedulerQueue, nodeQueue queue.NodeRunQueue, chain []string) *scheduler.Scheduler {
	cfg := scheduler.DefaultConfig()
	cfg.NodeID = uniqueID("schedule")
	cfg.Redis.ReadTimeout = 200 * time.Millisecond
	cfg.Redis.ReadCount = 10
	cfg.Fallback.Interval = 200 * time.Millisecond
	cfg.Fallback.StaleThreshold = 24 * time.Hour
	cfg.Strategy.Chain = chain
	cfg.Strategy.LabelMatch.LoadBalance = true
	return scheduler.NewSchedulerWithConfig(store, schedulerQueue, nodeQueue, cfg)
}

func waitRun(t *testing.T, ctx context.Context, runID string, timeout time.Duration, pred func(*model.Run) bool) *model.Run {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		r, err := testStore.GetRun(ctx, runID)
		if err == nil && r != nil && pred(r) {
			return r
		}
		time.Sleep(50 * time.Millisecond)
	}
	r, _ := testStore.GetRun(ctx, runID)
	if r == nil {
		t.Fatalf("run not found: %s", runID)
	}
	return r
}

type observeAckSchedulerQueue struct {
	inner     queue.SchedulerQueue
	beforeAck func(ctx context.Context, messageID string)
}

func (o *observeAckSchedulerQueue) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	return o.inner.ScheduleRun(ctx, runID, taskID)
}

func (o *observeAckSchedulerQueue) CreateSchedulerConsumerGroup(ctx context.Context) error {
	return o.inner.CreateSchedulerConsumerGroup(ctx)
}

func (o *observeAckSchedulerQueue) ConsumeSchedulerRuns(ctx context.Context, consumerID string, count int64, blockTimeout time.Duration) ([]*queue.SchedulerMessage, error) {
	return o.inner.ConsumeSchedulerRuns(ctx, consumerID, count, blockTimeout)
}

func (o *observeAckSchedulerQueue) AckSchedulerRun(ctx context.Context, messageID string) error {
	if o.beforeAck != nil {
		o.beforeAck(ctx, messageID)
	}
	return o.inner.AckSchedulerRun(ctx, messageID)
}

func (o *observeAckSchedulerQueue) GetSchedulerQueueLength(ctx context.Context) (int64, error) {
	return o.inner.GetSchedulerQueueLength(ctx)
}

func (o *observeAckSchedulerQueue) GetSchedulerPendingCount(ctx context.Context) (int64, error) {
	return o.inner.GetSchedulerPendingCount(ctx)
}

type delayedAckSchedulerQueue struct {
	inner queue.SchedulerQueue
	delay time.Duration
}

func (d *delayedAckSchedulerQueue) ScheduleRun(ctx context.Context, runID, taskID string) (string, error) {
	return d.inner.ScheduleRun(ctx, runID, taskID)
}

func (d *delayedAckSchedulerQueue) CreateSchedulerConsumerGroup(ctx context.Context) error {
	return d.inner.CreateSchedulerConsumerGroup(ctx)
}

func (d *delayedAckSchedulerQueue) ConsumeSchedulerRuns(ctx context.Context, consumerID string, count int64, blockTimeout time.Duration) ([]*queue.SchedulerMessage, error) {
	return d.inner.ConsumeSchedulerRuns(ctx, consumerID, count, blockTimeout)
}

func (d *delayedAckSchedulerQueue) AckSchedulerRun(ctx context.Context, messageID string) error {
	time.Sleep(d.delay)
	return d.inner.AckSchedulerRun(ctx, messageID)
}

func (d *delayedAckSchedulerQueue) GetSchedulerQueueLength(ctx context.Context) (int64, error) {
	return d.inner.GetSchedulerQueueLength(ctx)
}

func (d *delayedAckSchedulerQueue) GetSchedulerPendingCount(ctx context.Context) (int64, error) {
	return d.inner.GetSchedulerPendingCount(ctx)
}

func resetRedis(t *testing.T, ctx context.Context) {
	if testRedis == nil {
		return
	}
	if err := testRedis.Client().FlushDB(ctx).Err(); err != nil {
		t.Fatalf("redis FlushDB failed: %v", err)
	}
}

func resetDBNodes(t *testing.T, ctx context.Context) {
	if testStore == nil {
		return
	}
	if _, err := testStore.DB().ExecContext(ctx, "UPDATE nodes SET status='offline'"); err != nil {
		t.Fatalf("reset db nodes failed: %v", err)
	}
}

func resetState(t *testing.T, ctx context.Context) {
	resetRedis(t, ctx)
	resetDBNodes(t, ctx)
}

func TestMain(m *testing.M) {
	// 强制使用测试环境
	os.Setenv("APP_ENV", "test")
	cfg := config.Load()

	var err error
	testStore, err = storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		panic("Failed to connect to PostgreSQL: " + err.Error())
	}

	// 初始化 Redis
	testRedis, err = infra.NewRedisInfra(cfg.RedisURL)
	if err != nil {
		testRedis = nil
	}

	code := m.Run()

	testStore.Close()
	if testRedis != nil {
		testRedis.Close()
	}

	os.Exit(code)
}

// TC-SCHEDULE-001: 基本调度
func TestSchedule_Basic(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	// 1. 准备：创建 Task 和 queued Run
	taskID := uniqueID("task-sched")
	runID := uniqueID("run-sched")
	labelVal := uniqueID("lbl")

	task := &model.Task{
		ID:        taskID,
		Name:      "Schedule Test Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test prompt"},
		Labels:    map[string]string{"t": labelVal},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  json.RawMessage(`{"prompt": "test prompt"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	beforeRun, err := testStore.GetRun(ctx, runID)
	if err != nil || beforeRun == nil {
		t.Fatalf("Failed to get run before scheduling: %v", err)
	}

	// 2. 准备：创建一个在线节点
	nodeID := uniqueID("node-sched")
	node := &model.Node{
		ID:            nodeID,
		Status:        model.NodeStatusOnline,
		Labels:        mustLabelsJSON(t, map[string]string{"t": labelVal}),
		Capacity:      json.RawMessage(`{"max_concurrent": 5}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := testStore.UpsertNode(ctx, node); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeID)

	// 3. 准备：创建节点的消费者组
	if err := testRedis.CreateNodeConsumerGroup(ctx, nodeID); err != nil {
		t.Logf("CreateNodeConsumerGroup: %v (may already exist)", err)
	}

	// 4. 发布任务到调度队列
	_, err = testRedis.ScheduleRun(ctx, runID, taskID)
	if err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// 5. 创建并启动 Scheduler（短暂运行）
	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	// 6. 等待调度完成
	updatedRun := waitRun(t, ctx, runID, 2500*time.Millisecond, func(r *model.Run) bool {
		return r.Status == model.RunStatusAssigned
	})

	// 7. 验证 Run 状态变为 assigned
	if updatedRun.Status != model.RunStatusAssigned {
		t.Errorf("Run status = %s, want assigned", updatedRun.Status)
	}
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeID {
		t.Errorf("Run node_id = %v, want %s", updatedRun.NodeID, nodeID)
	}
	if updatedRun.StartedAt != nil {
		t.Errorf("Run started_at = %v, want nil", updatedRun.StartedAt)
	}
	if !updatedRun.UpdatedAt.After(beforeRun.UpdatedAt) {
		t.Errorf("Run updated_at not updated: before=%v after=%v", beforeRun.UpdatedAt, updatedRun.UpdatedAt)
	}

	taskAfter, err := testStore.GetTask(ctx, taskID)
	if err != nil || taskAfter == nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if taskAfter.Status != model.TaskStatusPending {
		t.Errorf("Task status = %s, want pending", taskAfter.Status)
	}

	// 8. 验证消息已发布到节点 Stream
	messages, err := testRedis.ConsumeNodeRuns(ctx, nodeID, uniqueID("consumer"), 10, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to consume node runs: %v", err)
	}
	if len(messages) < 1 {
		t.Fatalf("Expected at least 1 message, got %d", len(messages))
	}

	// 验证消息内容
	msg := messages[0]
	if msg.RunID != runID {
		t.Errorf("Message run_id = %s, want %s", msg.RunID, runID)
	}
	if msg.TaskID != taskID {
		t.Errorf("Message task_id = %s, want %s", msg.TaskID, taskID)
	}

	t.Logf("TC-SCHEDULE-001: 基本调度成功，run_id=%s, node_id=%s", runID, nodeID)
}

// TC-SCHEDULE-002: 无可用节点（标签不匹配）
func TestSchedule_NoAvailableNode(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	// 1. 准备：创建 Task 和 queued Run
	taskID := uniqueID("task-no-node")
	runID := uniqueID("run-no-node")

	task := &model.Task{
		ID:        taskID,
		Name:      "No Node Test Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test prompt"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  json.RawMessage(`{"prompt": "test prompt"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	var buf bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prevOut)

	// 2. 不创建任何在线节点

	// 3. 发布任务到调度队列
	_, err := testRedis.ScheduleRun(ctx, runID, taskID)
	if err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// 4. 启动 Scheduler
	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 2*time.Second)
	defer stopScheduler(t, schedCancel, done)

	// 5. 等待调度周期
	time.Sleep(1500 * time.Millisecond)

	// 6. 验证 Run 状态仍为 queued
	updatedRun, err := testStore.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	if updatedRun.Status != model.RunStatusQueued {
		t.Errorf("Run status = %s, want queued (no available node)", updatedRun.Status)
	}
	if updatedRun.NodeID != nil {
		t.Errorf("Run node_id = %v, want nil", updatedRun.NodeID)
	}
	if !strings.Contains(buf.String(), "[scheduler.run.no_nodes]") {
		t.Fatalf("Expected log contains [scheduler.run.no_nodes], got=%s", buf.String())
	}

	t.Logf("TC-SCHEDULE-002: 无可用节点时，Run 保持 queued 状态")
}

// TC-SCHEDULE-009: Redis 消息确认
// 【测试目的】验证调度器在处理完消息后，必须发送 ACK 确认给 Redis
// 【背景知识】Redis Stream 是一种消息队列，支持消费者组模式：
//   - 生产者：向 Stream 发送消息
//   - 消费者组：从 Stream 读取消息
//   - ACK：消费者处理完消息后，必须确认，否则消息会保留在"待处理列表(pending list)"中
//   - 作用：防止消息丢失，确保消息至少被处理一次
//
// 【消息生命周期】
//  1. 消息进入 Stream (XADD) → 2. 消费者读取 (XREADGROUP) → 3. 消息进入 pending 列表
//  4. 业务处理 → 5. 发送 ACK (XACK) → 6. 消息从 pending 列表移除
//
// 【测试原理】
//
//	我们故意延迟 ACK 300ms，让测试有机会观察到消息处于 pending 状态
//	然后验证 ACK 后消息确实从 pending 列表消失了
func TestSchedule_MessageAck(t *testing.T) {
	// 如果没有 Redis 连接，跳过测试
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx) // 清理环境，确保测试独立

	// 生成唯一标识符
	taskID := uniqueID("task-ack")
	runID := uniqueID("run-ack")
	nodeID := uniqueID("node-ack")

	// 【步骤1】创建基础数据：Task + Run + Node
	// 这些都是调度必需的实体，跟基本调度测试一样
	task := &model.Task{
		ID:        taskID,
		Name:      "Ack Test Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued, // 关键状态：等待调度
		Snapshot:  json.RawMessage(`{"prompt": "test"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	node := &model.Node{
		ID:       nodeID,
		Status:   model.NodeStatusOnline, // 节点在线，可以被调度
		Labels:   json.RawMessage(`{}`),
		Capacity: json.RawMessage(`{"max_concurrent": 5}`),
	}
	if err := testStore.UpsertNode(ctx, node); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeID)

	// 为节点创建消费者组，这是调度器发送任务通知的队列
	if err := testRedis.CreateNodeConsumerGroup(ctx, nodeID); err != nil {
		t.Logf("CreateNodeConsumerGroup: %v (may already exist)", err)
	}

	// 【步骤2】发布任务到 Redis Stream 队列
	// ScheduleRun 会向 Redis Stream "scheduler:runs" 添加一条消息
	// 返回的 msgID 是 Redis 分配的消息唯一标识（如 "1234567890-0"）
	msgID, err := testRedis.ScheduleRun(ctx, runID, taskID)
	if err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// 【步骤3】使用"延迟 ACK"包装器创建调度器
	// 这是测试的核心技巧！
	//
	// 正常流程：调度器读取消息 → 分配节点 → 立即 ACK
	// 测试需求：我们需要在 ACK 之前，验证消息确实在 pending 列表中
	//
	// delayedAckSchedulerQueue 是一个包装器，它会：
	// - 代理所有方法给真实的 Redis
	// - 但在 AckSchedulerRun 方法中，先 sleep 300ms，再执行真正的 ACK
	//
	// 这样我们就有时间窗口去观察 "pending 列表中有这条消息"
	q := &delayedAckSchedulerQueue{inner: testRedis, delay: 300 * time.Millisecond}
	sched := newTestScheduler(testStore, q, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 4*time.Second)
	defer stopScheduler(t, schedCancel, done)

	// 【步骤4】验证消息处于 pending 状态（第一次检查）
	//
	// 当调度器启动后，它会执行 XREADGROUP 读取消息
	// 此时 Redis 会将消息标记为"已分配给消费者"，并放入 pending 列表
	// 但业务处理可能还没完成，ACK 还没发送
	//
	// XPendingExt 命令：查询指定消费者组的 pending 列表
	// - Stream: "scheduler:runs" (队列名称)
	// - Group: "scheduler-consumer-group" (消费者组名称)
	// - Start/End: 只查询我们发送的那条消息
	// - Count: 1 (只查一条)
	//
	// 如果 pending 列表中有这条消息，说明：
	// 1. 调度器成功读取了消息 ✓
	// 2. 但还没 ACK（因为我们故意延迟了）✓
	// 3. 这正是我们想验证的状态 ✓
	deadline := time.Now().Add(2 * time.Second)
	seenPending := false
	for time.Now().Before(deadline) {
		pending, err := testRedis.Client().XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: queue.KeySchedulerRuns,       // Stream 名称：调度队列
			Group:  queue.SchedulerConsumerGroup, // 消费者组：调度器消费者组
			Start:  msgID,                        // 查询起始消息ID
			End:    msgID,                        // 查询结束消息ID（只查这一条）
			Count:  1,                            // 最多返回1条
		}).Result()
		// 如果查询成功且返回1条，说明消息在 pending 列表中
		if err == nil && len(pending) == 1 {
			seenPending = true
			break // 验证成功，退出循环
		}
		time.Sleep(20 * time.Millisecond) // 没查到就等20ms再试
	}
	if !seenPending {
		t.Fatalf("Expected scheduler pending contains msg_id=%s", msgID)
	}
	// 【结论】此时消息确实处于 pending 状态，说明调度器读取了但还没 ACK

	// 【步骤5】等待调度完成
	// waitRun 会轮询数据库，直到 Run 状态变为 assigned
	// 这说明调度器已经完成了：分配节点 + 更新数据库 + 发送节点通知
	_ = waitRun(t, ctx, runID, 3*time.Second, func(r *model.Run) bool {
		return r.Status == model.RunStatusAssigned
	})

	// 【步骤6】验证消息已从 pending 列表移除（第二次检查）
	//
	// 经过 300ms 的延迟后，delayedAckSchedulerQueue 会执行真正的 ACK
	// ACK 成功后，消息应该从 pending 列表中消失
	//
	// 如果 pending 列表为空，说明 ACK 成功
	// 如果 pending 列表还有消息，说明 ACK 失败（测试失败）
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pending, err := testRedis.Client().XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: queue.KeySchedulerRuns,
			Group:  queue.SchedulerConsumerGroup,
			Start:  msgID,
			End:    msgID,
			Count:  1,
		}).Result()
		// 如果查询成功且返回0条，说明消息已从 pending 列表移除（ACK成功）
		if err == nil && len(pending) == 0 {
			return // 测试通过！ACK 机制工作正常
		}
		time.Sleep(20 * time.Millisecond)
	}
	// 如果走到这里，说明2秒后消息还在 pending 列表中，ACK 可能失败了
	t.Fatalf("Expected scheduler pending removed for msg_id=%s", msgID)
}

// TC-SCHEDULE-003: 标签匹配调度
func TestSchedule_LabelMatching(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	// 1. 准备：创建带标签的 Task
	taskID := uniqueID("task-label")
	runID := uniqueID("run-label")
	labelVal := uniqueID("lbl")

	task := &model.Task{
		ID:        taskID,
		Name:      "Label Test Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test"},
		Labels:    map[string]string{"env": "prod", "t": labelVal},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  json.RawMessage(`{"prompt": "test prompt"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	// 2. 创建匹配标签的节点
	matchNodeID := uniqueID("node-label")
	matchNode := &model.Node{
		ID:            matchNodeID,
		Status:        model.NodeStatusOnline,
		Labels:        mustLabelsJSON(t, map[string]string{"env": "prod", "gpu": "true", "t": labelVal}),
		Capacity:      json.RawMessage(`{"max_concurrent": 5}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := testStore.UpsertNode(ctx, matchNode); err != nil {
		t.Fatalf("Failed to create matching node: %v", err)
	}
	defer testStore.DeleteNode(ctx, matchNodeID)

	// 3. 创建不匹配标签的节点
	unmatchNodeID := uniqueID("node-label")
	unmatchNode := &model.Node{
		ID:            unmatchNodeID,
		Status:        model.NodeStatusOnline,
		Labels:        mustLabelsJSON(t, map[string]string{"env": "staging", "t": labelVal}),
		Capacity:      json.RawMessage(`{"max_concurrent": 5}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := testStore.UpsertNode(ctx, unmatchNode); err != nil {
		t.Fatalf("Failed to create unmatching node: %v", err)
	}
	defer testStore.DeleteNode(ctx, unmatchNodeID)

	// 4. 创建消费者组
	if err := testRedis.CreateNodeConsumerGroup(ctx, matchNodeID); err != nil {
		t.Logf("CreateNodeConsumerGroup: %v (may already exist)", err)
	}

	// 5. 发布任务到调度队列
	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// 6. 启动 Scheduler
	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 2*time.Second)
	defer stopScheduler(t, schedCancel, done)
	time.Sleep(1500 * time.Millisecond)

	// 7. 验证分配到匹配标签的节点
	updatedRun, err := testStore.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	if updatedRun.Status != model.RunStatusAssigned {
		t.Errorf("Run status = %s, want assigned", updatedRun.Status)
	}
	if updatedRun.NodeID == nil || *updatedRun.NodeID != matchNodeID {
		nodeIDStr := "<nil>"
		if updatedRun.NodeID != nil {
			nodeIDStr = *updatedRun.NodeID
		}
		t.Errorf("Run node_id = %s, want %s (label matching)", nodeIDStr, matchNodeID)
	}

	t.Logf("TC-SCHEDULE-003: 任务成功分配到标签匹配的节点 %s", matchNodeID)
}

// TC-SCHEDULE-005: 负载均衡
func TestSchedule_LoadBalancing(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	// 1. 准备：创建 Task 和 Run
	taskID := uniqueID("task-lb")
	runID := uniqueID("run-lb")
	labelVal := uniqueID("lbl")

	task := &model.Task{
		ID:        taskID,
		Name:      "Load Balance Test Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test"},
		Labels:    map[string]string{"t": labelVal},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  json.RawMessage(`{"prompt": "test"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	// 2. 创建两个节点（相同 max_concurrent），通过已有 assigned Run 制造负载差异
	nodeAID := uniqueID("node-lb")
	nodeA := &model.Node{
		ID:            nodeAID,
		Status:        model.NodeStatusOnline,
		Labels:        mustLabelsJSON(t, map[string]string{"t": labelVal}),
		Capacity:      json.RawMessage(`{"max_concurrent": 5}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := testStore.UpsertNode(ctx, nodeA); err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeAID)

	nodeBID := uniqueID("node-lb")
	nodeB := &model.Node{
		ID:            nodeBID,
		Status:        model.NodeStatusOnline,
		Labels:        mustLabelsJSON(t, map[string]string{"t": labelVal}),
		Capacity:      json.RawMessage(`{"max_concurrent": 5}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := testStore.UpsertNode(ctx, nodeB); err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeBID)

	// 3. 创建消费者组
	if err := testRedis.CreateNodeConsumerGroup(ctx, nodeAID); err != nil {
		t.Logf("CreateNodeConsumerGroup: %v (may already exist)", err)
	}
	if err := testRedis.CreateNodeConsumerGroup(ctx, nodeBID); err != nil {
		t.Logf("CreateNodeConsumerGroup: %v (may already exist)", err)
	}

	for i := 0; i < 4; i++ {
		dummyID := uniqueID("run-lb-a")
		dummyNode := nodeAID
		dummy := &model.Run{
			ID:        dummyID,
			TaskID:    taskID,
			Status:    model.RunStatusAssigned,
			NodeID:    &dummyNode,
			Snapshot:  json.RawMessage(`{"prompt": "dummy"}`),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := testStore.CreateRun(ctx, dummy); err != nil {
			t.Fatalf("Failed to create dummy run A: %v", err)
		}
		defer testStore.DeleteRun(ctx, dummyID)
	}
	for i := 0; i < 1; i++ {
		dummyID := uniqueID("run-lb-b")
		dummyNode := nodeBID
		dummy := &model.Run{
			ID:        dummyID,
			TaskID:    taskID,
			Status:    model.RunStatusAssigned,
			NodeID:    &dummyNode,
			Snapshot:  json.RawMessage(`{"prompt": "dummy"}`),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := testStore.CreateRun(ctx, dummy); err != nil {
			t.Fatalf("Failed to create dummy run B: %v", err)
		}
		defer testStore.DeleteRun(ctx, dummyID)
	}

	// 4. 发布任务到调度队列
	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// 5. 启动 Scheduler
	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 2*time.Second)
	defer stopScheduler(t, schedCancel, done)
	time.Sleep(1500 * time.Millisecond)

	// 6. 验证分配到容量更大的节点 B
	updatedRun, err := testStore.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	if updatedRun.Status != model.RunStatusAssigned {
		t.Errorf("Run status = %s, want assigned", updatedRun.Status)
	}
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeBID {
		nodeIDStr := "<nil>"
		if updatedRun.NodeID != nil {
			nodeIDStr = *updatedRun.NodeID
		}
		t.Errorf("Run node_id = %s, want %s (load balancing)", nodeIDStr, nodeBID)
	}

	t.Logf("TC-SCHEDULE-005: 任务成功分配到容量更大的节点 %s", nodeBID)
}

// TC-SCHEDULE-008: 幂等性
func TestSchedule_Idempotent(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}
	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	// 1. 准备：创建 Task、在线 Node、以及已分配的 Run
	taskID := uniqueID("task-idem")
	runID := uniqueID("run-idem")
	nodeID := uniqueID("node-idem")

	task := &model.Task{
		ID:        taskID,
		Name:      "Idempotent Test Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	// Run 已经是 assigned 状态
	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusAssigned,
		NodeID:    &nodeID,
		Snapshot:  json.RawMessage(`{"prompt": "test"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	lenBefore, _ := testRedis.GetNodeRunsQueueLength(ctx, nodeID)
	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 2*time.Second)
	defer stopScheduler(t, schedCancel, done)

	time.Sleep(1200 * time.Millisecond)

	// 3. 验证 Run 状态不变
	updatedRun, err := testStore.GetRun(ctx, runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	if updatedRun.Status != model.RunStatusAssigned {
		t.Errorf("Run status = %s, want assigned", updatedRun.Status)
	}
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeID {
		t.Errorf("Run node_id changed, expected %s", nodeID)
	}
	lenAfter, _ := testRedis.GetNodeRunsQueueLength(ctx, nodeID)
	if lenAfter != lenBefore {
		t.Errorf("Node stream length changed: %d -> %d", lenBefore, lenAfter)
	}

	t.Logf("TC-SCHEDULE-008: 幂等性验证通过，已分配的 Run 不会被重复调度")
}

func TestSchedule_TC_SCHEDULE_004_LabelMismatch(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	taskID := uniqueID("task-label-mismatch")
	runID := uniqueID("run-label-mismatch")
	nodeID := uniqueID("node-label-mismatch")

	task := &model.Task{
		ID:        taskID,
		Name:      "Label Mismatch Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test"},
		Labels:    map[string]string{"env": "prod"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	run := &model.Run{
		ID:        runID,
		TaskID:    taskID,
		Status:    model.RunStatusQueued,
		Snapshot:  json.RawMessage(`{"prompt": "test"}`),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	node := &model.Node{
		ID:            nodeID,
		Status:        model.NodeStatusOnline,
		Labels:        json.RawMessage(`{"env": "staging"}`),
		Capacity:      json.RawMessage(`{"max_concurrent": 5}`),
		LastHeartbeat: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := testStore.UpsertNode(ctx, node); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeID)
	if err := testRedis.CreateNodeConsumerGroup(ctx, nodeID); err != nil {
		t.Logf("CreateNodeConsumerGroup: %v (may already exist)", err)
	}

	var buf bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prevOut)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 2*time.Second)
	defer stopScheduler(t, schedCancel, done)

	time.Sleep(1200 * time.Millisecond)

	updatedRun, err := testStore.GetRun(ctx, runID)
	if err != nil || updatedRun == nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	if updatedRun.Status != model.RunStatusQueued {
		t.Fatalf("Run status = %s, want queued", updatedRun.Status)
	}
	if updatedRun.NodeID != nil {
		t.Fatalf("Run node_id = %v, want nil", updatedRun.NodeID)
	}
	if !strings.Contains(buf.String(), "[scheduler.run.no_match]") {
		t.Fatalf("Expected log contains [scheduler.run.no_match], got=%s", buf.String())
	}
}

func TestSchedule_TC_SCHEDULE_006_FixedNodeScheduling(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	nodeAID := uniqueID("node-fixed-a")
	nodeBID := uniqueID("node-fixed-b")

	nodeA := &model.Node{ID: nodeAID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	nodeB := &model.Node{ID: nodeBID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, nodeA); err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeAID)
	if err := testStore.UpsertNode(ctx, nodeB); err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeBID)

	instID := uniqueID("inst-001")
	instNode := nodeAID
	inst := &model.Instance{
		ID:          instID,
		Name:        "inst",
		AccountID:   uniqueID("acc"),
		AgentTypeID: "test",
		NodeID:      &instNode,
		Status:      model.InstanceStatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := testStore.CreateAgentInstance(ctx, inst); err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer testStore.DeleteAgentInstance(ctx, instID)

	taskID := uniqueID("task-fixed")
	agentID := instID
	task := &model.Task{
		ID:        taskID,
		Name:      "Fixed Node Task",
		Status:    model.TaskStatusPending,
		Type:      model.TaskTypeGeneral,
		Prompt:    &model.Prompt{Content: "test"},
		AgentID:   &agentID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	runID := uniqueID("run-fixed")
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(`{"prompt":"test"}`), CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	testRedis.CreateNodeConsumerGroup(ctx, nodeAID)
	testRedis.CreateNodeConsumerGroup(ctx, nodeBID)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	updatedRun := waitRun(t, ctx, runID, 2500*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeAID {
		n := "<nil>"
		if updatedRun.NodeID != nil {
			n = *updatedRun.NodeID
		}
		t.Fatalf("Run node_id = %s, want %s", n, nodeAID)
	}

	messages, err := testRedis.ConsumeNodeRuns(ctx, nodeAID, uniqueID("consumer"), 10, 1*time.Second)
	if err != nil {
		t.Fatalf("Failed to consume node runs: %v", err)
	}
	if len(messages) < 1 {
		t.Fatalf("Expected at least 1 node message, got %d", len(messages))
	}
	if messages[0].RunID != runID || messages[0].TaskID != taskID {
		t.Fatalf("Unexpected node message: run_id=%s task_id=%s", messages[0].RunID, messages[0].TaskID)
	}
}

func TestSchedule_TC_SCHEDULE_007_FallbackPolling(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetState(t, ctx)

	taskID := uniqueID("task-fallback")
	runID := uniqueID("run-fallback")
	nodeID := uniqueID("node-fallback")

	task := &model.Task{ID: taskID, Name: "Fallback Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	createdAt := time.Now().Add(-10 * time.Minute)
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(`{"prompt":"test"}`), CreatedAt: createdAt, UpdatedAt: createdAt}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	node := &model.Node{ID: nodeID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, node); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeID)
	testRedis.CreateNodeConsumerGroup(ctx, nodeID)

	var buf bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prevOut)

	sched := newTestScheduler(testStore, nil, testRedis)
	sched.SetFallbackConfig(120*time.Millisecond, 5*time.Minute)
	schedCancel, done := startScheduler(ctx, sched, 1500*time.Millisecond)
	defer stopScheduler(t, schedCancel, done)

	updatedRun := waitRun(t, ctx, runID, 1200*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	if updatedRun.Status != model.RunStatusAssigned {
		t.Fatalf("Run status = %s, want assigned", updatedRun.Status)
	}
	if !strings.Contains(buf.String(), "[scheduler.fallback.found]") {
		t.Fatalf("Expected log contains [scheduler.fallback.found], got=%s", buf.String())
	}
}

// TC-SCHEDULE-010: 直接指定节点
func TestSchedule_TC_SCHEDULE_010_DirectNodeSpecification(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetRedis(t, ctx)

	nodeAID := uniqueID("node-dir-a")
	nodeBID := uniqueID("node-dir-b")

	nodeA := &model.Node{ID: nodeAID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	nodeB := &model.Node{ID: nodeBID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, nodeA); err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeAID)
	if err := testStore.UpsertNode(ctx, nodeB); err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeBID)

	taskID := uniqueID("task-direct")
	task := &model.Task{ID: taskID, Name: "Direct Node Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	// Snapshot 中指定 node_id 为 nodeA
	runID := uniqueID("run-direct")
	snapshot := fmt.Sprintf(`{"prompt":"test","node_id":"%s"}`, nodeAID)
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(snapshot), CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	testRedis.CreateNodeConsumerGroup(ctx, nodeAID)
	testRedis.CreateNodeConsumerGroup(ctx, nodeBID)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// 使用包含 direct 策略的策略链
	sched := newTestSchedulerWithChain(testStore, testRedis, testRedis, []string{"direct", "affinity", "label_match"})
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	updatedRun := waitRun(t, ctx, runID, 2500*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeAID {
		n := "<nil>"
		if updatedRun.NodeID != nil {
			n = *updatedRun.NodeID
		}
		t.Fatalf("Run node_id = %s, want %s (direct)", n, nodeAID)
	}
	t.Logf("TC-SCHEDULE-010: 直接指定节点调度成功，node_id=%s", nodeAID)
}

// TC-SCHEDULE-011: 直接指定节点不可用时回退
func TestSchedule_TC_SCHEDULE_011_DirectFallback(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetRedis(t, ctx)

	labelVal := uniqueID("lbl")
	// 只有 nodeB 在线
	nodeBID := uniqueID("node-dfb-b")
	nodeB := &model.Node{ID: nodeBID, Status: model.NodeStatusOnline, Labels: mustLabelsJSON(t, map[string]string{"t": labelVal}), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, nodeB); err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeBID)

	taskID := uniqueID("task-dfb")
	task := &model.Task{ID: taskID, Name: "Direct Fallback Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, Labels: map[string]string{"t": labelVal}, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	// Snapshot 中指定一个不存在的节点
	runID := uniqueID("run-dfb")
	snapshot := `{"prompt":"test","node_id":"node-does-not-exist"}`
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(snapshot), CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	testRedis.CreateNodeConsumerGroup(ctx, nodeBID)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// direct 失败后回退到 label_match
	sched := newTestSchedulerWithChain(testStore, testRedis, testRedis, []string{"direct", "label_match"})
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	updatedRun := waitRun(t, ctx, runID, 2500*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeBID {
		n := "<nil>"
		if updatedRun.NodeID != nil {
			n = *updatedRun.NodeID
		}
		t.Fatalf("Run node_id = %s, want %s (fallback to label_match)", n, nodeBID)
	}
	t.Logf("TC-SCHEDULE-011: direct 回退到 label_match 成功，node_id=%s", nodeBID)
}

// TC-SCHEDULE-012: 轮询策略
func TestSchedule_TC_SCHEDULE_012_RoundRobin(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetRedis(t, ctx)

	// 创建 3 个节点
	nodeIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		nodeIDs[i] = uniqueID(fmt.Sprintf("node-rr-%d", i))
		n := &model.Node{ID: nodeIDs[i], Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
		if err := testStore.UpsertNode(ctx, n); err != nil {
			t.Fatalf("Failed to create node %d: %v", i, err)
		}
		defer testStore.DeleteNode(ctx, nodeIDs[i])
		testRedis.CreateNodeConsumerGroup(ctx, nodeIDs[i])
	}

	taskID := uniqueID("task-rr")
	task := &model.Task{ID: taskID, Name: "RoundRobin Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	// 创建 3 个 Run
	runIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		runIDs[i] = uniqueID(fmt.Sprintf("run-rr-%d", i))
		r := &model.Run{ID: runIDs[i], TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(`{"prompt":"test"}`), CreatedAt: now, UpdatedAt: now}
		if err := testStore.CreateRun(ctx, r); err != nil {
			t.Fatalf("Failed to create run %d: %v", i, err)
		}
		defer testStore.DeleteRun(ctx, runIDs[i])
		if _, err := testRedis.ScheduleRun(ctx, runIDs[i], taskID); err != nil {
			t.Fatalf("Failed to schedule run %d: %v", i, err)
		}
	}

	// 使用 round_robin 策略
	sched := newTestSchedulerWithChain(testStore, testRedis, testRedis, []string{"round_robin"})
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	// 等待所有 Run 被调度
	for i := 0; i < 3; i++ {
		waitRun(t, ctx, runIDs[i], 2500*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	}

	// 验证至少分配到 2 个不同的节点
	nodeSet := make(map[string]bool)
	for _, rid := range runIDs {
		r, err := testStore.GetRun(ctx, rid)
		if err != nil || r == nil {
			t.Fatalf("Failed to get run %s: %v", rid, err)
		}
		if r.NodeID == nil {
			t.Fatalf("Run %s not assigned", rid)
		}
		nodeSet[*r.NodeID] = true
	}
	if len(nodeSet) < 2 {
		t.Fatalf("round_robin: expected runs distributed across >=2 nodes, got %d unique node(s)", len(nodeSet))
	}
	t.Logf("TC-SCHEDULE-012: 轮询策略分配到 %d 个不同节点", len(nodeSet))
}

// TC-SCHEDULE-013: 随机策略
func TestSchedule_TC_SCHEDULE_013_Random(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetRedis(t, ctx)

	nodeAID := uniqueID("node-rand-a")
	nodeBID := uniqueID("node-rand-b")
	nodeA := &model.Node{ID: nodeAID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	nodeB := &model.Node{ID: nodeBID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, nodeA); err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeAID)
	if err := testStore.UpsertNode(ctx, nodeB); err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeBID)

	taskID := uniqueID("task-rand")
	task := &model.Task{ID: taskID, Name: "Random Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	runID := uniqueID("run-rand")
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(`{"prompt":"test"}`), CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	testRedis.CreateNodeConsumerGroup(ctx, nodeAID)
	testRedis.CreateNodeConsumerGroup(ctx, nodeBID)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	sched := newTestSchedulerWithChain(testStore, testRedis, testRedis, []string{"random"})
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	updatedRun := waitRun(t, ctx, runID, 2500*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	if updatedRun.NodeID == nil {
		t.Fatalf("Run not assigned")
	}
	assignedNode := *updatedRun.NodeID
	if assignedNode != nodeAID && assignedNode != nodeBID {
		t.Fatalf("Run assigned to unexpected node %s", assignedNode)
	}
	t.Logf("TC-SCHEDULE-013: 随机策略成功分配到节点 %s", assignedNode)
}

// TC-SCHEDULE-014: 所有节点容量已满
func TestSchedule_TC_SCHEDULE_014_CapacityFull(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetRedis(t, ctx)

	nodeID := uniqueID("node-capfull")
	node := &model.Node{ID: nodeID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 2}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, node); err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeID)

	taskID := uniqueID("task-capfull")
	task := &model.Task{ID: taskID, Name: "CapFull Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	// 创建 2 个已分配 Run 占满容量
	for i := 0; i < 2; i++ {
		dummyID := uniqueID("run-capdum")
		dummyNode := nodeID
		dummy := &model.Run{ID: dummyID, TaskID: taskID, Status: model.RunStatusAssigned, NodeID: &dummyNode, Snapshot: json.RawMessage(`{"prompt":"dummy"}`), CreatedAt: now, UpdatedAt: now}
		if err := testStore.CreateRun(ctx, dummy); err != nil {
			t.Fatalf("Failed to create dummy run: %v", err)
		}
		defer testStore.DeleteRun(ctx, dummyID)
	}

	runID := uniqueID("run-capfull")
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(`{"prompt":"test"}`), CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	testRedis.CreateNodeConsumerGroup(ctx, nodeID)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	sched := newTestScheduler(testStore, testRedis, testRedis)
	schedCancel, done := startScheduler(ctx, sched, 2*time.Second)
	defer stopScheduler(t, schedCancel, done)

	time.Sleep(1500 * time.Millisecond)

	updatedRun, err := testStore.GetRun(ctx, runID)
	if err != nil || updatedRun == nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	if updatedRun.Status != model.RunStatusQueued {
		t.Fatalf("Run status = %s, want queued (capacity full)", updatedRun.Status)
	}
	t.Logf("TC-SCHEDULE-014: 容量已满时 Run 保持 queued 状态")
}

// TC-SCHEDULE-015: 策略链优先级 - direct 优先于 affinity
func TestSchedule_TC_SCHEDULE_015_ChainPriority(t *testing.T) {
	if testRedis == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	now := time.Now()
	resetRedis(t, ctx)

	nodeAID := uniqueID("node-pri-a")
	nodeBID := uniqueID("node-pri-b")

	nodeA := &model.Node{ID: nodeAID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	nodeB := &model.Node{ID: nodeBID, Status: model.NodeStatusOnline, Labels: json.RawMessage(`{}`), Capacity: json.RawMessage(`{"max_concurrent": 5}`), LastHeartbeat: &now, CreatedAt: now, UpdatedAt: now}
	if err := testStore.UpsertNode(ctx, nodeA); err != nil {
		t.Fatalf("Failed to create node A: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeAID)
	if err := testStore.UpsertNode(ctx, nodeB); err != nil {
		t.Fatalf("Failed to create node B: %v", err)
	}
	defer testStore.DeleteNode(ctx, nodeBID)

	// Instance 绑定在 Node A（亲和性指向 A）
	instID := uniqueID("inst-pri")
	instNode := nodeAID
	inst := &model.Instance{
		ID:          instID,
		Name:        "priority-inst",
		AccountID:   uniqueID("acc"),
		AgentTypeID: "test",
		NodeID:      &instNode,
		Status:      model.InstanceStatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := testStore.CreateAgentInstance(ctx, inst); err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}
	defer testStore.DeleteAgentInstance(ctx, instID)

	taskID := uniqueID("task-pri")
	agentID := instID
	task := &model.Task{ID: taskID, Name: "Priority Task", Status: model.TaskStatusPending, Type: model.TaskTypeGeneral, Prompt: &model.Prompt{Content: "test"}, AgentID: &agentID, CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateTask(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer testStore.DeleteTask(ctx, taskID)

	// Snapshot 直接指定 Node B（direct 策略指向 B）
	runID := uniqueID("run-pri")
	snapshot := fmt.Sprintf(`{"prompt":"test","node_id":"%s"}`, nodeBID)
	run := &model.Run{ID: runID, TaskID: taskID, Status: model.RunStatusQueued, Snapshot: json.RawMessage(snapshot), CreatedAt: now, UpdatedAt: now}
	if err := testStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}
	defer testStore.DeleteRun(ctx, runID)

	testRedis.CreateNodeConsumerGroup(ctx, nodeAID)
	testRedis.CreateNodeConsumerGroup(ctx, nodeBID)

	if _, err := testRedis.ScheduleRun(ctx, runID, taskID); err != nil {
		t.Fatalf("Failed to schedule run: %v", err)
	}

	// direct > affinity > label_match
	sched := newTestSchedulerWithChain(testStore, testRedis, testRedis, []string{"direct", "affinity", "label_match"})
	schedCancel, done := startScheduler(ctx, sched, 3*time.Second)
	defer stopScheduler(t, schedCancel, done)

	updatedRun := waitRun(t, ctx, runID, 2500*time.Millisecond, func(r *model.Run) bool { return r.Status == model.RunStatusAssigned })
	if updatedRun.NodeID == nil || *updatedRun.NodeID != nodeBID {
		n := "<nil>"
		if updatedRun.NodeID != nil {
			n = *updatedRun.NodeID
		}
		t.Fatalf("Run node_id = %s, want %s (direct > affinity)", n, nodeBID)
	}
	t.Logf("TC-SCHEDULE-015: direct 优先于 affinity，node_id=%s", nodeBID)
}
