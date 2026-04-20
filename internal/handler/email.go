package handler

import (
	"fmt"
	"time"

	"go-delay-queue/pkg/task"
)

// EmailHandler 邮件处理器
type EmailHandler struct {
	successRate float64 // 模拟成功率，用于测试重试机制
}

// NewEmailHandler 创建邮件处理器
func NewEmailHandler() *EmailHandler {
	return &EmailHandler{
		successRate: 0.7, // 70% 成功率
	}
}

// Name 返回处理器名称
func (h *EmailHandler) Name() string {
	return "email"
}

// Handle 执行邮件发送（模拟）
func (h *EmailHandler) Handle(t *task.Task) error {
	// 从 payload 获取参数
	to, _ := t.Payload["to"].(string)
	subject, _ := t.Payload["subject"].(string)
	content, _ := t.Payload["content"].(string)

	if to == "" {
		return fmt.Errorf("missing 'to' in payload")
	}

	// 模拟网络延迟
	time.Sleep(100 * time.Millisecond)

	// 模拟随机失败（用于测试重试）
	// 实际项目中删除这段代码
	// if rand.Float64() > h.successRate {
	//     return fmt.Errorf("email service temporarily unavailable")
	// }

	// 实际发送逻辑（这里只打印）
	fmt.Printf("[EmailHandler] Sending email to: %s\n", to)
	fmt.Printf("  Subject: %s\n", subject)
	fmt.Printf("  Content: %s\n", content)
	fmt.Printf("  TaskID: %s\n\n", t.TaskID)

	return nil
}