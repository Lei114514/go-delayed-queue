package retry

import (
	"math"
	"time"
)

// Strategy 重试策略接口
type Strategy interface {
	// NextDelay 计算下次重试的延迟时间
	NextDelay(retryCount int) time.Duration
}

// ExponentialBackoff 指数退避策略
type ExponentialBackoff struct {
	InitialInterval time.Duration // 初始间隔（默认 1 秒）
	MaxInterval     time.Duration // 最大间隔（默认 60 秒）
	Multiplier      float64       // 乘数（默认 2）
}

// NewExponentialBackoff 创建指数退避策略
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialInterval: 1 * time.Second,
		MaxInterval:     60 * time.Second,
		Multiplier:      2,
	}
}

// NextDelay 计算下次延迟
// 公式：min(InitialInterval * Multiplier^retryCount, MaxInterval)
func (e *ExponentialBackoff) NextDelay(retryCount int) time.Duration {
	if retryCount == 0 {
		return 0
	}

	// 计算指数退避：1s, 2s, 4s, 8s, 16s, 32s, 60s, 60s...
	delay := float64(e.InitialInterval) * math.Pow(e.Multiplier, float64(retryCount-1))

	// 不超过最大间隔
	if delay > float64(e.MaxInterval) {
		delay = float64(e.MaxInterval)
	}

	return time.Duration(delay)
}

// FixedDelay 固定延迟策略
type FixedDelay struct {
	Delay time.Duration
}

// NewFixedDelay 创建固定延迟策略
func NewFixedDelay(delay time.Duration) *FixedDelay {
	return &FixedDelay{Delay: delay}
}

// NextDelay 返回固定延迟
func (f *FixedDelay) NextDelay(retryCount int) time.Duration {
	if retryCount == 0 {
		return 0
	}
	return f.Delay
}

// NoRetry 不重试策略
type NoRetry struct{}

// NextDelay 永远返回 0（表示不重试）
func (n *NoRetry) NextDelay(retryCount int) time.Duration {
	return 0
}
