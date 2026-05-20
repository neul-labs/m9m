// Package quota defines the extension contract for per-workspace usage
// limits enforced inside the engine. The engine asks the Enforcer "may I
// perform this operation?" before spending compute on it — quota enforcement
// happens *before* cycles cost the customer.
//
// quota.Enforcer is closely related to billing.Provider: most embedders
// implement the Enforcer in terms of a Provider (Cloud-mode worker does).
// They're separate interfaces because (a) some embedders want quotas without
// a billing backend, and (b) the engine should not depend on the heavyweight
// billing surface for hot-path checks.
//
// This is part of the public embedding surface and commits to semver.
package quota

import (
	"context"
	"errors"
)

// Decision is the result of a Check call.
type Decision int

const (
	// Allow permits the operation.
	Allow Decision = iota
	// Warn permits the operation but signals the workspace is approaching its
	// quota. Embedders may emit notifications.
	Warn
	// Deny rejects the operation. Engine returns an HTTP 402-style error.
	Deny
)

// Request is the operation being checked.
type Request struct {
	WorkspaceID string
	// Kind names the resource consumed. Recognized values mirror
	// billing.UsageEvent.Kind:
	//   "execution"  — one workflow execution
	//   "node_run"   — one node invocation
	//   "webhook"    — one webhook delivery
	//   "storage_gb" — gigabytes of blob storage (point-in-time)
	Kind     string
	Quantity int64
}

// Enforcer is the extension contract for quota checks.
//
// Implementations are safe for concurrent use. Check is on the hot path and
// must return within a few milliseconds; embedders are expected to maintain
// an in-memory cache of per-workspace usage and tier limits.
type Enforcer interface {
	// Check decides whether to allow the request. On Allow / Warn the engine
	// proceeds; on Deny the engine rejects with an error.
	Check(ctx context.Context, req Request) (Decision, error)

	// Observe records that the operation was actually performed. Called
	// after a successful Check + execution. Failures should not block the
	// caller — the engine treats this as fire-and-forget.
	Observe(ctx context.Context, req Request) error
}

// ErrDenied is returned by helper functions when a Decision is Deny.
var ErrDenied = errors.New("quota: denied")
