package scheduler

import (
	"context"
	"fmt"
	"time"

	"go-delay-queue/internal/storage"
	"go-delay-queue/internal/worker"
)

// Scanner 任务扫描器
type Scanner struct {
	storage   storage.Storage
	workerPool *worker.Pool
	interval  time.Duration // 扫描间隔
}

// NewScanner 创建扫描器
func NewScanner(storage storage.Storage, workerPool *worker.Pool, interval time.Duration) *Scanner {
	return &Scanner{
		storage:    storage,
		workerPool: workerPool,
		interval:   interval,
	}
}

// Start 启动扫描器
func (s *Scanner) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	fmt.Printf("[Scanner] Started, scanning every %v\n", s.interval)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[Scanner] Stopping...")
			return

		case <-ticker.C:
			s.scan()
		}
	}
}

// scan 执行一次扫描
func (s *Scanner) scan() {
	// 从 Redis 获取到期任务
	// 注意：这里假设 storage 有 PopDueTask 方法
	// 需要在 storage 接口中添加
	tasks, err := s.storage.GetDueTasks()
	if err != nil {
		fmt.Printf("[Scanner] Failed to get due tasks: %v\n", err)
		return
	}

	if len(tasks) == 0 {
		return // 没有到期任务
	}

	fmt.Printf("[Scanner] Found %d due tasks\n", len(tasks))

	// 提交到 Worker Pool
	for _, task := range tasks {
		if s.workerPool.Submit(task) {
			// 从 Redis 删除任务（原子操作已在 PopDueTask 中完成）
			// 这里使用 ZRem 删除
			if redisStorage, ok := s.storage.(interface{ Remove(taskID string) error }); ok {
				redisStorage.Remove(task.TaskID)
			}
		} else {
			fmt.Printf("[Scanner] Worker pool full, task %s will be processed next round\n", task.TaskID)
		}
	}
}