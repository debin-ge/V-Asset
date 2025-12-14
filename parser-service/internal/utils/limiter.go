package utils

// ConcurrencyLimiter 并发限制器
type ConcurrencyLimiter struct {
	sem chan struct{}
}

// NewConcurrencyLimiter 创建并发限制器
func NewConcurrencyLimiter(max int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		sem: make(chan struct{}, max),
	}
}

// Acquire 获取信号量
func (l *ConcurrencyLimiter) Acquire() {
	l.sem <- struct{}{}
}

// Release 释放信号量
func (l *ConcurrencyLimiter) Release() {
	<-l.sem
}
