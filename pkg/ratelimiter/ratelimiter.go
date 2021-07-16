package ratelimiter

import (
	"sync"
	"time"
)

type RateLimiter interface {
	Start(fillRate time.Duration, capacity int)
	GetToken() bool
	fillBucket()
	startFill()
}

type Limiter struct {
	capacity int
	fillRate time.Duration
	tokens   int
	mu       sync.Mutex
}

func (limiter *Limiter) Start(fillRate time.Duration, capacity int) *Limiter {
	limiter.capacity = capacity
	limiter.tokens = capacity
	limiter.fillRate = fillRate
	go limiter.startFill()
	return limiter
}

func (limiter *Limiter) GetToken() bool {
	for limiter.getTokenCount() < 1 {
		time.Sleep(limiter.fillRate)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.tokens--
	return true
}

func (limiter *Limiter) getTokenCount() int {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	return limiter.tokens
}

func (limiter *Limiter) startFill() {
	for {
		time.Sleep(limiter.fillRate)
		limiter.fillBucket()
	}
}

func (limiter *Limiter) fillBucket() {
	limiter.mu.Lock()
	if limiter.tokens < limiter.capacity {
		limiter.tokens++
	}
	limiter.mu.Unlock()
}
