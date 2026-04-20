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

	// RetryCount 重试次数（内部使用，不对外暴露）
	RetryCount int `json:"-"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"-"`
}

// IsDue 检查任务是否到期
func (t *Task) IsDue() bool {
	return time.Now().Unix() >= t.ExecuteAt
}
