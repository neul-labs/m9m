package reliability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Retry Policy Tests
// ---------------------------------------------------------------------------

func TestRetryPolicy_SuccessOnFirstAttempt(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	callCount := 0
	err := rp.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "function should be called exactly once")
}

func TestRetryPolicy_SuccessAfterRetries(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      5,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	callCount := 0
	err := rp.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return fmt.Errorf("timeout occurred")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "function should be called 3 times (2 failures + 1 success)")
}

func TestRetryPolicy_AllRetriesExhausted(t *testing.T) {
	maxRetries := 3
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      maxRetries,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	callCount := 0
	err := rp.Execute(context.Background(), func(ctx context.Context) error {
		callCount++
		return fmt.Errorf("timeout error")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Contains(t, err.Error(), "timeout error")
	// Initial attempt + MaxRetries retries = MaxRetries + 1 total calls
	assert.Equal(t, maxRetries+1, callCount)
}

func TestRetryPolicy_ContextCancellation(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      10,
		InitialDelay:    50 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   1.0,
		RetryableErrors: []string{"timeout"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := rp.Execute(ctx, func(ctx context.Context) error {
		callCount++
		return fmt.Errorf("timeout error")
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	// Should have been called at least once, but not all retries
	assert.GreaterOrEqual(t, callCount, 1)
	assert.Less(t, callCount, 11)
}

func TestRetryPolicy_DefaultConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.True(t, config.Jitter)
	assert.Contains(t, config.RetryableErrors, "connection refused")
	assert.Contains(t, config.RetryableErrors, "timeout")
	assert.Contains(t, config.RetryableErrors, "temporary failure")
	assert.Contains(t, config.RetryableErrors, "rate limit")
	assert.Contains(t, config.RetryableErrors, "503")
	assert.Contains(t, config.RetryableErrors, "502")
	assert.Contains(t, config.RetryableErrors, "429")
}

func TestRetryPolicy_NilConfigUsesDefault(t *testing.T) {
	rp := NewRetryPolicy(nil)
	assert.NotNil(t, rp)
}

func TestRetryPolicy_RetryableErrors(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout", "connection refused"},
	})

	tests := []struct {
		name        string
		errMsg      string
		shouldRetry bool
	}{
		{
			name:        "retryable timeout error",
			errMsg:      "request timeout",
			shouldRetry: true,
		},
		{
			name:        "retryable connection refused error",
			errMsg:      "connection refused by server",
			shouldRetry: true,
		},
		{
			name:        "non-retryable error",
			errMsg:      "invalid input data",
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			err := rp.Execute(context.Background(), func(ctx context.Context) error {
				callCount++
				return fmt.Errorf("%s", tt.errMsg)
			})

			require.Error(t, err)
			if tt.shouldRetry {
				// Should have been called MaxRetries+1 times
				assert.Equal(t, 4, callCount, "retryable error should exhaust all retries")
				assert.Contains(t, err.Error(), "max retries exceeded")
			} else {
				// Non-retryable should fail immediately
				assert.Equal(t, 1, callCount, "non-retryable error should not be retried")
			}
		})
	}
}

func TestExecuteWithResult_SuccessOnFirstAttempt(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	callCount := 0
	result, err := ExecuteWithResult(context.Background(), rp, func(ctx context.Context) (string, error) {
		callCount++
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount)
}

func TestExecuteWithResult_SuccessAfterRetries(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      5,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	callCount := 0
	result, err := ExecuteWithResult(context.Background(), rp, func(ctx context.Context) (int, error) {
		callCount++
		if callCount < 3 {
			return 0, fmt.Errorf("timeout error")
		}
		return 42, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, callCount)
}

func TestExecuteWithResult_AllRetriesExhausted(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      2,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	result, err := ExecuteWithResult(context.Background(), rp, func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("timeout error")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Equal(t, "", result)
}

func TestExecuteWithResult_NonRetryableError(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Millisecond,
		MaxDelay:        10 * time.Millisecond,
		BackoffFactor:   2.0,
		RetryableErrors: []string{"timeout"},
	})

	callCount := 0
	result, err := ExecuteWithResult(context.Background(), rp, func(ctx context.Context) (string, error) {
		callCount++
		return "", fmt.Errorf("invalid input")
	})

	require.Error(t, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount, "non-retryable error should not be retried")
}

func TestExecuteWithResult_ContextCancellation(t *testing.T) {
	rp := NewRetryPolicy(&RetryConfig{
		MaxRetries:      10,
		InitialDelay:    50 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		BackoffFactor:   1.0,
		RetryableErrors: []string{"timeout"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := ExecuteWithResult(ctx, rp, func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("timeout error")
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// Circuit Breaker Tests
// ---------------------------------------------------------------------------

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   5,
		SuccessThreshold:   3,
		Timeout:            50 * time.Millisecond,
		MaxConcurrentCalls: 1,
	})

	// Successful calls keep the circuit closed
	for i := 0; i < 10; i++ {
		err := cb.Execute(context.Background(), func(ctx context.Context) error {
			return nil
		})
		require.NoError(t, err)
	}

	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_OpensOnFailures(t *testing.T) {
	threshold := 3
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   threshold,
		SuccessThreshold:   1,
		Timeout:            1 * time.Second,
		MaxConcurrentCalls: 1,
	})

	// Cause failures up to threshold
	for i := 0; i < threshold; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure %d", i)
		})
	}

	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_RejectsWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   2,
		SuccessThreshold:   1,
		Timeout:            1 * time.Second,
		MaxConcurrentCalls: 1,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure")
		})
	}
	require.Equal(t, StateOpen, cb.State())

	// Verify requests are rejected
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		t.Fatal("function should not be called when circuit is open")
		return nil
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCircuitOpen)
}

func TestCircuitBreaker_HalfOpenTransition(t *testing.T) {
	timeout := 20 * time.Millisecond
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   2,
		SuccessThreshold:   1,
		Timeout:            timeout,
		MaxConcurrentCalls: 1,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure")
		})
	}
	require.Equal(t, StateOpen, cb.State())

	// Wait for timeout to elapse
	time.Sleep(timeout + 10*time.Millisecond)

	// The next call should transition to half-open and be allowed through
	called := false
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		called = true
		return fmt.Errorf("still failing")
	})

	assert.True(t, called, "function should be called after timeout in half-open state")
	require.Error(t, err)
	assert.Equal(t, "still failing", err.Error())
	// After failure in half-open, should go back to open
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	timeout := 20 * time.Millisecond
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   2,
		SuccessThreshold:   1,
		Timeout:            timeout,
		MaxConcurrentCalls: 1,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure")
		})
	}
	require.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(timeout + 10*time.Millisecond)

	// Succeed in half-open state
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)

	// Circuit should close after success threshold reached
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_ClosesAfterMultipleSuccessesInHalfOpen(t *testing.T) {
	timeout := 20 * time.Millisecond
	successThreshold := 3
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   2,
		SuccessThreshold:   successThreshold,
		Timeout:            timeout,
		MaxConcurrentCalls: 10, // allow multiple calls in half-open
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure")
		})
	}
	require.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(timeout + 10*time.Millisecond)

	// First call transitions to half-open
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	// With successThreshold=3, one success isn't enough; still half-open
	assert.Equal(t, StateHalfOpen, cb.State())

	// Second success
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())

	// Third success should close the circuit
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   2,
		SuccessThreshold:   1,
		Timeout:            1 * time.Second,
		MaxConcurrentCalls: 1,
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure")
		})
	}
	require.Equal(t, StateOpen, cb.State())

	// Reset
	cb.Reset()

	assert.Equal(t, StateClosed, cb.State())

	// Verify it works again
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	require.NoError(t, err)
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   5,
		SuccessThreshold:   1,
		Timeout:            1 * time.Second,
		MaxConcurrentCalls: 1,
	})

	// Cause some successes and failures
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return fmt.Errorf("timeout error")
	})
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return fmt.Errorf("timeout error")
	})

	stats := cb.Stats()

	assert.Equal(t, "closed", stats["state"])
	assert.Equal(t, 2, stats["failureCount"])
	// After failures, successCount is reset to 0
	assert.Equal(t, 0, stats["successCount"])
	assert.NotZero(t, stats["lastFailureTime"])
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 3, config.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 1, config.MaxConcurrentCalls)
}

func TestCircuitBreaker_NilConfigUsesDefault(t *testing.T) {
	cb := NewCircuitBreaker(nil)
	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_OnStateChangeCallback(t *testing.T) {
	var transitions []string
	var mu sync.Mutex
	transitionDone := make(chan struct{}, 5)

	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:   2,
		SuccessThreshold:   1,
		Timeout:            20 * time.Millisecond,
		MaxConcurrentCalls: 1,
		OnStateChange: func(from, to CircuitBreakerState) {
			mu.Lock()
			transitions = append(transitions, fmt.Sprintf("%s->%s", from, to))
			mu.Unlock()
			transitionDone <- struct{}{}
		},
	})

	// Trigger failures to open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return fmt.Errorf("failure")
		})
	}

	// Wait for callback (it runs in a goroutine)
	select {
	case <-transitionDone:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for state change callback")
	}

	mu.Lock()
	require.Len(t, transitions, 1)
	assert.Equal(t, "closed->open", transitions[0])
	mu.Unlock()
}

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitBreakerState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Bulkhead Tests
// ---------------------------------------------------------------------------

func TestBulkhead_AllowsConcurrent(t *testing.T) {
	maxConcurrent := 3
	bh := NewBulkhead(&BulkheadConfig{
		MaxConcurrent: maxConcurrent,
		MaxWaiting:    10,
		Timeout:       1 * time.Second,
	})

	var running int32
	var maxRunning int32
	var wg sync.WaitGroup
	ready := make(chan struct{})

	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := bh.Execute(context.Background(), func(ctx context.Context) error {
				cur := atomic.AddInt32(&running, 1)
				// Track max concurrency
				for {
					old := atomic.LoadInt32(&maxRunning)
					if cur <= old || atomic.CompareAndSwapInt32(&maxRunning, old, cur) {
						break
					}
				}
				<-ready
				atomic.AddInt32(&running, -1)
				return nil
			})
			assert.NoError(t, err)
		}()
	}

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(maxConcurrent), atomic.LoadInt32(&running))
	close(ready)
	wg.Wait()

	assert.LessOrEqual(t, atomic.LoadInt32(&maxRunning), int32(maxConcurrent))
}

func TestBulkhead_RejectsOverCapacity(t *testing.T) {
	bh := NewBulkhead(&BulkheadConfig{
		MaxConcurrent: 1,
		MaxWaiting:    2, // Allow some into the waiting queue
		Timeout:       50 * time.Millisecond,
	})

	blocker := make(chan struct{})
	started := make(chan struct{})

	// Fill the single concurrent slot
	go func() {
		_ = bh.Execute(context.Background(), func(ctx context.Context) error {
			close(started)
			<-blocker
			return nil
		})
	}()

	<-started
	// Give it a moment to ensure the goroutine is occupying the slot
	time.Sleep(10 * time.Millisecond)

	// Launch 2 more goroutines to fill the waiting queue
	waitErrCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			waitErrCh <- bh.Execute(context.Background(), func(ctx context.Context) error {
				return nil
			})
		}()
	}

	// Give them time to enter the waiting queue
	time.Sleep(20 * time.Millisecond)

	// This should fail because MaxWaiting is full
	err := bh.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBulkheadFull)

	close(blocker)

	// Drain waitErrCh to prevent goroutine leak
	for i := 0; i < 2; i++ {
		<-waitErrCh
	}
}

func TestBulkhead_Stats(t *testing.T) {
	bh := NewBulkhead(&BulkheadConfig{
		MaxConcurrent: 5,
		MaxWaiting:    10,
		Timeout:       1 * time.Second,
	})

	stats := bh.Stats()
	assert.Equal(t, 5, stats["maxConcurrent"])
	assert.Equal(t, 0, stats["activeCalls"])
	assert.Equal(t, 0, stats["waiting"])
}

func TestBulkhead_NilConfigUsesDefault(t *testing.T) {
	bh := NewBulkhead(nil)
	assert.NotNil(t, bh)

	stats := bh.Stats()
	assert.Equal(t, 10, stats["maxConcurrent"])
}

func TestBulkhead_ContextCancellation(t *testing.T) {
	bh := NewBulkhead(&BulkheadConfig{
		MaxConcurrent: 1,
		MaxWaiting:    5,
		Timeout:       5 * time.Second,
	})

	blocker := make(chan struct{})
	started := make(chan struct{})

	// Fill the slot
	go func() {
		_ = bh.Execute(context.Background(), func(ctx context.Context) error {
			close(started)
			<-blocker
			return nil
		})
	}()

	<-started
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- bh.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for context cancellation")
	}

	close(blocker)
}

// ---------------------------------------------------------------------------
// Dead Letter Queue Tests
// ---------------------------------------------------------------------------

func TestDLQ_AddAndGet(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	item := &DeadLetterItem{
		ID:          "test-1",
		WorkflowID:  "wf-1",
		ExecutionID: "exec-1",
		Error:       "something went wrong",
		ErrorType:   "RuntimeError",
		MaxRetries:  3,
	}

	err := dlq.Add(item)
	require.NoError(t, err)

	retrieved, ok := dlq.Get("test-1")
	require.True(t, ok)
	assert.Equal(t, "test-1", retrieved.ID)
	assert.Equal(t, "wf-1", retrieved.WorkflowID)
	assert.Equal(t, "exec-1", retrieved.ExecutionID)
	assert.Equal(t, "something went wrong", retrieved.Error)
	assert.Equal(t, "RuntimeError", retrieved.ErrorType)
	assert.Equal(t, DLQStatusPending, retrieved.Status)
	assert.False(t, retrieved.FirstFailedAt.IsZero())
	assert.False(t, retrieved.LastFailedAt.IsZero())
}

func TestDLQ_AddDefaultID(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	item := &DeadLetterItem{
		WorkflowID: "wf-1",
		Error:      "something failed",
	}

	err := dlq.Add(item)
	require.NoError(t, err)
	assert.NotEmpty(t, item.ID)
	assert.Contains(t, item.ID, "dlq-")
}

func TestDLQ_GetNonExistent(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	_, ok := dlq.Get("nonexistent")
	assert.False(t, ok)
}

func TestDLQ_List(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	// Add items with different workflow IDs and error types
	items := []*DeadLetterItem{
		{ID: "item-1", WorkflowID: "wf-1", ErrorType: "RuntimeError", Error: "err1"},
		{ID: "item-2", WorkflowID: "wf-1", ErrorType: "TimeoutError", Error: "err2"},
		{ID: "item-3", WorkflowID: "wf-2", ErrorType: "RuntimeError", Error: "err3"},
		{ID: "item-4", WorkflowID: "wf-2", ErrorType: "ValidationError", Error: "err4"},
	}

	for _, item := range items {
		err := dlq.Add(item)
		require.NoError(t, err)
	}

	// List all items (nil filter)
	allItems := dlq.List(nil)
	assert.Len(t, allItems, 4)

	// Filter by WorkflowID
	wf1Items := dlq.List(&DLQFilter{WorkflowID: "wf-1"})
	assert.Len(t, wf1Items, 2)

	// Filter by ErrorType
	runtimeItems := dlq.List(&DLQFilter{ErrorType: "RuntimeError"})
	assert.Len(t, runtimeItems, 2)

	// Combined filter
	combined := dlq.List(&DLQFilter{WorkflowID: "wf-1", ErrorType: "RuntimeError"})
	assert.Len(t, combined, 1)
	assert.Equal(t, "item-1", combined[0].ID)
}

func TestDLQ_StatusTransitions(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	item := &DeadLetterItem{
		ID:          "item-1",
		WorkflowID:  "wf-1",
		Error:       "failed",
		ErrorType:   "RuntimeError",
		MaxRetries:  3,
	}
	err := dlq.Add(item)
	require.NoError(t, err)

	// Verify initial status
	retrieved, _ := dlq.Get("item-1")
	assert.Equal(t, DLQStatusPending, retrieved.Status)

	// Transition to retrying
	err = dlq.Retry("item-1")
	require.NoError(t, err)
	retrieved, _ = dlq.Get("item-1")
	assert.Equal(t, DLQStatusRetrying, retrieved.Status)
	assert.Equal(t, 1, retrieved.RetryCount)

	// Put on manual hold
	err = dlq.Hold("item-1")
	require.NoError(t, err)
	retrieved, _ = dlq.Get("item-1")
	assert.Equal(t, DLQStatusManualHold, retrieved.Status)

	// Resolve
	err = dlq.Resolve("item-1")
	require.NoError(t, err)
	retrieved, _ = dlq.Get("item-1")
	assert.Equal(t, DLQStatusResolved, retrieved.Status)

	// Test with second item for discard
	item2 := &DeadLetterItem{
		ID:         "item-2",
		WorkflowID: "wf-1",
		Error:      "another error",
		ErrorType:  "ValidationError",
	}
	err = dlq.Add(item2)
	require.NoError(t, err)

	err = dlq.Discard("item-2")
	require.NoError(t, err)
	retrieved, _ = dlq.Get("item-2")
	assert.Equal(t, DLQStatusDiscarded, retrieved.Status)
}

func TestDLQ_StatusTransitions_NotFound(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	tests := []struct {
		name string
		fn   func(id string) error
	}{
		{"Retry", dlq.Retry},
		{"Resolve", dlq.Resolve},
		{"Discard", dlq.Discard},
		{"Hold", dlq.Hold},
		{"Delete", dlq.Delete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("nonexistent")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "item not found")
		})
	}
}

func TestDLQ_Delete(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	item := &DeadLetterItem{
		ID:         "to-delete",
		WorkflowID: "wf-1",
		Error:      "error",
		ErrorType:  "RuntimeError",
	}
	err := dlq.Add(item)
	require.NoError(t, err)

	// Verify it exists
	_, ok := dlq.Get("to-delete")
	require.True(t, ok)

	// Delete it
	err = dlq.Delete("to-delete")
	require.NoError(t, err)

	// Verify it's gone
	_, ok = dlq.Get("to-delete")
	assert.False(t, ok)

	// Verify list is empty
	all := dlq.List(nil)
	assert.Len(t, all, 0)
}

func TestDLQ_Stats(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         1000,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	// Add items with different statuses
	items := []*DeadLetterItem{
		{ID: "item-1", WorkflowID: "wf-1", Error: "err1", ErrorType: "RuntimeError"},
		{ID: "item-2", WorkflowID: "wf-1", Error: "err2", ErrorType: "RuntimeError"},
		{ID: "item-3", WorkflowID: "wf-2", Error: "err3", ErrorType: "TimeoutError"},
	}
	for _, item := range items {
		err := dlq.Add(item)
		require.NoError(t, err)
	}

	// Transition some items
	_ = dlq.Resolve("item-1")
	_ = dlq.Discard("item-2")

	stats := dlq.Stats()
	assert.Equal(t, 3, stats["totalItems"])
	assert.Equal(t, 1000, stats["maxSize"])

	statusCounts := stats["statusCounts"].(map[string]int)
	assert.Equal(t, 1, statusCounts["pending"])
	assert.Equal(t, 1, statusCounts["resolved"])
	assert.Equal(t, 1, statusCounts["discarded"])
}

func TestDLQ_ExportImport(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	now := time.Now().Truncate(time.Second) // Truncate for JSON round-trip precision

	// Add items
	items := []*DeadLetterItem{
		{
			ID:            "exp-1",
			WorkflowID:    "wf-1",
			ExecutionID:   "exec-1",
			Error:         "error 1",
			ErrorType:     "RuntimeError",
			RetryCount:    1,
			MaxRetries:    3,
			FirstFailedAt: now,
			LastFailedAt:  now,
			Status:        DLQStatusPending,
		},
		{
			ID:            "exp-2",
			WorkflowID:    "wf-2",
			ExecutionID:   "exec-2",
			Error:         "error 2",
			ErrorType:     "TimeoutError",
			RetryCount:    0,
			MaxRetries:    5,
			FirstFailedAt: now,
			LastFailedAt:  now,
			Status:        DLQStatusRetrying,
		},
	}
	for _, item := range items {
		err := dlq.Add(item)
		require.NoError(t, err)
	}

	// Export
	data, err := dlq.Export()
	require.NoError(t, err)

	// Verify JSON is valid
	var exported []*DeadLetterItem
	err = json.Unmarshal(data, &exported)
	require.NoError(t, err)
	assert.Len(t, exported, 2)

	// Import into a new DLQ
	dlq2 := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq2.Stop()

	err = dlq2.Import(data)
	require.NoError(t, err)

	// Verify imported items
	item1, ok := dlq2.Get("exp-1")
	require.True(t, ok)
	assert.Equal(t, "wf-1", item1.WorkflowID)
	assert.Equal(t, "error 1", item1.Error)
	assert.Equal(t, DLQStatusPending, item1.Status)

	item2, ok := dlq2.Get("exp-2")
	require.True(t, ok)
	assert.Equal(t, "wf-2", item2.WorkflowID)
	assert.Equal(t, "error 2", item2.Error)
	assert.Equal(t, DLQStatusRetrying, item2.Status)

	// Verify list preserves order
	allItems := dlq2.List(nil)
	assert.Len(t, allItems, 2)
	assert.Equal(t, "exp-1", allItems[0].ID)
	assert.Equal(t, "exp-2", allItems[1].ID)
}

func TestDLQ_ImportInvalidJSON(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	err := dlq.Import([]byte("not valid json"))
	require.Error(t, err)
}

func TestDLQ_MaxSize(t *testing.T) {
	maxSize := 3
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         maxSize,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	// Fill to capacity
	for i := 0; i < maxSize; i++ {
		err := dlq.Add(&DeadLetterItem{
			ID:    fmt.Sprintf("item-%d", i),
			Error: fmt.Sprintf("error %d", i),
		})
		require.NoError(t, err)
	}

	// Verify at capacity
	allItems := dlq.List(nil)
	assert.Len(t, allItems, maxSize)

	// Adding one more should evict the oldest
	err := dlq.Add(&DeadLetterItem{
		ID:    "item-new",
		Error: "new error",
	})
	require.NoError(t, err)

	allItems = dlq.List(nil)
	assert.Len(t, allItems, maxSize)

	// The first item should have been evicted
	_, ok := dlq.Get("item-0")
	assert.False(t, ok, "oldest item should be evicted")

	// The new item should exist
	_, ok = dlq.Get("item-new")
	assert.True(t, ok)
}

func TestDLQ_Callbacks(t *testing.T) {
	var addedItems []*DeadLetterItem
	var resolvedItems []*DeadLetterItem
	var mu sync.Mutex
	addedCh := make(chan struct{}, 5)
	resolvedCh := make(chan struct{}, 5)

	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
		OnItemAdded: func(item *DeadLetterItem) {
			mu.Lock()
			addedItems = append(addedItems, item)
			mu.Unlock()
			addedCh <- struct{}{}
		},
		OnItemResolved: func(item *DeadLetterItem) {
			mu.Lock()
			resolvedItems = append(resolvedItems, item)
			mu.Unlock()
			resolvedCh <- struct{}{}
		},
	})
	defer dlq.Stop()

	// Add an item
	item := &DeadLetterItem{
		ID:         "cb-1",
		WorkflowID: "wf-1",
		Error:      "callback test",
		ErrorType:  "RuntimeError",
	}
	err := dlq.Add(item)
	require.NoError(t, err)

	// Wait for added callback (runs in goroutine)
	select {
	case <-addedCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for OnItemAdded callback")
	}

	mu.Lock()
	require.Len(t, addedItems, 1)
	assert.Equal(t, "cb-1", addedItems[0].ID)
	mu.Unlock()

	// Resolve the item
	err = dlq.Resolve("cb-1")
	require.NoError(t, err)

	// Wait for resolved callback
	select {
	case <-resolvedCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for OnItemResolved callback")
	}

	mu.Lock()
	require.Len(t, resolvedItems, 1)
	assert.Equal(t, "cb-1", resolvedItems[0].ID)
	mu.Unlock()
}

func TestDLQ_GetPendingForRetry(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	items := []*DeadLetterItem{
		{ID: "item-1", Error: "err1", Status: DLQStatusPending, RetryCount: 0, MaxRetries: 3},
		{ID: "item-2", Error: "err2", Status: DLQStatusPending, RetryCount: 3, MaxRetries: 3}, // exhausted retries
		{ID: "item-3", Error: "err3", ErrorType: "RuntimeError"},                                // Will get default pending status
	}
	for _, item := range items {
		err := dlq.Add(item)
		require.NoError(t, err)
	}

	// Resolve one to make it not pending
	_ = dlq.Resolve("item-3")

	pending := dlq.GetPendingForRetry()
	// item-1 is pending with retries remaining
	// item-2 is pending but retries exhausted (RetryCount >= MaxRetries)
	// item-3 is resolved
	assert.Len(t, pending, 1)
	assert.Equal(t, "item-1", pending[0].ID)
}

// ---------------------------------------------------------------------------
// DLQ Filter Tests
// ---------------------------------------------------------------------------

func TestDLQFilter_Matches(t *testing.T) {
	now := time.Now()
	item := &DeadLetterItem{
		ID:           "filter-test",
		WorkflowID:   "wf-123",
		Error:        "test error",
		ErrorType:    "TimeoutError",
		Status:       DLQStatusPending,
		LastFailedAt: now,
	}

	tests := []struct {
		name     string
		filter   *DLQFilter
		expected bool
	}{
		{
			name:     "empty filter matches everything",
			filter:   &DLQFilter{},
			expected: true,
		},
		{
			name:     "matching WorkflowID",
			filter:   &DLQFilter{WorkflowID: "wf-123"},
			expected: true,
		},
		{
			name:     "non-matching WorkflowID",
			filter:   &DLQFilter{WorkflowID: "wf-999"},
			expected: false,
		},
		{
			name:     "matching Status",
			filter:   &DLQFilter{Status: DLQStatusPending},
			expected: true,
		},
		{
			name:     "non-matching Status",
			filter:   &DLQFilter{Status: DLQStatusResolved},
			expected: false,
		},
		{
			name:     "matching ErrorType",
			filter:   &DLQFilter{ErrorType: "TimeoutError"},
			expected: true,
		},
		{
			name:     "non-matching ErrorType",
			filter:   &DLQFilter{ErrorType: "ValidationError"},
			expected: false,
		},
		{
			name:     "Since before LastFailedAt matches",
			filter:   &DLQFilter{Since: now.Add(-1 * time.Hour)},
			expected: true,
		},
		{
			name:     "Since after LastFailedAt does not match",
			filter:   &DLQFilter{Since: now.Add(1 * time.Hour)},
			expected: false,
		},
		{
			name: "all filters matching",
			filter: &DLQFilter{
				WorkflowID: "wf-123",
				Status:     DLQStatusPending,
				ErrorType:  "TimeoutError",
				Since:      now.Add(-1 * time.Minute),
			},
			expected: true,
		},
		{
			name: "one filter not matching fails whole match",
			filter: &DLQFilter{
				WorkflowID: "wf-123",
				Status:     DLQStatusResolved, // does not match
				ErrorType:  "TimeoutError",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Health Checker Tests
// ---------------------------------------------------------------------------

func TestHealthChecker_RegisterAndCheck(t *testing.T) {
	hc := NewHealthChecker(1 * time.Hour) // Long interval so background loop doesn't interfere
	defer hc.Stop()

	hc.Register("test-service", func(ctx context.Context) error {
		return nil
	})

	hc.CheckNow()

	status := hc.Status()
	require.Contains(t, status, "test-service")
	assert.True(t, status["test-service"].Healthy)
	assert.Equal(t, "", status["test-service"].LastError)
	assert.Equal(t, 0, status["test-service"].Consecutive)
	assert.False(t, status["test-service"].LastCheck.IsZero())
}

func TestHealthChecker_UnhealthyCheck(t *testing.T) {
	hc := NewHealthChecker(1 * time.Hour)
	defer hc.Stop()

	hc.Register("failing-service", func(ctx context.Context) error {
		return errors.New("service unavailable")
	})

	hc.CheckNow()

	status := hc.Status()
	require.Contains(t, status, "failing-service")
	assert.False(t, status["failing-service"].Healthy)
	assert.Equal(t, "service unavailable", status["failing-service"].LastError)
	assert.Equal(t, 1, status["failing-service"].Consecutive)

	// Check again to verify consecutive counter increments
	hc.CheckNow()
	status = hc.Status()
	assert.Equal(t, 2, status["failing-service"].Consecutive)
}

func TestHealthChecker_IsHealthy(t *testing.T) {
	hc := NewHealthChecker(1 * time.Hour)
	defer hc.Stop()

	// No checks registered - should be healthy
	assert.True(t, hc.IsHealthy())

	// Register a healthy check
	hc.Register("service-a", func(ctx context.Context) error {
		return nil
	})
	hc.Register("service-b", func(ctx context.Context) error {
		return nil
	})

	// Before CheckNow, all are initialized as healthy
	assert.True(t, hc.IsHealthy())

	hc.CheckNow()
	assert.True(t, hc.IsHealthy())

	// Register a failing check
	hc.Register("service-c", func(ctx context.Context) error {
		return errors.New("down")
	})

	hc.CheckNow()
	assert.False(t, hc.IsHealthy(), "should be unhealthy when one check fails")
}

func TestHealthChecker_Unregister(t *testing.T) {
	hc := NewHealthChecker(1 * time.Hour)
	defer hc.Stop()

	hc.Register("temporary", func(ctx context.Context) error {
		return errors.New("failing")
	})

	hc.CheckNow()
	assert.False(t, hc.IsHealthy())

	// Unregister the failing check
	hc.Unregister("temporary")

	status := hc.Status()
	assert.NotContains(t, status, "temporary")
	assert.True(t, hc.IsHealthy())
}

func TestHealthChecker_ConsecutiveResetOnSuccess(t *testing.T) {
	hc := NewHealthChecker(1 * time.Hour)
	defer hc.Stop()

	callCount := 0
	hc.Register("flaky", func(ctx context.Context) error {
		callCount++
		if callCount <= 2 {
			return errors.New("temporarily down")
		}
		return nil
	})

	// First two checks fail
	hc.CheckNow()
	status := hc.Status()
	assert.Equal(t, 1, status["flaky"].Consecutive)

	hc.CheckNow()
	status = hc.Status()
	assert.Equal(t, 2, status["flaky"].Consecutive)

	// Third check succeeds
	hc.CheckNow()
	status = hc.Status()
	assert.True(t, status["flaky"].Healthy)
	assert.Equal(t, 0, status["flaky"].Consecutive)
	assert.Equal(t, "", status["flaky"].LastError)
}

func TestHealthChecker_MultipleChecks(t *testing.T) {
	hc := NewHealthChecker(1 * time.Hour)
	defer hc.Stop()

	hc.Register("db", func(ctx context.Context) error { return nil })
	hc.Register("cache", func(ctx context.Context) error { return nil })
	hc.Register("queue", func(ctx context.Context) error { return errors.New("queue down") })

	hc.CheckNow()

	status := hc.Status()
	require.Len(t, status, 3)
	assert.True(t, status["db"].Healthy)
	assert.True(t, status["cache"].Healthy)
	assert.False(t, status["queue"].Healthy)
}

// ---------------------------------------------------------------------------
// DLQ Default Config Test
// ---------------------------------------------------------------------------

func TestDLQ_DefaultConfig(t *testing.T) {
	config := DefaultDLQConfig()

	assert.Equal(t, 10000, config.MaxSize)
	assert.Equal(t, 7*24*time.Hour, config.RetentionPeriod)
	assert.False(t, config.AutoRetry)
	assert.Equal(t, 5*time.Minute, config.RetryInterval)
	assert.Equal(t, 3, config.MaxAutoRetries)
}

func TestDLQ_NilConfigUsesDefault(t *testing.T) {
	dlq := NewDeadLetterQueue(nil)
	defer dlq.Stop()
	assert.NotNil(t, dlq)

	stats := dlq.Stats()
	assert.Equal(t, 10000, stats["maxSize"])
}

// ---------------------------------------------------------------------------
// DLQ List with filter by status
// ---------------------------------------------------------------------------

func TestDLQ_ListFilterByStatus(t *testing.T) {
	dlq := NewDeadLetterQueue(&DLQConfig{
		MaxSize:         100,
		RetentionPeriod: time.Hour,
	})
	defer dlq.Stop()

	items := []*DeadLetterItem{
		{ID: "s-1", Error: "err1", ErrorType: "RuntimeError"},
		{ID: "s-2", Error: "err2", ErrorType: "RuntimeError"},
		{ID: "s-3", Error: "err3", ErrorType: "RuntimeError"},
	}
	for _, item := range items {
		err := dlq.Add(item)
		require.NoError(t, err)
	}

	_ = dlq.Resolve("s-1")
	_ = dlq.Discard("s-2")
	// s-3 remains pending

	pendingItems := dlq.List(&DLQFilter{Status: DLQStatusPending})
	assert.Len(t, pendingItems, 1)
	assert.Equal(t, "s-3", pendingItems[0].ID)

	resolvedItems := dlq.List(&DLQFilter{Status: DLQStatusResolved})
	assert.Len(t, resolvedItems, 1)
	assert.Equal(t, "s-1", resolvedItems[0].ID)

	discardedItems := dlq.List(&DLQFilter{Status: DLQStatusDiscarded})
	assert.Len(t, discardedItems, 1)
	assert.Equal(t, "s-2", discardedItems[0].ID)
}

// ---------------------------------------------------------------------------
// Sentinel Error Tests
// ---------------------------------------------------------------------------

func TestSentinelErrors(t *testing.T) {
	assert.NotNil(t, ErrCircuitOpen)
	assert.NotNil(t, ErrBulkheadFull)
	assert.Equal(t, "circuit breaker is open", ErrCircuitOpen.Error())
	assert.Equal(t, "bulkhead is full", ErrBulkheadFull.Error())
}
