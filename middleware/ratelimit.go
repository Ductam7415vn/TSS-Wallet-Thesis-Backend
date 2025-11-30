package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"tss-wallet-backend/config"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rps      int
	burst    int
	cleanup  time.Duration
}

type tokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     rps,
		burst:   burst,
		cleanup: 10 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop removes stale buckets periodically
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			if now.Sub(bucket.lastUpdate) > rl.cleanup {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string, customRPS int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rps := rl.rps
	if customRPS > 0 {
		rps = customRPS
	}

	now := time.Now()
	bucket, exists := rl.buckets[key]

	if !exists {
		rl.buckets[key] = &tokenBucket{
			tokens:     float64(rl.burst) - 1,
			lastUpdate: now,
		}
		return true
	}

	// Add tokens based on elapsed time
	elapsed := now.Sub(bucket.lastUpdate).Seconds()
	bucket.tokens += elapsed * float64(rps)
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}
	bucket.lastUpdate = now

	// Check if we have tokens available
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

// RemainingTokens returns the number of remaining tokens for a key
func (rl *RateLimiter) RemainingTokens(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return rl.burst
	}

	return int(bucket.tokens)
}

// Global rate limiter instance
var globalRateLimiter *RateLimiter

// InitRateLimiter initializes the global rate limiter
func InitRateLimiter() {
	globalRateLimiter = NewRateLimiter(
		config.AppConfig.RateLimitRPS,
		config.AppConfig.RateLimitBurst,
	)
}

// RateLimitMiddleware applies rate limiting to requests
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if globalRateLimiter == nil {
			InitRateLimiter()
		}

		// Use client IP as the rate limit key
		key := c.ClientIP()

		// Check if there's a custom rate limit from API key
		customRPS := 0
		if rps, exists := c.Get("rate_limit_rps"); exists {
			customRPS = rps.(int)
		}

		// Check rate limit
		if !globalRateLimiter.Allow(key, customRPS) {
			remaining := globalRateLimiter.RemainingTokens(key)
			c.Header("X-RateLimit-Remaining", string(rune(remaining)))
			c.Header("X-RateLimit-Limit", string(rune(config.AppConfig.RateLimitRPS)))
			c.Header("Retry-After", "1")

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please slow down.",
				"retry_after": 1,
			})
			return
		}

		// Add rate limit headers
		remaining := globalRateLimiter.RemainingTokens(key)
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))
		c.Header("X-RateLimit-Limit", string(rune(config.AppConfig.RateLimitRPS)))

		c.Next()
	}
}

// StrictRateLimitMiddleware applies stricter rate limiting for sensitive endpoints
func StrictRateLimitMiddleware(rps, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rps, burst)

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.Allow(key, 0) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests to this endpoint. Please wait before trying again.",
				"retry_after": 5,
			})
			return
		}

		c.Next()
	}
}

// KeyGenRateLimitMiddleware applies rate limiting specifically for KeyGen operations
// KeyGen is expensive, so we limit it more strictly
func KeyGenRateLimitMiddleware() gin.HandlerFunc {
	// Allow only 1 keygen per minute per IP
	limiter := NewRateLimiter(1, 2)

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.Allow(key, 0) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "keygen_rate_limit",
				"message": "KeyGen is a resource-intensive operation. Please wait before creating another wallet.",
				"retry_after": 60,
			})
			return
		}

		c.Next()
	}
}

// SigningRateLimitMiddleware applies rate limiting for signing operations
func SigningRateLimitMiddleware() gin.HandlerFunc {
	// Allow 5 signing operations per second per IP
	limiter := NewRateLimiter(5, 10)

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !limiter.Allow(key, 0) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "signing_rate_limit",
				"message": "Too many signing requests. Please slow down.",
				"retry_after": 1,
			})
			return
		}

		c.Next()
	}
}
