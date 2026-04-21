package api

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    
    "go-delay-queue/internal/storage"
    "go-delay-queue/pkg/task"
)

type Handler struct {
	storage storage.Storage  // 使用接口
}

func NewHandler(storage storage.Storage) *Handler {
	return &Handler{storage: storage}
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
        c.JSON(http.StatusBadRequest, CreateTaskResponse{
            Code:    400,
            Message: "参数错误: " + err.Error(),
        })
        return
    }
    
    t := task.NewTask(req.TaskID, req.TaskType, req.ExecuteAt, req.Payload)
    
    if err := h.storage.Add(t); err != nil {
        if err == storage.ErrTaskExists {
            c.JSON(http.StatusConflict, CreateTaskResponse{
                Code:    409,
                Message: "task already exists",
            })
            return
        }
        c.JSON(http.StatusInternalServerError, CreateTaskResponse{
            Code:    500,
            Message: "internal error",
        })
        return
    }
    
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