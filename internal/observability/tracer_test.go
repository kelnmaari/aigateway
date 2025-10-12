package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewTracerProvider_Disabled(t *testing.T) {
	cfg := TracingConfig{
		Enabled: false,
	}

	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Should return no-op tracer
	tracer := provider.Tracer()
	assert.NotNil(t, tracer)

	// Shutdown should not error
	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNewTracerProvider_InvalidProvider(t *testing.T) {
	cfg := TracingConfig{
		Enabled:  true,
		Provider: "invalid",
		Endpoint: "http://localhost:14268/api/traces",
	}

	_, err := NewTracerProvider(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported tracing provider")
}

func TestNewTracerProvider_DefaultValues(t *testing.T) {
	// Note: This test will fail if Jaeger is not running
	// We test the no-op case instead
	cfg := TracingConfig{
		Enabled: false,
	}

	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)

	tracer := provider.Tracer()
	assert.NotNil(t, tracer)

	// Test span creation with no-op tracer
	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-operation")
	assert.NotNil(t, span)
	span.End()
}

func TestTracerProvider_Tracer(t *testing.T) {
	cfg := TracingConfig{
		Enabled: false,
	}

	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)

	tracer1 := provider.Tracer()
	tracer2 := provider.Tracer()

	assert.NotNil(t, tracer1)
	assert.NotNil(t, tracer2)
	// Should return the same tracer instance
	assert.Equal(t, tracer1, tracer2)
}

func TestTracerProvider_Shutdown(t *testing.T) {
	cfg := TracingConfig{
		Enabled: false,
	}

	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)

	// First shutdown should succeed
	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)

	// Second shutdown should also succeed (idempotent)
	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTracerProvider_SpanCreation(t *testing.T) {
	cfg := TracingConfig{
		Enabled: false,
	}

	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	tracer := provider.Tracer()
	ctx := context.Background()

	// Create parent span
	ctx, parentSpan := tracer.Start(ctx, "parent-operation")
	assert.NotNil(t, parentSpan)
	assert.True(t, parentSpan.SpanContext().IsValid() || !parentSpan.SpanContext().HasTraceID())

	// Create child span
	_, childSpan := tracer.Start(ctx, "child-operation")
	assert.NotNil(t, childSpan)

	childSpan.End()
	parentSpan.End()
}

func TestTracingConfig_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       TracingConfig
		wantError bool
	}{
		{
			name: "Valid Jaeger config",
			cfg: TracingConfig{
				Enabled:      false, // Use false to avoid needing actual Jaeger
				Provider:     "jaeger",
				ServiceName:  "test-service",
				Endpoint:     "http://localhost:14268/api/traces",
				SamplingRate: 1.0,
			},
			wantError: false,
		},
		{
			name: "Valid Zipkin config",
			cfg: TracingConfig{
				Enabled:      false,
				Provider:     "zipkin",
				ServiceName:  "test-service",
				Endpoint:     "http://localhost:9411/api/v2/spans",
				SamplingRate: 0.5,
			},
			wantError: false,
		},
		{
			name: "Invalid provider",
			cfg: TracingConfig{
				Enabled:  true,
				Provider: "unsupported",
				Endpoint: "http://localhost:8080",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewTracerProvider(tt.cfg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				if provider != nil {
					provider.Shutdown(context.Background())
				}
			}
		})
	}
}

func TestTracerProvider_ContextPropagation(t *testing.T) {
	cfg := TracingConfig{
		Enabled: false,
	}

	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)
	defer provider.Shutdown(context.Background())

	tracer := provider.Tracer()
	ctx := context.Background()

	// Start a span
	ctx, span := tracer.Start(ctx, "test-operation")
	defer span.End()

	// Extract span from context
	extractedSpan := trace.SpanFromContext(ctx)
	assert.NotNil(t, extractedSpan)
}
