package handlers

import (
	"context"
	"fmt"
	"time"

	framecache "github.com/pitabwire/frame/cache"
)

// rateLimitWindow is the sliding window duration for rate limiting.
const rateLimitWindow = 1 * time.Minute

// RateLimiter enforces per-tenant rate limits using Valkey atomic counters.
type RateLimiter struct {
	cache     framecache.RawCache
	maxPerWin int
}

// NewRateLimiter creates a new rate limiter.
// If cache is nil, rate limiting is disabled (all requests allowed).
func NewRateLimiter(cache framecache.RawCache, maxPerWindow int) *RateLimiter {
	return &RateLimiter{cache: cache, maxPerWin: maxPerWindow}
}

// Allow checks whether a request from the given tenant should be allowed.
// Returns true if allowed, false if rate-limited.
func (rl *RateLimiter) Allow(ctx context.Context, tenantID string) bool {
	if rl.cache == nil || rl.maxPerWin <= 0 {
		return true
	}

	// Use a time-bucketed key so counters auto-expire each window.
	bucket := time.Now().Unix() / int64(rateLimitWindow.Seconds())
	key := fmt.Sprintf("rl:event:%s:%d", tenantID, bucket)

	count, err := rl.cache.Increment(ctx, key, 1)
	if err != nil {
		// On cache failure, allow the request (fail-open).
		return true
	}

	// Set TTL on the first increment so the key expires after the window.
	if count == 1 {
		_ = rl.cache.Set(ctx, key, []byte("1"), rateLimitWindow+time.Second)
	}

	return count <= int64(rl.maxPerWin)
}
