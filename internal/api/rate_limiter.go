// Package api provides HTTP API functionality
package api

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiterConfig configures the rate limiter
type RateLimiterConfig struct {
	// Global rate limiting
	GlobalRequestsPerSecond int
	GlobalBurst             int

	// Per-IP rate limiting
	IPRequestsPerSecond int
	IPBurst             int

	// Per-endpoint rate limiting (optional overrides)
	EndpointLimits map[string]EndpointLimit

	// Cleanup interval for expired entries
	CleanupInterval time.Duration

	// Enable/disable
	Enabled bool
}

// EndpointLimit defines rate limits for a specific endpoint
type EndpointLimit struct {
	RequestsPerSecond int
	Burst             int
	// Methods to apply limit to (empty = all methods)
	Methods []string
}

// DefaultRateLimiterConfig returns sensible defaults
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		GlobalRequestsPerSecond: 1000,
		GlobalBurst:             100,
		IPRequestsPerSecond:     100,
		IPBurst:                 20,
		CleanupInterval:         5 * time.Minute,
		Enabled:                 true,
		EndpointLimits: map[string]EndpointLimit{
			"/api/v1/workflows/execute": {
				RequestsPerSecond: 10,
				Burst:             5,
			},
			"/api/v1/expressions/evaluate": {
				RequestsPerSecond: 50,
				Burst:             10,
			},
			"/api/v1/auth/login": {
				RequestsPerSecond: 5,
				Burst:             3,
			},
		},
	}
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(tokensPerSecond int, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:         float64(burst),
		maxTokens:      float64(burst),
		refillRate:     float64(tokensPerSecond),
		lastRefillTime: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	tb.lastRefillTime = now

	// Refill tokens
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	// Check if we have tokens available
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// Tokens returns the current number of available tokens
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	config         *RateLimiterConfig
	globalBucket   *TokenBucket
	ipBuckets      map[string]*TokenBucket
	endpointBuckets map[string]map[string]*TokenBucket // endpoint -> IP -> bucket
	mu             sync.RWMutex
	stopCleanup    chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &RateLimiter{
		config:          config,
		globalBucket:    NewTokenBucket(config.GlobalRequestsPerSecond, config.GlobalBurst),
		ipBuckets:       make(map[string]*TokenBucket),
		endpointBuckets: make(map[string]map[string]*TokenBucket),
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go rl.cleanupLoop()
	}

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ip string, endpoint string, method string) (bool, int) {
	if !rl.config.Enabled {
		return true, 0
	}

	// Check global limit
	if !rl.globalBucket.Allow() {
		return false, 1
	}

	// Check endpoint-specific limit if configured
	if limit, ok := rl.config.EndpointLimits[endpoint]; ok {
		if len(limit.Methods) == 0 || contains(limit.Methods, method) {
			if !rl.getEndpointBucket(endpoint, ip, limit).Allow() {
				return false, 1
			}
		}
	}

	// Check per-IP limit
	if !rl.getIPBucket(ip).Allow() {
		return false, 1
	}

	return true, 0
}

// getIPBucket gets or creates a token bucket for an IP
func (rl *RateLimiter) getIPBucket(ip string) *TokenBucket {
	rl.mu.RLock()
	bucket, ok := rl.ipBuckets[ip]
	rl.mu.RUnlock()

	if ok {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, ok = rl.ipBuckets[ip]; ok {
		return bucket
	}

	bucket = NewTokenBucket(rl.config.IPRequestsPerSecond, rl.config.IPBurst)
	rl.ipBuckets[ip] = bucket
	return bucket
}

// getEndpointBucket gets or creates a token bucket for an endpoint+IP combination
func (rl *RateLimiter) getEndpointBucket(endpoint, ip string, limit EndpointLimit) *TokenBucket {
	rl.mu.RLock()
	if ipBuckets, ok := rl.endpointBuckets[endpoint]; ok {
		if bucket, ok := ipBuckets[ip]; ok {
			rl.mu.RUnlock()
			return bucket
		}
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, ok := rl.endpointBuckets[endpoint]; !ok {
		rl.endpointBuckets[endpoint] = make(map[string]*TokenBucket)
	}

	if bucket, ok := rl.endpointBuckets[endpoint][ip]; ok {
		return bucket
	}

	bucket := NewTokenBucket(limit.RequestsPerSecond, limit.Burst)
	rl.endpointBuckets[endpoint][ip] = bucket
	return bucket
}

// cleanupLoop periodically cleans up expired buckets
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes buckets that haven't been used recently
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clean up IP buckets that are at max capacity (haven't been used)
	for ip, bucket := range rl.ipBuckets {
		if bucket.Tokens() >= bucket.maxTokens {
			delete(rl.ipBuckets, ip)
		}
	}

	// Clean up endpoint buckets
	for endpoint, ipBuckets := range rl.endpointBuckets {
		for ip, bucket := range ipBuckets {
			if bucket.Tokens() >= bucket.maxTokens {
				delete(ipBuckets, ip)
			}
		}
		if len(ipBuckets) == 0 {
			delete(rl.endpointBuckets, endpoint)
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// Stats returns current rate limiter statistics
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"enabled":          rl.config.Enabled,
		"globalTokens":     rl.globalBucket.Tokens(),
		"trackedIPs":       len(rl.ipBuckets),
		"trackedEndpoints": len(rl.endpointBuckets),
	}
}

// Middleware returns an HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(errorHandler *ErrorHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			endpoint := r.URL.Path

			allowed, retryAfter := rl.Allow(ip, endpoint, r.Method)
			if !allowed {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("X-RateLimit-Limit", "100")
				w.Header().Set("X-RateLimit-Remaining", "0")

				apiErr := ErrRateLimited(retryAfter)
				requestID := r.Header.Get("X-Request-ID")
				errorHandler.HandleAPIError(w, r, apiErr, requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := indexOf(xff, ","); idx != -1 {
			return xff[:idx]
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func indexOf(s string, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// SlidingWindowRateLimiter implements sliding window rate limiting
// for more accurate rate limiting at window boundaries
type SlidingWindowRateLimiter struct {
	windowSize   time.Duration
	maxRequests  int
	windows      map[string]*slidingWindow
	mu           sync.RWMutex
}

type slidingWindow struct {
	counts    []int
	timestamps []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowRateLimiter creates a sliding window rate limiter
func NewSlidingWindowRateLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		windows:     make(map[string]*slidingWindow),
	}
}

// Allow checks if a request from the given key should be allowed
func (sw *SlidingWindowRateLimiter) Allow(key string) bool {
	sw.mu.RLock()
	window, ok := sw.windows[key]
	sw.mu.RUnlock()

	if !ok {
		sw.mu.Lock()
		if window, ok = sw.windows[key]; !ok {
			window = &slidingWindow{
				counts:     make([]int, 0),
				timestamps: make([]time.Time, 0),
			}
			sw.windows[key] = window
		}
		sw.mu.Unlock()
	}

	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	// Remove expired entries
	validIdx := 0
	for i, ts := range window.timestamps {
		if ts.After(cutoff) {
			validIdx = i
			break
		}
	}
	if validIdx > 0 {
		window.counts = window.counts[validIdx:]
		window.timestamps = window.timestamps[validIdx:]
	}

	// Count current requests
	total := 0
	for _, c := range window.counts {
		total += c
	}

	if total >= sw.maxRequests {
		return false
	}

	// Add this request
	window.counts = append(window.counts, 1)
	window.timestamps = append(window.timestamps, now)
	return true
}
