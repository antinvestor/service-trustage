package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/ratelimiter"
	"github.com/pitabwire/frame/security"
)

// RateLimiter enforces per-tenant rate limits using Frame's cache-backed window limiter.
type RateLimiter struct {
	limiter *ratelimiter.WindowLimiter
}

// NewRateLimiter creates a new rate limiter.
// If cache is nil or maxPerWindow <= 0, rate limiting is disabled.
func NewRateLimiter(raw cache.RawCache, maxPerWindow int) *RateLimiter {
	if raw == nil || maxPerWindow <= 0 {
		return &RateLimiter{limiter: nil}
	}

	cfg := ratelimiter.DefaultWindowConfig()
	cfg.WindowDuration = time.Minute
	cfg.MaxPerWindow = maxPerWindow
	cfg.KeyPrefix = "trustage:formstore:tenant"
	cfg.FailOpen = true

	limiter, err := ratelimiter.NewWindowLimiter(raw, cfg)
	if err != nil {
		return &RateLimiter{limiter: nil}
	}

	return &RateLimiter{limiter: limiter}
}

// Allow checks whether a request from the current tenant should be allowed.
func (rl *RateLimiter) Allow(ctx context.Context) bool {
	if rl == nil || rl.limiter == nil {
		return true
	}

	claims := security.ClaimsFromContext(ctx)
	if claims == nil {
		return rl.limiter.Allow(ctx, "unknown")
	}

	key := fmt.Sprintf("%s:%s", claims.GetTenantID(), claims.GetPartitionID())
	return rl.limiter.Allow(ctx, key)
}
