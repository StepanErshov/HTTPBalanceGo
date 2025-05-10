package ratelimiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	capacity     int
	rate         int
	tokens       int
	lastRefill   time.Time
	mutex        sync.Mutex
}

func NewTokenBucket(capacity, rate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		rate:       rate,
		tokens:     capacity,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * tb.rate

	if tokensToAdd > 0 {
		if tb.tokens+tokensToAdd > tb.capacity {
			tb.tokens = tb.capacity
		} else {
			tb.tokens += tokensToAdd
		}
		tb.lastRefill = now
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

type RateLimiter struct {
	buckets map[string]*TokenBucket
	mutex   sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
	}
}

func (rl *RateLimiter) Allow(clientID string, capacity, rate int) bool {
	rl.mutex.RLock()
	bucket, exists := rl.buckets[clientID]
	rl.mutex.RUnlock()

	if !exists {
		rl.mutex.Lock()
		bucket = NewTokenBucket(capacity, rate)
		rl.buckets[clientID] = bucket
		rl.mutex.Unlock()
	}

	return bucket.Allow()
}