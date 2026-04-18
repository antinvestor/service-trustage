// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"context"
	"fmt"
	"time"

	framecache "github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/ratelimiter"
	"github.com/pitabwire/frame/security"
)

// RateLimiter enforces per-tenant rate limits with chunked shared-cache reservations.
type RateLimiter struct {
	limiter *ratelimiter.LeasedWindowLimiter
}

// NewRateLimiter creates the default event-ingest limiter.
func NewRateLimiter(cache framecache.RawCache, maxPerWindow int) *RateLimiter {
	return NewNamedRateLimiter(cache, "trustage:event_ingest", maxPerWindow)
}

// NewNamedRateLimiter creates a tenant limiter for a specific ingress class.
func NewNamedRateLimiter(cache framecache.RawCache, keyPrefix string, maxPerWindow int) *RateLimiter {
	limiter, err := ratelimiter.NewLeasedWindowLimiter(cache, &ratelimiter.WindowConfig{
		WindowDuration: time.Minute,
		MaxPerWindow:   maxPerWindow,
		KeyPrefix:      keyPrefix,
		FailOpen:       true,
	})
	if err != nil {
		return nil
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
