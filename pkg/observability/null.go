package observability

import "context"

// NullEmitter discards all telemetry. Default for embedders that do not need
// observability.
type NullEmitter struct{}

// NewNullEmitter returns an Emitter that discards everything.
func NewNullEmitter() *NullEmitter { return &NullEmitter{} }

func (NullEmitter) EmitEvent(ctx context.Context, evt Event)                {}
func (NullEmitter) IncCounter(ctx context.Context, c Counter)               {}
func (NullEmitter) SetGauge(ctx context.Context, g Gauge)                   {}
func (NullEmitter) ObserveHistogram(ctx context.Context, h Histogram)       {}
func (NullEmitter) Flush(ctx context.Context) error                         { return nil }

var _ Emitter = (*NullEmitter)(nil)
