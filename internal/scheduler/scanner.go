package scheduler

import (
	"context"
	"time"

	"go.uber.org/zap"

	"go-delay-queue/internal/logger"
	"go-delay-queue/internal/metrics"
	"go-delay-queue/internal/storage"
	"go-delay-queue/internal/worker"
)

// Scanner 任务扫描器
type Scanner struct {
	storage    storage.Storage
	workerPool *worker.Pool
	collector  *metrics.Collector
	interval   time.Duration
}

// NewScanner 创建扫描器
func NewScanner(storage storage.Storage, workerPool *worker.Pool, collector *metrics.Collector, interval time.Duration) *Scanner {
	return &Scanner{
		storage:    storage,
		workerPool: workerPool,
		collector:  collector,
		interval:   interval,
	}
}

// Start 启动扫描器
func (s *Scanner) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	logger.Info("Scanner started", zap.Duration("interval", s.interval))

	for {
		select {
		case <-ctx.Done():
			logger.Info("Scanner stopping")
			return

		case <-ticker.C:
			s.scan()
		}
	}
}

// scan 执行一次扫描
func (s *Scanner) scan() {
	tasks, err := s.storage.GetDueTasks()
	if err != nil {
		logger.Error("Failed to get due tasks", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	logger.Info("Found due tasks", zap.Int("count", len(tasks)))

	for _, task := range tasks {
		// 从 Redis 删除任务（防止重复处理）
		if err := s.storage.Remove(task.TaskID); err != nil {
			logger.Error("Failed to remove task", err, zap.String("task_id", task.TaskID))
			continue
		}

		// 提交到 Worker Pool
		if !s.workerPool.Submit(task) {
			logger.Warn("Worker pool full, task dropped", zap.String("task_id", task.TaskID))
		} else {
			logger.Debug("Task submitted to worker pool", zap.String("task_id", task.TaskID))
		}
	}
}
