package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-delay-queue/internal/logger"
	"go-delay-queue/internal/metrics"
	"go-delay-queue/internal/storage"
	"go-delay-queue/pkg/task"
)

// Handler API 处理器
type Handler struct {
	storage   storage.Storage
	collector *metrics.Collector
}

// NewHandler 创建 Handler
func NewHandler(storage storage.Storage, collector *metrics.Collector) *Handler {
	return &Handler{
		storage:   storage,
		collector: collector,
	}
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	TaskID    string                 `json:"task_id" binding:"required"`
	TaskType  string                 `json:"task_type" binding:"required"`
	ExecuteAt int64                  `json:"execute_at" binding:"required"`
	Payload   map[string]interface{} `json:"payload"`
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	} `json:"data"`
}

// CreateTask 创建延迟任务
func (h *Handler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Info("Invalid request",
			zap.String("error", err.Error()),
			zap.String("client_ip", c.ClientIP()),
		)
		c.JSON(http.StatusBadRequest, CreateTaskResponse{
			Code:    400,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	t := task.NewTask(req.TaskID, req.TaskType, req.ExecuteAt, req.Payload)

	if err := h.storage.Add(t); err != nil {
		if err == storage.ErrTaskExists {
			logger.Info("Task already exists",
				zap.String("task_id", req.TaskID),
			)
			c.JSON(http.StatusConflict, CreateTaskResponse{
				Code:    409,
				Message: "task already exists",
			})
			return
		}
		logger.Error("Failed to create task", err,
			zap.String("task_id", req.TaskID),
		)
		c.JSON(http.StatusInternalServerError, CreateTaskResponse{
			Code:    500,
			Message: "internal error",
		})
		return
	}

	h.collector.RecordPending(1)
	logger.Info("Task created",
		zap.String("task_id", req.TaskID),
		zap.String("type", req.TaskType),
		zap.Int64("execute_at", req.ExecuteAt),
	)

	resp := CreateTaskResponse{
		Code:    0,
		Message: "success",
	}
	resp.Data.TaskID = req.TaskID
	resp.Data.Status = "pending"

	c.JSON(http.StatusOK, resp)
}

// ListTasks 列出所有任务（调试用）
func (h *Handler) ListTasks(c *gin.Context) {
	tasks := h.storage.GetAll()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": tasks,
	})
}

// GetMetrics 获取监控指标
func (h *Handler) GetMetrics(c *gin.Context) {
	stats := h.collector.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
	})
}

// HealthCheck 健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"status": "healthy",
		},
	})
}
