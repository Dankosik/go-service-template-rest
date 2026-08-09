package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// benchmarkJSONSink keeps JSON encoding in the measured server path while
// discarding only the final encoded bytes.
type benchmarkJSONSink struct{}

func (benchmarkJSONSink) Write(data []byte) (int, error) {
	return len(data), nil
}

type otelLoadRecorder struct {
	active metric.Int64UpDownCounter
	shed   metric.Int64Counter
}

// newOTelLoadRecorder stands in for internal/infra/telemetry, which this
// example does not build. It emits that package's series under that package's
// names so the benchmark carries the instrumentation cost a real service pays;
// only the descriptions differ, to say where the numbers came from.
func newOTelLoadRecorder(provider metric.MeterProvider) (otelLoadRecorder, error) {
	meter := provider.Meter(telemetry.GRPCServerMeterName)
	active, err := meter.Int64UpDownCounter(
		telemetry.ActiveRPCsInstrument,
		metric.WithDescription("RPCs currently executing a handler during the local gRPC benchmark."),
		metric.WithUnit(telemetry.RPCsUnit),
	)
	if err != nil {
		return otelLoadRecorder{}, fmt.Errorf("build active-RPC benchmark instrument: %w", err)
	}
	shed, err := meter.Int64Counter(
		telemetry.ShedRPCsInstrument,
		metric.WithDescription("RPCs rejected by admission during the local gRPC benchmark."),
		metric.WithUnit(telemetry.RPCsUnit),
	)
	if err != nil {
		return otelLoadRecorder{}, fmt.Errorf("build shed-RPC benchmark instrument: %w", err)
	}
	return otelLoadRecorder{active: active, shed: shed}, nil
}

func (r otelLoadRecorder) Admitted(ctx context.Context) func() {
	r.active.Add(ctx, 1)
	return func() { r.active.Add(ctx, -1) }
}

func (r otelLoadRecorder) Shed(ctx context.Context) {
	r.shed.Add(ctx, 1)
}

func shutdownProviders(
	parent context.Context,
	meterProvider *sdkmetric.MeterProvider,
	tracerProvider *sdktrace.TracerProvider,
) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), time.Second)
	defer cancel()
	_ = errors.Join(meterProvider.Shutdown(ctx), tracerProvider.Shutdown(ctx))
}
