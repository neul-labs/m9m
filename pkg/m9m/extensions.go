package m9m

import (
	"github.com/neul-labs/m9m/pkg/billing"
	"github.com/neul-labs/m9m/pkg/blob"
	"github.com/neul-labs/m9m/pkg/observability"
	"github.com/neul-labs/m9m/pkg/quota"
	"github.com/neul-labs/m9m/pkg/snapshot"
)

// The extension interface aliases below define the contract between the
// embedding API (pkg/m9m) and the extension packages. They commit to semver:
// new methods are additive only; breaking changes require a v2.
//
// Embedders construct concrete implementations (typically in their own
// repo — e.g. m9m-cloud wires Stripe billing, Hetzner-OSS snapshot/blob,
// gRPC observability) and pass them in via the With* options.

type (
	billingProvider      = billing.Provider
	snapshotSink         = snapshot.Sink
	blobStore            = blob.Store
	quotaEnforcer        = quota.Enforcer
	observabilityEmitter = observability.Emitter
)

// defaultBillingProvider returns the no-op billing provider used when an
// embedder does not configure one. Self-hosted users get unrestricted
// quota.
func defaultBillingProvider() billingProvider { return billing.NewNullProvider() }

// defaultSnapshotSink returns the disabled snapshot sink. Workflows still
// execute, but Hibernate/Revive operations return ErrDisabled — the engine
// keeps the process resident instead.
func defaultSnapshotSink() snapshotSink { return snapshot.NewDisabled() }

// defaultBlobStore returns the disabled blob store. Used by embedders that
// keep all per-tenant state inside the engine's storage backend without
// separate object storage.
func defaultBlobStore() blobStore { return blob.NewDisabled() }

// defaultQuotaEnforcer returns the unlimited quota enforcer.
func defaultQuotaEnforcer() quotaEnforcer { return quota.NewUnlimited() }

// defaultObservabilityEmitter returns the null emitter (discards
// everything). Embedders typically replace this with PrometheusEmitter or a
// gRPC control-plane emitter.
func defaultObservabilityEmitter() observabilityEmitter { return observability.NewNullEmitter() }

// WithBilling sets the billing provider. The engine calls CheckQuota and
// ReportUsage on this provider during workflow execution. Self-hosted users
// who want unrestricted execution should leave this unset (the default is
// NullProvider, which always allows). Cloud-mode workers pass a
// Stripe-backed Provider.
func WithBilling(p billing.Provider) Option {
	return func(e *Engine) {
		if p != nil {
			e.billing = p
		}
	}
}

// WithSnapshot sets the snapshot sink. The engine calls Save during
// hibernation and Load during revive. Default is Disabled (no
// hibernate/revive). Cloud-mode workers wire an S3-compatible sink against
// Hetzner Object Storage.
func WithSnapshot(s snapshot.Sink) Option {
	return func(e *Engine) {
		if s != nil {
			e.snapshot = s
		}
	}
}

// WithBlob sets the per-workspace blob store used for workflow JSON,
// execution logs, encrypted credentials, npm caches, and other per-tenant
// content. Default is Disabled (per-tenant state stays inside the storage
// backend). Cloud-mode workers wire an S3-compatible store against Hetzner
// Object Storage.
func WithBlob(s blob.Store) Option {
	return func(e *Engine) {
		if s != nil {
			e.blob = s
		}
	}
}

// WithQuota sets the quota enforcer. The engine asks Check before every
// execution and reports usage via Observe. Default is Unlimited. Cloud-mode
// workers pass a billing-aware enforcer; self-hosted multi-tenant
// deployments pass a ConfigEnforcer with YAML-driven limits.
func WithQuota(q quota.Enforcer) Option {
	return func(e *Engine) {
		if q != nil {
			e.quota = q
		}
	}
}

// WithObservability sets the telemetry emitter. The engine emits events,
// counters, gauges, and histograms during execution. Default is NullEmitter.
// Cloud-mode workers emit per-tenant cost-attribution metrics to the
// orchestrator over gRPC.
func WithObservability(em observability.Emitter) Option {
	return func(e *Engine) {
		if em != nil {
			e.observability = em
		}
	}
}

// Billing returns the configured billing provider. Always non-nil; returns
// the default NullProvider if WithBilling was not called.
func (e *Engine) Billing() billing.Provider { return e.billing }

// Snapshot returns the configured snapshot sink. Always non-nil.
func (e *Engine) Snapshot() snapshot.Sink { return e.snapshot }

// Blob returns the configured blob store. Always non-nil.
func (e *Engine) Blob() blob.Store { return e.blob }

// Quota returns the configured quota enforcer. Always non-nil.
func (e *Engine) Quota() quota.Enforcer { return e.quota }

// Observability returns the configured telemetry emitter. Always non-nil.
func (e *Engine) Observability() observability.Emitter { return e.observability }
