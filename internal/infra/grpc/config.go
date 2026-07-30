// Package grpcx owns the native gRPC server transport adapter.
package grpcx

import (
	"context"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Config contains transport bounds already validated by the runtime config
// owner. NewServer validates them again so direct package users cannot
// accidentally recover an unlimited native default.
type Config struct {
	MaxConcurrentRPCs          int
	MaxConcurrentStreams       uint32
	MaxHeaderListBytes         uint32
	MaxReceiveMessageBytes     int
	MaxSendMessageBytes        int
	LogHealthChecks            bool
	AccessLogSuccessSampleRate float64
	AccessLogSlowThreshold     time.Duration
	TelemetryHealthChecks      bool
	TransportCredentials       credentials.TransportCredentials
}

// RegisterService attaches one generated service implementation to a registrar.
type RegisterService func(grpc.ServiceRegistrar)

// Options supplies process-owned collaborators without making this transport
// import feature packages or the bootstrap composition root.
type Options struct {
	Logger          *slog.Logger
	MeterProvider   metric.MeterProvider
	TracerProvider  trace.TracerProvider
	Propagators     propagation.TextMapPropagator
	DomainErrors    []problem.Mapper
	Load            LoadRecorder
	Services        []RegisterService
	UnaryPolicy     []grpc.UnaryServerInterceptor
	StreamingPolicy []grpc.StreamServerInterceptor
}

// LoadRecorder observes the process-wide admission decision.
type LoadRecorder interface {
	Admitted(ctx context.Context) func()
	Shed(ctx context.Context)
}
