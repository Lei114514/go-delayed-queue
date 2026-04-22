package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"go-delay-queue/internal/handler"
	"go-delay-queue/internal/logger"
	"go-delay-queue/internal/metrics"
	"go-delay-queue/internal/retry"
	"go-delay-queue/internal/storage"
	"go-delay-queue/pkg/task"
)

// Pool Worker 池
type Pool struct {
	workerCount   int
	taskChan      chan *task.Task
	registry      *handler.Registry
	storage       storage.Storage
	collector     *metrics.Collector
	retryStrategy retry.Strategy
	wg            sync.WaitGroup
}

// NewPool 创建 Worker Pool
func NewPool(workerCount int, registry *handler.Registry, storage storage.Storage, collector *metrics.Collector) *Pool {
	return &Pool{
		workerCount:   workerCount,
		taskChan:      make(chan *task.Task, 100),
		registry:      registry,
		storage:       storage,
		collector:     collector,
		retryStrategy: retry.NewExponentialBackoff(),
	}
}

// Start 启动 Worker Pool
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Stop 停止 Worker Pool（优雅关闭）
func (p *Pool) Stop() {
	close(p.taskChan)
	p.wg.Wait()
}

// Submit 提交任务
func (p *Pool) Submit(t *task.Task) bool {
	select {
	case p.taskChan <- t:
		return true
	default:
		return false
	}
}

// worker 单个 worker 协程
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping", zap.Int("worker_id", id))
			return

		case t, ok := <-p.taskChan:
			if !ok {
				logger.Info("Task channel closed, worker exiting", zap.Int("worker_id", id))
				return
			}

			p.processTask(ctx, t, id)
		}
	}
}

// processTask 处理单个任务（支持重试）
func (p *Pool) processTask(ctx context.Context, t *task.Task, workerID int) {
	log := logger.With(
		zap.String("task_id", t.TaskID),
		zap.String("type", t.TaskType),
		zap.Int("worker_id", workerID),
	)

	log.Info("Processing task")
	p.collector.RecordProcessing(1)
	p.collector.RecordPending(-1)

	// 查找处理器
	h, ok := p.registry.Get(t.TaskType)
	if !ok {
		log.Error("No handler for task type", zap.String("task_type", t.TaskType))
		p.collector.RecordFail()
		return
	}

	// 执行处理
	if err := h.Handle(t); err != nil {
		log.Error("Task failed", zap.Error(err))
		p.collector.RecordFail()
		p.handleRetry(ctx, t, workerID)
	} else {
		log.Info("Task completed")
		p.collector.RecordComplete()
	}
}

// handleRetry 处理重试逻辑
func (p *Pool) handleRetry(ctx context.Context, t *task.Task, workerID int) {
	// 检查是否还可以重试
	if !t.CanRetry() {
		logger.Info("Task exceeded max retry, giving up",
			zap.String("task_id", t.TaskID),
			zap.Int("max_retry", t.MaxRetry),
		)
		return
	}

	// 计算下次执行时间
	delay := p.retryStrategy.NextDelay(t.RetryCount + 1)
	nextExecuteAt := time.Now().Add(delay).Unix()

	// 更新任务状态
	t.MarkRetry(nextExecuteAt)
	p.collector.RecordRetry()

	logger.Info("Retrying task",
		zap.String("task_id", t.TaskID),
		zap.Duration("delay", delay),
		zap.Int64("next_execute_at", nextExecuteAt),
	)

	// 使用 Update 更新 Redis 中的任务
	if err := p.storage.Update(t); err != nil {
		logger.Error("Failed to retry task", err, zap.String("task_id", t.TaskID))
	}
}
