package metrics

import (
	"sync"
	"time"
)

// Collector 指标收集器
type Collector struct {
	mu sync.RWMutex

	// 任务统计
	pendingTasks    int64
	processingTasks int64
	completedTasks  int64
	failedTasks     int64

	// 执行统计
	totalExecutions int64
	retryCount      int64

	// 启动时间
	startTime time.Time
}

// NewCollector 创建指标收集器
func NewCollector() *Collector {
	return &Collector{
		startTime: time.Now(),
	}
}

// Stats 统计数据
type Stats struct {
	PendingTasks    int64         `json:"pending"`
	ProcessingTasks int64         `json:"processing"`
	CompletedTasks  int64         `json:"completed"`
	FailedTasks     int64         `json:"failed"`
	TotalTasks      int64         `json:"total_tasks"`
	TotalExecutions int64         `json:"total_executions"`
	RetryCount      int64         `json:"retry_count"`
	Uptime          time.Duration `json:"uptime"`
}

// RecordPending 记录待处理任务
func (c *Collector) RecordPending(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pendingTasks += delta
}

// RecordProcessing 记录处理中任务
func (c *Collector) RecordProcessing(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processingTasks += delta
}

// RecordComplete 记录完成任务
func (c *Collector) RecordComplete() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.completedTasks++
	c.totalExecutions++
	if c.processingTasks > 0 {
		c.processingTasks--
	}
}

// RecordFail 记录失败任务
func (c *Collector) RecordFail() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failedTasks++
	c.totalExecutions++
	if c.processingTasks > 0 {
		c.processingTasks--
	}
}

// RecordRetry 记录重试
func (c *Collector) RecordRetry() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryCount++
}

// GetStats 获取统计数据
func (c *Collector) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Stats{
		PendingTasks:    c.pendingTasks,
		ProcessingTasks: c.processingTasks,
		CompletedTasks:  c.completedTasks,
		FailedTasks:     c.failedTasks,
		TotalTasks:      c.pendingTasks + c.processingTasks + c.completedTasks + c.failedTasks,
		TotalExecutions: c.totalExecutions,
		RetryCount:      c.retryCount,
		Uptime:          time.Since(c.startTime),
	}
}
