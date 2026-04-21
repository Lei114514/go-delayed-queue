package task

import "time"

// Task 延迟任务结构体
type Task struct {
	// TaskID 任务唯一标识，用于去重
	TaskID string `json:"task_id"`

	// TaskType 任务类型，如：email、sms
	TaskType string `json:"task_type"`

	// ExecuteAt 执行时间（Unix 时间戳）
	ExecuteAt int64 `json:"execute_at"`

	// Payload 任务数据，根据 TaskType 变化
	Payload map[string]interface{} `json:"payload"`

	// RetryCount 当前重试次数
	RetryCount int `json:"retry_count"`

	// MaxRetry 最大重试次数（默认 3）
	MaxRetry int `json:"max_retry"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"-"`
}

// IsDue 检查任务是否到期
func (t *Task) IsDue() bool {
	return time.Now().Unix() >= t.ExecuteAt
}

// CanRetry 检查是否还可以重试
func (t *Task) CanRetry() bool {
	maxRetry := t.MaxRetry
	if maxRetry == 0 {
		maxRetry = 3 // 默认最多重试 3 次
	}
	return t.RetryCount < maxRetry
}

// MarkRetry 标记重试，更新执行时间和重试次数
func (t *Task) MarkRetry(nextExecuteAt int64) {
	t.RetryCount++
	t.ExecuteAt = nextExecuteAt
}

// NewTask 创建任务（设置默认值）
func NewTask(taskID, taskType string, executeAt int64, payload map[string]interface{}) *Task {
	return &Task{
		TaskID:     taskID,
		TaskType:   taskType,
		ExecuteAt:  executeAt,
		Payload:    payload,
		RetryCount: 0,
		MaxRetry:   3, // 默认重试 3 次
		CreatedAt:  time.Now(),
	}
}
