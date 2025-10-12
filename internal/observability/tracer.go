// Package observability provides distributed tracing and monitoring capabilities.
//
// This package implements OpenTelemetry integration for distributed tracing,
// allowing detailed visibility into request flows across the proxy system.
package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig holds tracing configuration options
type TracingConfig struct {
	Enabled      bool
	Provider     string // "jaeger" or "zipkin"
	ServiceName  string
	Endpoint     string
	SamplingRate float64 // 0.0 to 1.0 (1.0 = 100% sampling)
}

// TracerProvider manages OpenTelemetry tracer provider and tracer instances
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewTracerProvider creates and configures OpenTelemetry tracer provider.
//
// It initializes the tracer with specified exporter (Jaeger or Zipkin),
// sets up context propagation, and configures sampling rate.
// If tracing is disabled, returns a no-op tracer.
//
// Parameters:
//   - cfg: Tracing configuration with provider type, endpoint, and sampling
//
// Returns:
//   - *TracerProvider: Configured tracer provider
//   - error: If initialization fails
func NewTracerProvider(cfg TracingConfig) (*TracerProvider, error) {
	if !cfg.Enabled {
		// Return no-op tracer if tracing is disabled
		return &TracerProvider{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		}, nil
	}

	// Set default service name if not provided
	if cfg.ServiceName == "" {
		cfg.ServiceName = "ollama-proxy"
	}

	// Set default sampling rate if not provided
	if cfg.SamplingRate == 0 {
		cfg.SamplingRate = 1.0
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String("1.6.1"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on provider type
	var exporter sdktrace.SpanExporter
	switch cfg.Provider {
	case "jaeger":
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(cfg.Endpoint),
		))
		if err != nil {
			return nil, fmt.Errorf("failed to create jaeger exporter: %w", err)
		}
	case "zipkin":
		exporter, err = zipkin.New(cfg.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to create zipkin exporter: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported tracing provider: %s (supported: jaeger, zipkin)", cfg.Provider)
	}

	// Create tracer provider with sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation (W3C Trace Context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: tp,
		tracer:   tp.Tracer("ollama-proxy"),
	}, nil
}

// Tracer returns the tracer instance for creating spans.
//
// Use this tracer to create spans for custom operations that need tracing.
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// Shutdown flushes and shuts down the tracer provider.
//
// This should be called on application shutdown to ensure all spans
// are exported before termination. It waits for all pending exports
// to complete.
//
// Parameters:
//   - ctx: Context for shutdown timeout
//
// Returns:
//   - error: If shutdown fails
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}
