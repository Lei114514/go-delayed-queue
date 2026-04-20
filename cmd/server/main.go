package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"go-delay-queue/internal/api"
	"go-delay-queue/internal/storage"
)

func main() {
	// 从环境变量读取 Redis 配置
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0 // 简化处理，实际需要 parseInt

	// 初始化 Redis 存储
	store := storage.NewRedisStorage(redisAddr, redisPassword, redisDB)

	// 检查 Redis 连接
	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// 初始化 Handler
	handler := api.NewHandler(store)

	// 设置路由
	r := gin.Default()
	r.POST("/task", handler.CreateTask)
	r.GET("/tasks", handler.ListTasks)
	// r.GET("/health", handler.HealthCheck)  // Day 6 再实现

	// 启动服务
	port := getEnv("PORT", "8081")
	r.Run(":" + port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}