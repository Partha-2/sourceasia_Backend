package ratelimiter

import (
	"sync"
	"time"
)

const (
	MaxRequestsPerWindow = 5
	WindowDuration       = 1 * time.Minute
)

type UserBucket struct {
	mu          sync.Mutex
	Count       int
	WindowStart time.Time
	Rejected    int
}

type RateLimiter struct {
	mu    sync.RWMutex
	users map[string]*UserBucket
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		users: make(map[string]*UserBucket),
	}
}

func (rl *RateLimiter) Allow(userID string) (bool, int) {
	rl.mu.RLock()
	bucket, exists := rl.users[userID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		bucket, exists = rl.users[userID]
		if !exists {
			bucket = &UserBucket{
				WindowStart: time.Now(),
			}
			rl.users[userID] = bucket
		}
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	if now.Sub(bucket.WindowStart) >= WindowDuration {
		bucket.Count = 0
		bucket.WindowStart = now
	}

	if bucket.Count >= MaxRequestsPerWindow {
		bucket.Rejected++
		elapsed := now.Sub(bucket.WindowStart)
		retryAfter := int(WindowDuration.Seconds() - elapsed.Seconds())
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	bucket.Count++
	return true, 0
}

type UserStats struct {
	AcceptedCurrentWindow int `json:"accepted_current_window"`
	RejectedCumulative    int `json:"rejected_cumulative"`
}

func (rl *RateLimiter) Stats() map[string]UserStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := make(map[string]UserStats, len(rl.users))
	for id, bucket := range rl.users {
		bucket.mu.Lock()
		now := time.Now()
		accepted := bucket.Count
		if now.Sub(bucket.WindowStart) >= WindowDuration {
			accepted = 0
		}
		stats[id] = UserStats{
			AcceptedCurrentWindow: accepted,
			RejectedCumulative:    bucket.Rejected,
		}
		bucket.mu.Unlock()
	}
	return stats
}
