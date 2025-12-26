package monitoring

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracingConfig holds configuration for distributed tracing
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   string // "jaeger", "otlp", "stdout"
	Endpoint       string
	SamplingRate   float64
}

// TracingManager manages distributed tracing
type TracingManager struct {
	tracer         oteltrace.Tracer
	tracerProvider *trace.TracerProvider
	config         TracingConfig
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config TracingConfig) (*TracingManager, error) {
	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on type
	var exporter trace.SpanExporter
	switch config.ExporterType {
	case "jaeger":
		exporter, err = createJaegerExporter(config.Endpoint)
	case "otlp":
		exporter, err = createOTLPExporter(config.Endpoint)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", config.ExporterType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer := tp.Tracer(config.ServiceName)

	return &TracingManager{
		tracer:         tracer,
		tracerProvider: tp,
		config:         config,
	}, nil
}

func createJaegerExporter(endpoint string) (trace.SpanExporter, error) {
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces"
	}
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

func createOTLPExporter(endpoint string) (trace.SpanExporter, error) {
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)

	return otlptrace.New(context.Background(), client)
}

// Shutdown gracefully shuts down the tracer provider
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.tracerProvider != nil {
		return tm.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new span
func (tm *TracingManager) StartSpan(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return tm.tracer.Start(ctx, spanName, opts...)
}

// StartWorkflowSpan starts a span for workflow execution
func (tm *TracingManager) StartWorkflowSpan(ctx context.Context, workflowID, workflowName string) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, fmt.Sprintf("workflow.%s", workflowName),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	)

	span.SetAttributes(
		attribute.String("workflow.id", workflowID),
		attribute.String("workflow.name", workflowName),
	)

	return ctx, span
}

// StartNodeSpan starts a span for node execution
func (tm *TracingManager) StartNodeSpan(ctx context.Context, nodeType, nodeName string) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, fmt.Sprintf("node.%s", nodeType),
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
	)

	span.SetAttributes(
		attribute.String("node.type", nodeType),
		attribute.String("node.name", nodeName),
	)

	return ctx, span
}

// StartHTTPSpan starts a span for HTTP operations
func (tm *TracingManager) StartHTTPSpan(ctx context.Context, method, url string) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, fmt.Sprintf("http.%s", method),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)

	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.url", url),
	)

	return ctx, span
}

// StartDatabaseSpan starts a span for database operations
func (tm *TracingManager) StartDatabaseSpan(ctx context.Context, dbType, operation, query string) (context.Context, oteltrace.Span) {
	ctx, span := tm.StartSpan(ctx, fmt.Sprintf("db.%s.%s", dbType, operation),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)

	span.SetAttributes(
		attribute.String("db.system", dbType),
		attribute.String("db.operation", operation),
		attribute.String("db.statement", query),
	)

	return ctx, span
}

// RecordError records an error in the span
func (tm *TracingManager) RecordError(span oteltrace.Span, err error) {
	if span != nil && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddEvent adds an event to the span
func (tm *TracingManager) AddEvent(span oteltrace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil {
		span.AddEvent(name, oteltrace.WithAttributes(attrs...))
	}
}

// SetAttributes sets attributes on the span
func (tm *TracingManager) SetAttributes(span oteltrace.Span, attrs ...attribute.KeyValue) {
	if span != nil {
		span.SetAttributes(attrs...)
	}
}

// ExtractSpanContext extracts span context from carrier
func (tm *TracingManager) ExtractSpanContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// InjectSpanContext injects span context into carrier
func (tm *TracingManager) InjectSpanContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// GetTracer returns the underlying tracer
func (tm *TracingManager) GetTracer() oteltrace.Tracer {
	return tm.tracer
}