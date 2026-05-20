// Package observability defines the extension contract for telemetry —
// metrics, traces, and structured events emitted by the m9m engine.
//
// m9m OSS ships a PrometheusEmitter that exposes per-workspace-labeled
// metrics on the standard Prometheus HTTP handler, and a NullEmitter that
// discards everything. m9m Cloud's cloud-mode worker uses a custom
// CloudControlPlaneEmitter that reports per-tenant cost-attribution metrics
// to the orchestrator over gRPC.
//
// This is part of the public embedding surface and commits to semver.
package observability

import "context"

// Severity classifies an event for filtering and routing.
type Severity int

const (
	SeverityDebug Severity = iota
	SeverityInfo
	SeverityWarn
	SeverityError
)

// Event is a structured event emitted by the engine.
type Event struct {
	WorkspaceID string
	Severity    Severity
	// Kind names the event category. Conventional values:
	//   "execution.started", "execution.completed", "execution.failed",
	//   "node.invoked", "snapshot.saved", "snapshot.loaded", "quota.denied",
	//   "hibernate.start", "hibernate.complete"
	Kind string
	// Message is a human-readable description. May be empty.
	Message string
	// Attrs is structured context, similar to slog.Attr. Optional.
	Attrs map[string]any
}

// Counter records a monotonically increasing metric. Used for "how many
// executions ran?" style measurements.
type Counter struct {
	WorkspaceID string
	Name        string // e.g. "execution_total"
	Labels      map[string]string
	Value       int64
}

// Gauge records a current-value metric. Used for "how many concurrent
// executions are running?" style measurements.
type Gauge struct {
	WorkspaceID string
	Name        string
	Labels      map[string]string
	Value       float64
}

// Histogram records a value distribution. Used for "execution duration"
// style measurements.
type Histogram struct {
	WorkspaceID string
	Name        string
	Labels      map[string]string
	Value       float64
}

// Emitter is the extension contract for telemetry.
//
// All methods are safe for concurrent use. The engine emits on the hot path;
// implementations must return within microseconds for counter/gauge calls.
// Slow implementations should buffer internally and flush asynchronously.
type Emitter interface {
	// EmitEvent records a structured event.
	EmitEvent(ctx context.Context, evt Event)

	// IncCounter increments a counter by Value.
	IncCounter(ctx context.Context, c Counter)

	// SetGauge sets a gauge to Value.
	SetGauge(ctx context.Context, g Gauge)

	// ObserveHistogram records a histogram observation.
	ObserveHistogram(ctx context.Context, h Histogram)

	// Flush drains any buffered data. Called on shutdown and hibernation.
	// Implementations that emit synchronously may return nil immediately.
	Flush(ctx context.Context) error
}
