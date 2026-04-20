package main

import (
    "github.com/gin-gonic/gin"
    
    "go-delay-queue/internal/api"
    "go-delay-queue/internal/storage"
)

func main() {
    // 初始化存储
    store := storage.NewMemoryStorage()
    
    // 初始化 Handler
    handler := api.NewHandler(store)
    
    // 设置路由
    r := gin.Default()
    r.POST("/task", handler.CreateTask)
    r.GET("/tasks", handler.ListTasks)  // 调试用
    
    // 启动服务
    r.Run(":8081")
}