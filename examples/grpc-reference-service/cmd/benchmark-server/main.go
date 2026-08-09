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
	// The service identity scoping this command's error reasons. A production
	// composition reads its own from configuration; this one has none to read
	// and names itself, so the measured error path is shaped like production's.
	benchmarkErrorDomain = "grpc-reference-benchmark"
)

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
			// The listener binds loopback only, so this command is the one place
			// plaintext is not a deployment decision.
			TransportCredentials: nil,
			Logger:               slog.New(slog.NewJSONHandler(benchmarkJSONSink{}, nil)),
			MeterProvider:        meterProvider,
			TracerProvider:       tracerProvider,
			Propagators:          propagation.TraceContext{},
			Load:                 load,
			// Without these mappers the reference service's own limit errors
			// reach the caller as INTERNAL; grpcreference.ErrStreamLimit owns why.
			DomainErrors: grpcreference.DomainErrors(),
			// Named so the measured error path carries the same details a
			// production one does; an empty domain would silently drop ErrorInfo
			// from every error this command measures.
			ErrorDomain: benchmarkErrorDomain,
			Services: []grpcx.RegisterService{
				func(registrar grpc.ServiceRegistrar) {
					referencev1.RegisterEchoServiceServer(registrar, grpcreference.Service{})
				},
			},
			// This command measures the transport, so it adds no policy of its
			// own beyond what NewServer composes.
			UnaryPolicy:  nil,
			StreamPolicy: nil,
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
	return stopServer(ctx, server, serveDone)
}

// stopServer waits out the shutdown budget, forces whatever is left, and reports
// every failure the stop produced rather than the first.
//
// Each step contributes to one list because none of them cancels the next: a
// graceful stop that ran out of budget still has to be forced, and Serve's own
// result still has to be collected either way.
func stopServer(parent context.Context, server *grpcx.Server, serveDone <-chan error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), shutdownTimeout)
	defer cancel()

	var stopErrors []error
	if err := server.Shutdown(ctx); err != nil {
		stopErrors = append(stopErrors, err, server.Close())
	}
	select {
	case serveErr := <-serveDone:
		stopErrors = append(stopErrors, serveErr)
	case <-ctx.Done():
		stopErrors = append(stopErrors, ctx.Err(), server.Close())
	}

	if err := errors.Join(stopErrors...); err != nil {
		return fmt.Errorf("stop gRPC benchmark server: %w", err)
	}
	return nil
}

// benchmarkJSONSink keeps JSON encoding in the measured server path while
// discarding only the final encoded bytes.
type benchmarkJSONSink struct{}

func (benchmarkJSONSink) Write(data []byte) (int, error) {
	return len(data), nil
}

type benchmarkServerSettings struct {
	transport      grpcx.Config
	maxConnections int
}

// settingsFromDefaults is this example's crossing from loaded configuration to
// the transport adapter's bounds, and the third place that crossing is written
// out: cmd/service/internal/bootstrap.grpcServerConfig is the production one and
// serverConfigFromRuntime in internal/infra/grpc/config_parity_test.go is that
// package's oracle. Neither is importable from here — one is internal to another
// binary, the other is a test — so a bound added to grpcx.Config has to be added
// in all three. TestBenchmarkServerProcessLifecycle asks the target-side
// question that makes a bound missed here fail rather than quietly change what
// the benchmark measures.
func settingsFromDefaults(defaults config.GRPCServerConfig) (benchmarkServerSettings, error) {
	if defaults.MaxConnections <= 0 {
		return benchmarkServerSettings{}, errors.New("build gRPC benchmark server: max connections must be positive")
	}
	// The remaining transport bounds are proven by grpcx.NewServer, which owns
	// the one range check every composition root would otherwise repeat.
	return benchmarkServerSettings{
		transport: grpcx.Config{
			MaxConcurrentRPCs:          defaults.MaxConcurrentRPCs,
			MaxConcurrentHealthRPCs:    defaults.MaxConnections,
			MaxConcurrentStreams:       defaults.MaxConcurrentStreams,
			MaxHeaderListBytes:         defaults.MaxHeaderListBytes,
			MaxReceiveMessageBytes:     defaults.MaxReceiveMessageBytes,
			MaxSendMessageBytes:        defaults.MaxSendMessageBytes,
			AccessLogHealthChecks:      defaults.AccessLogHealthChecks,
			AccessLogSuccessSampleRate: defaults.AccessLogSuccessSampleRate,
			AccessLogSlowThreshold:     defaults.AccessLogSlowThreshold,
			TelemetryHealthChecks:      defaults.TelemetryHealthChecks,
			UnaryTimeout:               defaults.UnaryTimeout,
			StreamTimeout:              defaults.StreamTimeout,
			MaxConnectionIdle:          defaults.MaxConnectionIdle,
			ServerPingInterval:         defaults.ServerPingInterval,
			ServerPingTimeout:          defaults.ServerPingTimeout,
			MinClientPingInterval:      defaults.MinClientPingInterval,
			PermitPingWithoutStream:    defaults.PermitPingWithoutStream,
			MaxConnectionAge:           defaults.MaxConnectionAge,
			MaxConnectionAgeGrace:      defaults.MaxConnectionAgeGrace,
		},
		maxConnections: defaults.MaxConnections,
	}, nil
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
