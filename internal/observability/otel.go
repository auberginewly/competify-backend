// Package observability initializes OpenTelemetry tracing and Prometheus metrics.
// Every Agent execution is automatically wrapped in a span via TraceAgentExecution.
package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const tracerName = "competify-ai"

// InitTracer initializes the global OTel TracerProvider with a Jaeger exporter.
// jaegerEndpoint e.g. "http://localhost:14268/api/traces".
// Returns a shutdown func that must be deferred by the caller.
func InitTracer(serviceName, jaegerEndpoint string) (func(), error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(jaegerEndpoint),
	))
	if err != nil {
		return nil, fmt.Errorf("observability.InitTracer: jaeger exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("service.version", "0.1.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("observability.InitTracer: resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		// Sample every trace in dev; use ParentBased(TraceIDRatioBased(0.1)) in prod.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	shutdown := func() {
		ctx := context.Background()
		_ = tp.Shutdown(ctx)
	}
	return shutdown, nil
}
