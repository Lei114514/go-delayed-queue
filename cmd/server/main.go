package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"go-delay-queue/internal/api"
	"go-delay-queue/internal/handler"
	"go-delay-queue/internal/scheduler"
	"go-delay-queue/internal/storage"
	"go-delay-queue/internal/worker"
)

func main() {
	// 创建可取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化 Redis 存储
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	store := storage.NewRedisStorage(redisAddr, "", 0)

	// 检查 Redis 连接
	if err := store.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// 初始化 Handler Registry
	registry := handler.NewRegistry()
	registry.Register(handler.NewEmailHandler())
	log.Printf("Registered handlers: %v\n", registry.List())

	// 初始化 Worker Pool（5 个 worker，传入 storage 用于重试）
	pool := worker.NewPool(5, registry, store)
	pool.Start(ctx)
	log.Println("Worker pool started with 5 workers")

	// 初始化 Scanner（每秒扫描一次）
	scanner := scheduler.NewScanner(store, pool, time.Second)
	go scanner.Start(ctx)
	log.Println("Scanner started")

	// 初始化 HTTP Handler
	httpHandler := api.NewHandler(store)

	// 设置 HTTP 路由
	r := gin.Default()
	r.POST("/task", httpHandler.CreateTask)
	r.GET("/tasks", httpHandler.ListTasks)

	// 启动 HTTP 服务（非阻塞）
	port := getEnv("PORT", "8082")  // 使用 8082 端口
	go func() {
		if err := r.Run(":" + port); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	log.Printf("HTTP server started on :%s\n", port)

	// 等待中断信号（优雅关闭）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// 1. 停止 Scanner（取消 context）
	cancel()
	time.Sleep(100 * time.Millisecond) // 给 Scanner 时间退出

	// 2. 停止 Worker Pool
	pool.Stop()

	// 3. 关闭 Redis
	store.Close()

	log.Println("Shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
