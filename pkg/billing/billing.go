// Package billing defines the extension contract for plugging billing,
// usage metering, and tier enforcement into m9m.
//
// m9m OSS does not ship a billing implementation that talks to a payment
// processor. Embedders (such as m9m Cloud) provide a concrete Provider that
// integrates with Stripe, Paddle, or any other backend. Self-hosters typically
// run the NullProvider (unlimited) or ConfigProvider (YAML-driven static
// quotas) shipped here.
//
// This is part of the public embedding surface and commits to semver. New
// methods on Provider are additive only; breaking changes require a v2 of
// this package.
package billing

import (
	"context"
	"errors"
)

// Tier names a plan a workspace is on. Higher tiers typically have larger
// quotas and shorter hibernation windows. Tier values are opaque strings —
// embedders define their own scheme (e.g. "free", "indie", "pro", "team").
type Tier string

// CommonTiers lists the conventional tier strings used by m9m Cloud.
// Embedders are free to use their own; these are recommendations only.
const (
	TierFree       Tier = "free"
	TierIndie      Tier = "indie"
	TierPro        Tier = "pro"
	TierTeam       Tier = "team"
	TierEnterprise Tier = "enterprise"
)

// UsageEvent is a single reportable unit of usage attributed to a workspace.
// Concrete Providers translate this into provider-specific metering records
// (e.g. Stripe usage records, Paddle metric events).
type UsageEvent struct {
	WorkspaceID string
	// Kind identifies the resource consumed. Recognized values:
	//   "execution"  — one workflow execution
	//   "node_run"   — one node invocation (sub-execution metric)
	//   "webhook"    — one webhook delivery
	//   "storage_gb" — gigabytes of blob storage (point-in-time)
	Kind     string
	Quantity int64
	// Metadata carries provider-specific hints (workflow_id, region, etc.).
	// Optional.
	Metadata map[string]string
}

// QuotaDecision is returned by CheckQuota and tells the engine whether to
// allow, deny, or warn about the requested operation.
type QuotaDecision int

const (
	// QuotaAllow permits the operation.
	QuotaAllow QuotaDecision = iota
	// QuotaWarn permits the operation but signals the customer is approaching
	// their quota. Embedders may emit notifications.
	QuotaWarn
	// QuotaDeny rejects the operation. The engine should return a 402-style
	// error to the caller.
	QuotaDeny
)

// QuotaRequest describes the operation about to be performed so the Provider
// can decide whether to allow it.
type QuotaRequest struct {
	WorkspaceID string
	Kind        string // matches UsageEvent.Kind
	Quantity    int64
}

// Subscription is a thin view of a workspace's billing state, exposed for
// engine logic that needs the tier or seat count.
type Subscription struct {
	WorkspaceID string
	Tier        Tier
	Seats       int
	Active      bool
	// PeriodEnd is when the current billing period ends. May be zero for
	// providers that don't expose this.
	PeriodEnd int64 // unix seconds; 0 = unknown
}

// Provider is the extension contract for billing integrations.
//
// All methods are expected to be safe for concurrent use. Implementations may
// cache aggressively; embedders are responsible for invalidation via webhook
// notifications from their billing backend.
//
// The engine calls these methods on the hot path; latency budgets are tight.
// In particular, CheckQuota is called before every execution and should
// return within a few milliseconds (use an in-memory cache).
type Provider interface {
	// Subscribe records that a workspace is starting a subscription on a tier.
	// Idempotent. Returns the resulting subscription state.
	Subscribe(ctx context.Context, workspaceID string, tier Tier) (*Subscription, error)

	// Cancel ends a workspace's subscription. The workspace typically falls
	// back to TierFree after the current period ends.
	Cancel(ctx context.Context, workspaceID string) error

	// ReportUsage records a usage event for later aggregation and billing.
	// Implementations may batch reports; failures should not block the
	// caller — the engine treats this as fire-and-forget.
	ReportUsage(ctx context.Context, event UsageEvent) error

	// CheckQuota decides whether to allow an operation against the workspace's
	// current quota for the current billing period.
	CheckQuota(ctx context.Context, req QuotaRequest) (QuotaDecision, error)

	// GetSubscription returns the current subscription state for a workspace,
	// or a zero-value *Subscription with Tier=TierFree if none exists.
	GetSubscription(ctx context.Context, workspaceID string) (*Subscription, error)
}

// ErrNotImplemented is returned by Providers that do not implement a feature.
// The engine treats this as an open permission (the operation is allowed).
var ErrNotImplemented = errors.New("billing: not implemented by this provider")
