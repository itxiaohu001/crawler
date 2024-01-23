package sleeper

import (
	"math/rand"
	"time"
)

// Sleeper 用于管理睡眠行为
type Sleeper struct {
	baseDelay time.Duration
	jitter    time.Duration
}

// NewSleeper 创建一个新的Sleeper实例
func NewSleeper(baseDelay, jitter time.Duration) *Sleeper {
	return &Sleeper{
		baseDelay: baseDelay,
		jitter:    jitter,
	}
}

// Sleep 执行带有随机浮动的睡眠
func (s *Sleeper) Sleep() {
	delay := s.baseDelay + time.Duration(rand.Int63n(int64(s.jitter)))
	time.Sleep(delay)
}
