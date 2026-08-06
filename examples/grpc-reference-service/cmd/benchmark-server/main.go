package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcreference "github.com/example/go-service-template-rest/examples/grpc-reference-service"
	referencev1 "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/reference/v1"
	"github.com/example/go-service-template-rest/internal/config"
	grpcx "github.com/example/go-service-template-rest/internal/infra/grpc"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc"
)

const (
	readyPrefix     = "GRPC_BENCH_READY="
	shutdownTimeout = 5 * time.Second
)

type benchmarkServerSettings struct {
	transport      grpcx.Config
	maxConnections int
}

type otelLoadRecorder struct {
	active metric.Int64UpDownCounter
	shed   metric.Int64Counter
}

// benchmarkJSONSink keeps JSON encoding in the measured server path while
// discarding only the final encoded bytes.
type benchmarkJSONSink struct{}

func (benchmarkJSONSink) Write(data []byte) (int, error) {
	return len(data), nil
}

func main() {
	os.Exit(mainExitCode())
}

func mainExitCode() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "gRPC benchmark server: %v\n", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, ready io.Writer) error {
	settings, err := settingsFromDefaults(config.DefaultGRPCServerConfig())
	if err != nil {
		return err
	}

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen for gRPC benchmark: %w", err)
	}
	defer func() { _ = listener.Close() }()

	meterProvider := sdkmetric.NewMeterProvider()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()))
	defer shutdownProviders(ctx, meterProvider, tracerProvider)

	load, err := newOTelLoadRecorder(meterProvider)
	if err != nil {
		return err
	}
	server, err := grpcx.NewServer( //nolint:contextcheck // Construction precedes RPC contexts; grpcx passes each stream context to its interceptors.
		settings.transport,
		grpcx.Options{
			Logger:         slog.New(slog.NewJSONHandler(benchmarkJSONSink{}, nil)),
			MeterProvider:  meterProvider,
			TracerProvider: tracerProvider,
			Propagators:    propagation.TraceContext{},
			Load:           load,
			// Without these mappers the reference service's own limit errors
			// reach the caller as INTERNAL; grpcreference.ErrStreamLimit owns why.
			DomainErrors: grpcreference.DomainErrors(),
			Services: []grpcx.RegisterService{
				func(registrar grpc.ServiceRegistrar) {
					referencev1.RegisterEchoServiceServer(registrar, grpcreference.Service{})
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("build gRPC benchmark server: %w", err)
	}

	limitedListener := netutil.LimitListener(listener, settings.maxConnections)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(limitedListener)
	}()

	server.MarkServing()
	if _, err := fmt.Fprintf(ready, "%s%s\n", readyPrefix, listener.Addr().String()); err != nil {
		_ = server.Close()
		return fmt.Errorf("publish gRPC benchmark readiness: %w", err)
	}

	select {
	case err := <-serveDone:
		if err != nil {
			return err
		}
		return errors.New("gRPC benchmark server stopped before shutdown")
	case <-ctx.Done():
	}

	server.StartDrain()
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}

	select {
	case serveErr := <-serveDone:
		if serveErr != nil {
			shutdownErr = errors.Join(shutdownErr, serveErr)
		}
	case <-shutdownCtx.Done():
		shutdownErr = errors.Join(shutdownErr, shutdownCtx.Err(), server.Close())
	}
	if shutdownErr != nil {
		return fmt.Errorf("stop gRPC benchmark server: %w", shutdownErr)
	}
	return nil
}

func settingsFromDefaults(defaults config.GRPCServerConfig) (benchmarkServerSettings, error) {
	if defaults.MaxConnections <= 0 {
		return benchmarkServerSettings{}, errors.New("build gRPC benchmark server: max connections must be positive")
	}
	// The remaining transport bounds are proven by grpcx.NewServer, which owns
	// the one range check every composition root would otherwise repeat.
	return benchmarkServerSettings{
		transport: grpcx.Config{
			MaxConcurrentRPCs:          defaults.MaxConcurrentRPCs,
			MaxConcurrentStreams:       defaults.MaxConcurrentStreams,
			MaxHeaderListBytes:         defaults.MaxHeaderListBytes,
			MaxReceiveMessageBytes:     defaults.MaxReceiveMessageBytes,
			MaxSendMessageBytes:        defaults.MaxSendMessageBytes,
			LogHealthChecks:            defaults.AccessLogHealthChecks,
			AccessLogSuccessSampleRate: defaults.AccessLogSuccessSampleRate,
			AccessLogSlowThreshold:     defaults.AccessLogSlowThreshold,
			TelemetryHealthChecks:      defaults.TelemetryHealthChecks,
		},
		maxConnections: defaults.MaxConnections,
	}, nil
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
