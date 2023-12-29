package utils

import (
	"fmt"
	"sync"
)

type Bar struct {
	mu    sync.Mutex
	total int
	cur   int
}

func NewBar(t int) *Bar {
	return &Bar{
		mu:    sync.Mutex{},
		total: t,
		cur:   0,
	}
}

func (b *Bar) Add() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cur++
}

func (b *Bar) Print() {
	fmt.Printf("\r当前进度 %d/%d %.2f%%", b.cur, b.total, percentage(b.cur, b.total))
}

func percentage(current, total int) float64 {
	per := calculatePercentage(current, total)
	return per
}

func calculatePercentage(current, total int) float64 {
	if total == 0 {
		return 0
	}
	return (float64(current) / float64(total)) * 100
}
