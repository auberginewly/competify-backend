// Package observability initializes OpenTelemetry tracing and Prometheus metrics.
// Every Agent execution is automatically wrapped in a span via TraceAgentExecution.
package observability

// InitTracer initializes OTel TracerProvider with Jaeger exporter.
//
// Phase 8 implementation:
//
//	exp, _ := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
//	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), ...)
//	otel.SetTracerProvider(tp)
func InitTracer(serviceName, jaegerEndpoint string) error {
	// TODO Phase 8
	return nil
}
