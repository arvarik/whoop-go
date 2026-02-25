package whoop

import (
	"context"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// rateLimiter encapsulates the local token-bucket rate limiting
// to ensure we do not exceed WHOOP's limits (100 req/min).
type rateLimiter struct {
	limiter        *rate.Limiter
	isAutoLimiting atomic.Bool
}

// newRateLimiter initializes a rate limiter configured for 100 requests per minute.
// The burst is set to 100 to allow initial rapid requests up to the limit.
func newRateLimiter() *rateLimiter {
	// 100 requests per minute = 100 / 60 requests per second
	limit := rate.Limit(100.0 / 60.0)

	// Ensure we initialize rand for jitter
	// Note: rand.Seed is deprecated in Go 1.20 as the global RNG is automatically seeded.

	rl := &rateLimiter{
		limiter: rate.NewLimiter(limit, 100),
	}
	rl.isAutoLimiting.Store(true) // Default to honoring local rate limits
	return rl
}

// Wait blocks until a token is available or the context is canceled.
func (rl *rateLimiter) Wait(ctx context.Context) error {
	if !rl.isAutoLimiting.Load() {
		return nil
	}
	return rl.limiter.Wait(ctx)
}

// SetAutoLimiting enables or disables the rate limiter.
func (rl *rateLimiter) SetAutoLimiting(enabled bool) {
	rl.isAutoLimiting.Store(enabled)
}

// calculateBackoff computes the duration to wait before the next retry attempt
// using exponential backoff with full jitter to avoid thundering herd.
func calculateBackoff(attempt int, base, max time.Duration) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	if max <= 0 {
		max = 60 * time.Second
	}

	// Exponential backoff: base * 2^attempt
	backoff := float64(base) * math.Pow(2, float64(attempt))

	// Cap at maximum backoff
	if backoff > float64(max) {
		backoff = float64(max)
	}

	// Apply full jitter
	// jitter = rand_between(0, backoff)
	jitter := rand.Float64() * backoff

	return time.Duration(jitter)
}
