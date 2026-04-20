package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"go-delay-queue/pkg/task"
)

// RedisStorage Redis 存储实现
type RedisStorage struct {
	client          *redis.Client
	zsetKey         string // ZSet 索引 key
	detailKeyPrefix string // 任务详情 key 前缀
}

// NewRedisStorage 创建 Redis 存储实例
func NewRedisStorage(addr, password string, db int) *RedisStorage {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisStorage{
		client:          client,
		zsetKey:         "delay:queue:tasks",
		detailKeyPrefix: "task:detail:",
	}
}

// Ping 检查 Redis 连接
func (r *RedisStorage) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// detailKey 生成任务详情 key
func (r *RedisStorage) detailKey(taskID string) string {
	return r.detailKeyPrefix + taskID
}

// Add 添加任务到 Redis
// 使用 Pipeline 事务保证原子性：
// 1. ZSet 存索引 (member=task_id, score=execute_at)
// 2. String 存详情 (key=task:detail:{task_id})
func (r *RedisStorage) Add(t *task.Task) error {
	ctx := context.Background()

	// 1. 检查 ZSet 中是否已存在该 task_id
	exists, err := r.client.ZScore(ctx, r.zsetKey, t.TaskID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("check task exists failed: %w", err)
	}
	if exists > 0 {
		return ErrTaskExists
	}

	// 2. 序列化任务详情
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task failed: %w", err)
	}

	// 3. 使用 Pipeline 原子执行 ZAdd 和 Set
	pipe := r.client.Pipeline()
	pipe.ZAdd(ctx, r.zsetKey, redis.Z{
		Score:  float64(t.ExecuteAt),
		Member: t.TaskID, // 只存 task_id
	})
	pipe.Set(ctx, r.detailKey(t.TaskID), data, 0) // 0 表示永不过期

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline exec failed: %w", err)
	}

	return nil
}

// PopDueTask 弹出并返回一个到期任务（原子操作）
func (r *RedisStorage) PopDueTask() (*task.Task, error) {
	ctx := context.Background()
	now := time.Now().Unix()

	// 1. 使用 ZRANGEBYSCORE 查询到期任务（只取第一个）
	results, err := r.client.ZRangeByScore(ctx, r.zsetKey, &redis.ZRangeBy{
		Min:    "0",
		Max:    strconv.FormatInt(now, 10),
		Offset: 0,
		Count:  1,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("zrangebyscore failed: %w", err)
	}

	// 没有到期任务
	if len(results) == 0 {
		return nil, nil
	}

	taskID := results[0]

	// 2. 尝试删除 ZSet 中的索引（ZREM）
	removed, err := r.client.ZRem(ctx, r.zsetKey, taskID).Result()
	if err != nil {
		return nil, fmt.Errorf("zrem failed: %w", err)
	}

	// 如果 removed == 0，说明被其他节点抢走了
	if removed == 0 {
		return nil, nil
	}

	// 3. 获取任务详情
	data, err := r.client.Get(ctx, r.detailKey(taskID)).Result()
	if err == redis.Nil {
		// 详情不存在，说明数据不一致
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task detail failed: %w", err)
	}

	// 4. 删除任务详情（可选，也可以保留用于审计）
	// r.client.Del(ctx, r.detailKey(taskID))

	// 5. 反序列化任务
	var t task.Task
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, fmt.Errorf("unmarshal task failed: %w", err)
	}

	return &t, nil
}

// GetDueTasks 获取所有到期任务（非原子，仅用于调试）
func (r *RedisStorage) GetDueTasks() ([]*task.Task, error) {
	ctx := context.Background()
	now := time.Now().Unix()

	// 从 ZSet 获取所有到期任务的 task_id
	taskIDs, err := r.client.ZRangeByScore(ctx, r.zsetKey, &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(now, 10),
	}).Result()

	if err != nil {
		return nil, err
	}

	// 批量获取任务详情
	var tasks []*task.Task
	for _, taskID := range taskIDs {
		data, err := r.client.Get(ctx, r.detailKey(taskID)).Result()
		if err != nil {
			continue
		}

		var t task.Task
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			continue
		}
		tasks = append(tasks, &t)
	}

	return tasks, nil
}

// GetAll 获取所有任务（调试用）
func (r *RedisStorage) GetAll() []*task.Task {
	ctx := context.Background()

	// 获取所有 task_id
	taskIDs, err := r.client.ZRange(ctx, r.zsetKey, 0, -1).Result()
	if err != nil {
		return nil
	}

	var tasks []*task.Task
	for _, taskID := range taskIDs {
		data, err := r.client.Get(ctx, r.detailKey(taskID)).Result()
		if err != nil {
			continue
		}

		var t task.Task
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			continue
		}
		tasks = append(tasks, &t)
	}

	return tasks
}

// GetPendingCount 获取待处理任务数量
func (r *RedisStorage) GetPendingCount() (int64, error) {
	ctx := context.Background()
	return r.client.ZCard(ctx, r.zsetKey).Result()
}

// Close 关闭 Redis 连接
func (r *RedisStorage) Close() error {
	return r.client.Close()
}