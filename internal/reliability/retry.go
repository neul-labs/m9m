// Package reliability provides reliability features for workflow execution
package reliability

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryableErrors []string // Error patterns that should be retried
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []string{
			"connection refused",
			"timeout",
			"temporary failure",
			"rate limit",
			"503",
			"502",
			"429",
		},
	}
}

// RetryPolicy defines a retry policy
type RetryPolicy struct {
	config *RetryConfig
}

// NewRetryPolicy creates a new retry policy
func NewRetryPolicy(config *RetryConfig) *RetryPolicy {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryPolicy{config: config}
}

// RetryFunc is a function that can be retried
type RetryFunc func(ctx context.Context) error

// Execute executes a function with retries
func (rp *RetryPolicy) Execute(ctx context.Context, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt <= rp.config.MaxRetries; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !rp.isRetryable(err) {
			return err
		}

		// Check if we've exhausted retries
		if attempt >= rp.config.MaxRetries {
			break
		}

		// Calculate delay
		delay := rp.calculateDelay(attempt)

		// Wait or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ExecuteWithResult executes a function that returns a result with retries
func ExecuteWithResult[T any](ctx context.Context, rp *RetryPolicy, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= rp.config.MaxRetries; attempt++ {
		res, err := fn(ctx)
		if err == nil {
			return res, nil
		}

		lastErr = err

		if !rp.isRetryable(err) {
			return result, err
		}

		if attempt >= rp.config.MaxRetries {
			break
		}

		delay := rp.calculateDelay(attempt)

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (rp *RetryPolicy) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	for _, pattern := range rp.config.RetryableErrors {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func (rp *RetryPolicy) calculateDelay(attempt int) time.Duration {
	delay := float64(rp.config.InitialDelay) * math.Pow(rp.config.BackoffFactor, float64(attempt))

	if delay > float64(rp.config.MaxDelay) {
		delay = float64(rp.config.MaxDelay)
	}

	if rp.config.Jitter {
		// Add up to 25% jitter
		jitter := delay * 0.25 * rand.Float64()
		delay += jitter
	}

	return time.Duration(delay)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CircuitBreakerState represents the circuit breaker state
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold    int           // Failures before opening
	SuccessThreshold    int           // Successes in half-open before closing
	Timeout             time.Duration // Time before trying half-open
	MaxConcurrentCalls  int           // Max concurrent calls in half-open
	OnStateChange       func(from, to CircuitBreakerState)
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:   5,
		SuccessThreshold:   3,
		Timeout:            30 * time.Second,
		MaxConcurrentCalls: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config           *CircuitBreakerConfig
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	halfOpenCalls    int
	mu               sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// ErrCircuitOpen is returned when the circuit is open
var ErrCircuitOpen = fmt.Errorf("circuit breaker is open")

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := fn(ctx)
	cb.recordResult(err)
	return err
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":           cb.state.String(),
		"failureCount":    cb.failureCount,
		"successCount":    cb.successCount,
		"lastFailureTime": cb.lastFailureTime,
	}
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenCalls = 1
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited concurrent calls
		if cb.halfOpenCalls < cb.config.MaxConcurrentCalls {
			cb.halfOpenCalls++
			return true
		}
		return false
	}

	return false
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.successCount = 0
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) transitionTo(state CircuitBreakerState) {
	if cb.config.OnStateChange != nil && cb.state != state {
		go cb.config.OnStateChange(cb.state, state)
	}

	cb.state = state
	if state == StateClosed {
		cb.failureCount = 0
		cb.successCount = 0
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCalls = 0
}

// BulkheadConfig configures the bulkhead
type BulkheadConfig struct {
	MaxConcurrent int
	MaxWaiting    int
	Timeout       time.Duration
}

// Bulkhead implements the bulkhead pattern for isolation
type Bulkhead struct {
	config    *BulkheadConfig
	semaphore chan struct{}
	waiting   int
	mu        sync.Mutex
}

// NewBulkhead creates a new bulkhead
func NewBulkhead(config *BulkheadConfig) *Bulkhead {
	if config == nil {
		config = &BulkheadConfig{
			MaxConcurrent: 10,
			MaxWaiting:    100,
			Timeout:       30 * time.Second,
		}
	}

	return &Bulkhead{
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrent),
	}
}

// ErrBulkheadFull is returned when the bulkhead is full
var ErrBulkheadFull = fmt.Errorf("bulkhead is full")

// Execute executes a function through the bulkhead
func (b *Bulkhead) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	// Check waiting queue
	b.mu.Lock()
	if b.waiting >= b.config.MaxWaiting {
		b.mu.Unlock()
		return ErrBulkheadFull
	}
	b.waiting++
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.waiting--
		b.mu.Unlock()
	}()

	// Try to acquire semaphore
	select {
	case b.semaphore <- struct{}{}:
		defer func() { <-b.semaphore }()
		return fn(ctx)
	case <-time.After(b.config.Timeout):
		return fmt.Errorf("bulkhead timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns bulkhead statistics
func (b *Bulkhead) Stats() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	return map[string]interface{}{
		"maxConcurrent": b.config.MaxConcurrent,
		"activeCalls":   len(b.semaphore),
		"waiting":       b.waiting,
	}
}
