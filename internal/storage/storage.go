package storage

import "go-delay-queue/pkg/task"

type Storage interface {
	Add(t *task.Task) error
	GetDueTasks() ([]*task.Task, error)
	GetPendingCount() (int64, error)
	GetAll() []*task.Task
}
