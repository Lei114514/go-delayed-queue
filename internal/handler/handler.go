package handler

import "go-delay-queue/pkg/task"

// TaskHandler 任务处理器接口
type TaskHandler interface {
	// Name 返回处理器名称
	Name() string

	// Handle 执行任务
	// 返回 error 表示执行失败，会触发重试
	Handle(t *task.Task) error
}

// Registry 任务处理器注册中心
type Registry struct {
	handlers map[string]TaskHandler
}

// NewRegistry 创建注册中心
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]TaskHandler),
	}
}

// Register 注册处理器
func (r *Registry) Register(handler TaskHandler) {
	r.handlers[handler.Name()] = handler
}

// Get 获取处理器
func (r *Registry) Get(taskType string) (TaskHandler, bool) {
	h, ok := r.handlers[taskType]
	return h, ok
}

// List 列出所有支持的处理器
func (r *Registry) List() []string {
	var names []string
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}