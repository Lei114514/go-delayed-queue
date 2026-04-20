package worker

import (
	"context"
	"fmt"
	"sync"

	"go-delay-queue/internal/handler"
	"go-delay-queue/pkg/task"
)

// Pool Worker 池
type Pool struct {
	workerCount int
	taskChan    chan *task.Task
	registry    *handler.Registry
	wg          sync.WaitGroup
}

// NewPool 创建 Worker Pool
func NewPool(workerCount int, registry *handler.Registry) *Pool {
	return &Pool{
		workerCount: workerCount,
		taskChan:    make(chan *task.Task, 100), // 缓冲通道
		registry:    registry,
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
	close(p.taskChan) // 关闭通道，通知 workers 停止
	p.wg.Wait()       // 等待所有 worker 完成
}

// Submit 提交任务
func (p *Pool) Submit(t *task.Task) bool {
	select {
	case p.taskChan <- t:
		return true
	default:
		// 通道已满
		return false
	}
}

// worker 单个 worker 协程
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// 收到停止信号
			fmt.Printf("[Worker %d] Stopping...\n", id)
			return

		case t, ok := <-p.taskChan:
			if !ok {
				// 通道已关闭
				fmt.Printf("[Worker %d] Task channel closed, exiting\n", id)
				return
			}

			// 执行任务
			p.processTask(t, id)
		}
	}
}

// processTask 处理单个任务
func (p *Pool) processTask(t *task.Task, workerID int) {
	fmt.Printf("[Worker %d] Processing task: %s (type: %s)\n", workerID, t.TaskID, t.TaskType)

	// 查找处理器
	h, ok := p.registry.Get(t.TaskType)
	if !ok {
		fmt.Printf("[Worker %d] No handler for task type: %s\n", workerID, t.TaskType)
		return
	}

	// 执行处理
	if err := h.Handle(t); err != nil {
		fmt.Printf("[Worker %d] Task failed: %s, error: %v\n", workerID, t.TaskID, err)
		// TODO: 重试逻辑 (Day 5 实现)
	} else {
		fmt.Printf("[Worker %d] Task completed: %s\n", workerID, t.TaskID)
	}
}