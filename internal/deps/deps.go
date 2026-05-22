// Package deps locks all runtime dependencies into go.mod.
// This file is intentionally empty of logic; its only purpose is to import
// every third-party package we will use so that `go mod tidy` keeps them.
//
// After Phase 1+ implementation fills real imports in other files,
// this file can be safely deleted — the direct imports will survive in go.mod.
package deps

import (
	_ "github.com/cloudwego/eino/compose"
	_ "github.com/cloudwego/eino/schema"
	_ "github.com/cloudwego/eino-ext/components/model/openai"
	_ "github.com/cloudwego/hertz/pkg/app/server"
	_ "github.com/hertz-contrib/websocket"
	_ "github.com/nats-io/nats.go"
	_ "github.com/nats-io/nats-server/v2/server"
	_ "go.opentelemetry.io/otel"
	_ "go.opentelemetry.io/otel/attribute"
	_ "go.opentelemetry.io/otel/codes"
	_ "go.opentelemetry.io/otel/exporters/jaeger"
	_ "go.opentelemetry.io/otel/sdk/resource"
	_ "go.opentelemetry.io/otel/sdk/trace"
	_ "go.opentelemetry.io/otel/semconv/v1.24.0"
	_ "go.opentelemetry.io/otel/trace"
	_ "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/promauto"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/go-resty/resty/v2"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/pgvector/pgvector-go"
	_ "golang.org/x/sync/errgroup"
	_ "golang.org/x/time/rate"
)
