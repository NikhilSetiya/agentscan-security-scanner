package tracing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	ServiceName     string  `json:"service_name"`
	ServiceVersion  string  `json:"service_version"`
	Environment     string  `json:"environment"`
	JaegerEndpoint  string  `json:"jaeger_endpoint"`
	SamplingRate    float64 `json:"sampling_rate"`
	Enabled         bool    `json:"enabled"`
}

// DefaultConfig returns default tracing configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:     "agentscan",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		SamplingRate:    1.0,
		Enabled:         true,
	}
}

// TracingService manages distributed tracing
type TracingService struct {
	tracer   oteltrace.Tracer
	config   *Config
	provider *trace.TracerProvider
}

// NewTracingService creates a new tracing service
func NewTracingService(config *Config) (*TracingService, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return &TracingService{
			tracer: otel.Tracer("noop"),
			config: config,
		}, nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer(config.ServiceName)

	return &TracingService{
		tracer:   tracer,
		config:   config,
		provider: tp,
	}, nil
}

// Shutdown shuts down the tracing service
func (ts *TracingService) Shutdown(ctx context.Context) error {
	if ts.provider != nil {
		return ts.provider.Shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new span
func (ts *TracingService) StartSpan(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, name, opts...)
}

// StartHTTPSpan starts a span for HTTP requests
func (ts *TracingService) StartHTTPSpan(ctx context.Context, method, path string) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, fmt.Sprintf("%s %s", method, path),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		oteltrace.WithAttributes(
			semconv.HTTPMethodKey.String(method),
			semconv.HTTPRouteKey.String(path),
		),
	)
}

// StartScanSpan starts a span for scan operations
func (ts *TracingService) StartScanSpan(ctx context.Context, operation string, jobID, repoURL string) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, fmt.Sprintf("scan.%s", operation),
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
		oteltrace.WithAttributes(
			attribute.String("scan.job_id", jobID),
			attribute.String("scan.repo_url", repoURL),
			attribute.String("scan.operation", operation),
		),
	)
}

// StartAgentSpan starts a span for agent operations
func (ts *TracingService) StartAgentSpan(ctx context.Context, agentName, operation string) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, fmt.Sprintf("agent.%s.%s", agentName, operation),
		oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
		oteltrace.WithAttributes(
			attribute.String("agent.name", agentName),
			attribute.String("agent.operation", operation),
		),
	)
}

// StartDatabaseSpan starts a span for database operations
func (ts *TracingService) StartDatabaseSpan(ctx context.Context, operation, table string) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, fmt.Sprintf("db.%s", operation),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			semconv.DBOperationKey.String(operation),
			semconv.DBSQLTableKey.String(table),
		),
	)
}

// StartCacheSpan starts a span for cache operations
func (ts *TracingService) StartCacheSpan(ctx context.Context, operation, key string) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, fmt.Sprintf("cache.%s", operation),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			semconv.DBSystemRedis,
			semconv.DBOperationKey.String(operation),
			attribute.String("cache.key", key),
		),
	)
}

// AddSpanAttributes adds attributes to the current span
func (ts *TracingService) AddSpanAttributes(span oteltrace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the current span
func (ts *TracingService) AddSpanEvent(span oteltrace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

// RecordError records an error in the current span
func (ts *TracingService) RecordError(span oteltrace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(oteltrace.StatusCodeError, err.Error())
}

// SetSpanStatus sets the status of the current span
func (ts *TracingService) SetSpanStatus(span oteltrace.Span, code oteltrace.StatusCode, description string) {
	span.SetStatus(code, description)
}

// TracingMiddleware creates a middleware for distributed tracing
func (ts *TracingService) TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ts.config.Enabled {
			c.Next()
			return
		}

		// Extract trace context from headers
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start span
		ctx, span := ts.StartHTTPSpan(ctx, c.Request.Method, c.FullPath())
		defer span.End()

		// Add request attributes
		span.SetAttributes(
			semconv.HTTPURLKey.String(c.Request.URL.String()),
			semconv.HTTPUserAgentKey.String(c.Request.UserAgent()),
			semconv.HTTPClientIPKey.String(c.ClientIP()),
		)

		// Update request context
		c.Request = c.Request.WithContext(ctx)

		// Inject trace context into response headers
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(c.Writer.Header()))

		// Process request
		c.Next()

		// Add response attributes
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(c.Writer.Status()),
			semconv.HTTPResponseSizeKey.Int(c.Writer.Size()),
		)

		// Set span status based on HTTP status code
		if c.Writer.Status() >= 400 {
			span.SetStatus(oteltrace.StatusCodeError, fmt.Sprintf("HTTP %d", c.Writer.Status()))
		} else {
			span.SetStatus(oteltrace.StatusCodeOk, "")
		}

		// Record errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				ts.RecordError(span, err.Err)
			}
		}
	}
}

// InstrumentHTTPClient instruments an HTTP client for tracing
func (ts *TracingService) InstrumentHTTPClient(client *http.Client) *http.Client {
	if !ts.config.Enabled {
		return client
	}

	// Wrap the transport
	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	client.Transport = &tracingTransport{
		base:    client.Transport,
		service: ts,
	}

	return client
}

// tracingTransport wraps http.RoundTripper for tracing
type tracingTransport struct {
	base    http.RoundTripper
	service *TracingService
}

// RoundTrip implements http.RoundTripper
func (tt *tracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, span := tt.service.tracer.Start(req.Context(), fmt.Sprintf("HTTP %s", req.Method),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			semconv.HTTPMethodKey.String(req.Method),
			semconv.HTTPURLKey.String(req.URL.String()),
		),
	)
	defer span.End()

	// Inject trace context into request headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Update request context
	req = req.WithContext(ctx)

	// Make the request
	resp, err := tt.base.RoundTrip(req)
	if err != nil {
		tt.service.RecordError(span, err)
		return resp, err
	}

	// Add response attributes
	span.SetAttributes(
		semconv.HTTPStatusCodeKey.Int(resp.StatusCode),
	)

	// Set span status
	if resp.StatusCode >= 400 {
		span.SetStatus(oteltrace.StatusCodeError, fmt.Sprintf("HTTP %d", resp.StatusCode))
	} else {
		span.SetStatus(oteltrace.StatusCodeOk, "")
	}

	return resp, nil
}

// TraceableFunction wraps a function with tracing
func (ts *TracingService) TraceableFunction(ctx context.Context, name string, fn func(ctx context.Context) error) error {
	ctx, span := ts.StartSpan(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		ts.RecordError(span, err)
		return err
	}

	ts.SetSpanStatus(span, oteltrace.StatusCodeOk, "")
	return nil
}

// TraceableFunctionWithResult wraps a function with tracing and returns a result
func (ts *TracingService) TraceableFunctionWithResult[T any](ctx context.Context, name string, fn func(ctx context.Context) (T, error)) (T, error) {
	ctx, span := ts.StartSpan(ctx, name)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		ts.RecordError(span, err)
		return result, err
	}

	ts.SetSpanStatus(span, oteltrace.StatusCodeOk, "")
	return result, nil
}

// GetTraceID returns the trace ID from the context
func GetTraceID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the context
func GetSpanID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// WithTraceContext adds trace and span IDs to the logging context
func WithTraceContext(ctx context.Context) context.Context {
	traceID := GetTraceID(ctx)
	spanID := GetSpanID(ctx)

	if traceID != "" {
		ctx = context.WithValue(ctx, "trace_id", traceID)
	}
	if spanID != "" {
		ctx = context.WithValue(ctx, "span_id", spanID)
	}

	return ctx
}