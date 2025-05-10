package ratelimiter

import (
	"sync"
	"testing"
)

func TestConcurrentRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	clientID := "test-client"
	var wg sync.WaitGroup
	allowed := 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(clientID, 10, 1) {
				allowed++
			}
		}()
	}

	wg.Wait()
	if allowed != 10 {
		t.Errorf("Expected exactly 10 allowed requests, got %d", allowed)
	}
}