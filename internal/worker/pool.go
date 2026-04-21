package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-delay-queue/internal/handler"
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
	retryStrategy retry.Strategy
	wg            sync.WaitGroup
}

// NewPool 创建 Worker Pool
func NewPool(workerCount int, registry *handler.Registry, storage storage.Storage) *Pool {
	return &Pool{
		workerCount:   workerCount,
		taskChan:      make(chan *task.Task, 100),
		registry:      registry,
		storage:       storage,
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
			fmt.Printf("[Worker %d] Stopping...\n", id)
			return

		case t, ok := <-p.taskChan:
			if !ok {
				fmt.Printf("[Worker %d] Task channel closed\n", id)
				return
			}

			p.processTask(ctx, t, id)
		}
	}
}

// processTask 处理单个任务（支持重试）
func (p *Pool) processTask(ctx context.Context, t *task.Task, workerID int) {
	fmt.Printf("[Worker %d] Processing task: %s (type: %s, retry: %d/%d)\n",
		workerID, t.TaskID, t.TaskType, t.RetryCount, t.MaxRetry)

	// 查找处理器
	h, ok := p.registry.Get(t.TaskType)
	if !ok {
		fmt.Printf("[Worker %d] No handler for task type: %s\n", workerID, t.TaskType)
		return
	}

	// 执行处理
	if err := h.Handle(t); err != nil {
		fmt.Printf("[Worker %d] Task failed: %s, error: %v\n", workerID, t.TaskID, err)

		// 尝试重试
		p.handleRetry(ctx, t, workerID)
	} else {
		fmt.Printf("[Worker %d] Task completed: %s\n", workerID, t.TaskID)
	}
}

// handleRetry 处理重试逻辑
func (p *Pool) handleRetry(ctx context.Context, t *task.Task, workerID int) {
	// 检查是否还可以重试
	if !t.CanRetry() {
		fmt.Printf("[Worker %d] Task %s exceeded max retry (%d), giving up\n",
			workerID, t.TaskID, t.MaxRetry)
		return
	}

	// 计算下次执行时间
	delay := p.retryStrategy.NextDelay(t.RetryCount + 1)
	nextExecuteAt := time.Now().Add(delay).Unix()

	// 更新任务状态
	t.MarkRetry(nextExecuteAt)

	fmt.Printf("[Worker %d] Retrying task %s after %v (execute_at: %d)\n",
		workerID, t.TaskID, delay, nextExecuteAt)

	// 使用 Update 更新 Redis 中的任务
	if err := p.storage.Update(t); err != nil {
		fmt.Printf("[Worker %d] Failed to retry task %s: %v\n", workerID, t.TaskID, err)
	}
}
