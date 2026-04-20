package storage

import (
    "errors"
    "sync"
    "time"
    
    "go-delay-queue/pkg/task"
)

var (
    ErrTaskExists = errors.New("task already exists")
)

// MemoryStorage 内存存储实现（临时使用）
type MemoryStorage struct {
    mu     sync.RWMutex
    tasks  map[string]*task.Task  // key: task_id
}

func NewMemoryStorage() *MemoryStorage {
    return &MemoryStorage{
        tasks: make(map[string]*task.Task),
    }
}

// Add 添加任务
func (m *MemoryStorage) Add(t *task.Task) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // 去重检查
    if _, exists := m.tasks[t.TaskID]; exists {
        return ErrTaskExists
    }
    
    t.CreatedAt = time.Now()
    m.tasks[t.TaskID] = t
    return nil
}

// GetDueTasks 获取所有到期任务
func (m *MemoryStorage) GetDueTasks() ([]*task.Task, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    var dueTasks []*task.Task
    now := time.Now().Unix()
    
    for _, t := range m.tasks {
        if t.ExecuteAt <= now {
            dueTasks = append(dueTasks, t)
        }
    }
    
    return dueTasks, nil
}

// Remove 删除任务
func (m *MemoryStorage) Remove(taskID string) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    delete(m.tasks, taskID)
    return nil
}

// GetAll 获取所有任务（调试用）
func (m *MemoryStorage) GetAll() []*task.Task {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    var result []*task.Task
    for _, t := range m.tasks {
        result = append(result, t)
    }
    return result
}