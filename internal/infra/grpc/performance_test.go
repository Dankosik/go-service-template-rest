// The cost of each composition layer, and the proof that the benchmarked servers
// are the ones they claim to be.
//
// A benchmark measuring a chain it did not build reports a number that means
// nothing, so the variants that claim to be the repository chain do not rebuild
// it: they call unaryChain, which is what NewServer calls. What remains for a
// test to catch is the set of builtins reaching it, which
// TestBenchmarkVariantsCoverEveryBuiltinPolicy holds, and that each variant's
// layers actually run, which TestGRPCBenchmarkVariantsComposeExpectedLayers
// drives every variant through.

package grpcx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const benchmarkUnaryFullMethod = "/grpcx.benchmark.Service/Unary"

// benchmarkShape is the server a variant stands up, and the only thing
// newBenchmarkFixture branches on. Everything else a variant changes is a value
// on its row below, so a new variant on an existing shape adds no control flow.
type benchmarkShape string

const (
	shapeBare   benchmarkShape = "bare"   // grpc.NewServer with nothing attached
	shapeOTel   benchmarkShape = "otel"   // plus the OpenTelemetry stats handler
	shapePolicy benchmarkShape = "policy" // plus the repository chain, less policyVariantExcludes
	shapeFull   benchmarkShape = "full"   // NewServer itself
)

// benchmarkVariant is one measured composition: the server it stands up, the
// settings that server is built with, and what one RPC through it must show.
//
// One row answers both what a variant measures and what proves it measured that.
// The alternative — a bare name whose meaning is reassembled from arms of the
// fixture switch and arms of the assertion switch — is how a variant ends up
// measuring something other than what its name claims with nothing to say so.
//
// name reaches recorded benchmark output, so changing one orphans every number
// filed under it; docs/grpc.md names full_json in its profiling example.
type benchmarkVariant struct {
	name  string
	shape benchmarkShape

	// logLevel and sampleRate are the access-log settings the server is built
	// with. Only shapeFull reads them, because no other shape builds a Config.
	logLevel   slog.Level
	sampleRate float64

	// handlerWork makes the handler checksum its payload, so the chain can be
	// measured against a handler that does something rather than one that
	// returns immediately.
	handlerWork bool

	// wantAccessLogRecord is whether one successful RPC must reach the access
	// log. False covers both ways a full-shape server suppresses the record —
	// the level rejects it, or sampling drops it — which is the pair
	// full_log_disabled and full_success_unsampled exist to separate.
	wantAccessLogRecord bool
}

var benchmarkVariants = []benchmarkVariant{
	{name: "bare", shape: shapeBare},
	{name: "otel", shape: shapeOTel},
	{name: "policy", shape: shapePolicy},
	{
		name:       "full_log_disabled",
		shape:      shapeFull,
		logLevel:   slog.LevelError,
		sampleRate: 1,
	},
	{
		name:       "full_success_unsampled",
		shape:      shapeFull,
		logLevel:   slog.LevelInfo,
		sampleRate: 0,
	},
	{
		name:                "full_json",
		shape:               shapeFull,
		logLevel:            slog.LevelInfo,
		sampleRate:          1,
		wantAccessLogRecord: true,
	},
	{
		name:                "full_handler_work",
		shape:               shapeFull,
		logLevel:            slog.LevelInfo,
		sampleRate:          1,
		handlerWork:         true,
		wantAccessLogRecord: true,
	},
}

// policyVariantExcludes names the builtins the policy shape leaves out, so that
// what it measures is a decision rather than an omission. access_log and
// recovery are excluded because BenchmarkGRPCAccessLog and the full_* variants
// isolate their cost.
var policyVariantExcludes = []string{builtinAccessLog, builtinRecovery}

// knownBuiltinPolicies is the builtin set this file was written against.
//
// unaryChain owns the order surrounding these, so what this guards is the set
// itself: a policy added to builtinPolicies would otherwise join the full_*
// variants and the policy variant silently, and every recorded number would
// compare against a chain that no longer exists — a quiet regression in the
// numbers rather than a red test.
var knownBuiltinPolicies = []string{
	builtinAccessLog,
	builtinDeadline,
	builtinRecovery,
	builtinAdmission,
	builtinPolicyErrorBoundary,
}

// benchmarkUnaryTimeout is the unary bound the measured shapes run under. It is
// the production default rather than a benchmark number, because the deadline
// policy costs a timer per RPC only when it is enabled — measuring it disabled
// would record a chain no deployment runs.
const benchmarkUnaryTimeout = 8 * time.Second

func TestBenchmarkVariantsCoverEveryBuiltinPolicy(t *testing.T) {
	names := make([]string, 0, len(knownBuiltinPolicies))
	for _, policy := range builtinPolicies(
		slog.New(slog.DiscardHandler),
		accessLogPolicy{},
		newAdmissionPolicy(1, 1, noopLoadRecorder{}),
		benchmarkUnaryTimeout,
	) {
		names = append(names, policy.name)
	}

	if !slices.Equal(names, knownBuiltinPolicies) {
		t.Fatalf(
			"builtinPolicies() = %v, want %v; decide whether the benchmark variants should "+
				"measure the change, then update knownBuiltinPolicies, policyVariantExcludes, "+
				"and the interceptor order doc.go publishes",
			names,
			knownBuiltinPolicies,
		)
	}
	for _, excluded := range policyVariantExcludes {
		if !slices.Contains(names, excluded) {
			t.Fatalf("policyVariantExcludes names %q, which builtinPolicies no longer returns", excluded)
		}
	}
}

func TestGRPCBenchmarkVariantsComposeExpectedLayers(t *testing.T) {
	for _, variant := range benchmarkVariants {
		t.Run(variant.name, func(t *testing.T) {
			fixture, err := newBenchmarkFixture(variant)
			if err != nil {
				t.Fatalf("newBenchmarkFixture(%q) error = %v", variant.name, err)
			}
			t.Cleanup(func() {
				if err := fixture.close(); err != nil {
					t.Errorf("benchmark fixture close error = %v", err)
				}
			})

			payload := []byte("composition-probe")
			response, header, err := fixture.invoke(t.Context(), payload, true)
			if err != nil {
				t.Fatalf("invoke(%q) error = %v", variant.name, err)
			}
			if !bytes.Equal(response, payload) {
				t.Fatalf("invoke(%q) response = %q, want %q", variant.name, response, payload)
			}
			if err := fixture.finishRPCs(t.Context()); err != nil {
				t.Fatalf("finish benchmark RPCs for %q: %v", variant.name, err)
			}
			fixture.assertComposition(t, variant, header)
		})
	}
}

func BenchmarkGRPCUnary(b *testing.B) {
	payloads := []struct {
		name string
		size int
	}{
		{name: "64B", size: 64},
		{name: "1KiB", size: 1 << 10},
		{name: "64KiB", size: 64 << 10},
		{name: "1MiB", size: 1 << 20},
	}

	for _, variant := range benchmarkVariants {
		b.Run(variant.name, func(b *testing.B) {
			for _, payloadCase := range payloads {
				b.Run(payloadCase.name, func(b *testing.B) {
					fixture, err := newBenchmarkFixture(variant)
					if err != nil {
						b.Fatalf("newBenchmarkFixture(%q) error = %v", variant.name, err)
					}
					b.Cleanup(func() {
						if err := fixture.close(); err != nil {
							b.Errorf("benchmark fixture close error = %v", err)
						}
					})

					payload := bytes.Repeat([]byte{'x'}, payloadCase.size)
					response, _, err := fixture.invoke(b.Context(), payload, false)
					if err != nil {
						b.Fatalf("untimed invoke(%q) error = %v", variant.name, err)
					}
					if !bytes.Equal(response, payload) {
						b.Fatalf("untimed invoke(%q) returned the wrong payload", variant.name)
					}

					b.ReportAllocs()
					b.SetBytes(int64(len(payload)))
					b.ResetTimer()
					for b.Loop() {
						response, _, err := fixture.invoke(b.Context(), payload, false)
						if err != nil {
							b.Fatalf("invoke(%q) error = %v", variant.name, err)
						}
						if !bytes.Equal(response, payload) {
							b.Fatalf("invoke(%q) returned the wrong payload", variant.name)
						}
					}
				})
			}
		})
	}
}

func BenchmarkGRPCAccessLog(b *testing.B) {
	ctx := reqctx.ContextWithRequestID(b.Context(), "benchmark-request-id")
	handler := func(context.Context, any) (any, error) {
		return struct{}{}, nil
	}

	for _, testCase := range []struct {
		name   string
		level  slog.Level
		method string
		policy accessLogPolicy
	}{
		{
			name:   "full_json",
			level:  slog.LevelInfo,
			method: benchmarkUnaryFullMethod,
			policy: accessLogPolicy{successSampleRate: 1},
		},
		{
			name:   "success_unsampled",
			level:  slog.LevelInfo,
			method: benchmarkUnaryFullMethod,
			policy: accessLogPolicy{successSampleRate: 0},
		},
		{
			name:   "level_disabled",
			level:  slog.LevelWarn,
			method: benchmarkUnaryFullMethod,
			policy: accessLogPolicy{successSampleRate: 1},
		},
		{
			name:   "health_excluded",
			level:  slog.LevelInfo,
			method: healthMethodPrefix + "Check",
			policy: accessLogPolicy{successSampleRate: 1},
		},
	} {
		b.Run(testCase.name, func(b *testing.B) {
			log := slog.New(slog.NewJSONHandler(
				benchmarkDiscardWriter{},
				&slog.HandlerOptions{Level: testCase.level},
			))
			interceptor := asUnaryInterceptor(accessLogAround(log, testCase.policy))
			info := &grpc.UnaryServerInfo{FullMethod: testCase.method}

			b.ReportAllocs()
			for b.Loop() {
				if _, err := interceptor(ctx, nil, info, handler); err != nil {
					b.Fatalf("access-log interceptor error = %v", err)
				}
			}
		})
	}
}

// benchmarkDiscardWriter retains JSON encoding work while excluding sink I/O.
//
// firstRecordWriter below does the same and is not reused here, because it also
// takes a lock to retain the first record for the composition assertions. This
// benchmark measures the access-log policy itself, so the only lock in the
// measured path should be one production also pays.
type benchmarkDiscardWriter struct{}

func (benchmarkDiscardWriter) Write(payload []byte) (int, error) {
	return len(payload), nil
}

type benchmarkFixture struct {
	connection     *grpc.ClientConn
	listener       *bufconn.Listener
	server         benchmarkServer
	serveDone      chan error
	signals        *benchmarkSignals
	meterProvider  *sdkmetric.MeterProvider
	tracerProvider *sdktrace.TracerProvider
	stopOnce       sync.Once
	stopErr        error
	serveOnce      sync.Once
	serveErr       error
}

type benchmarkServer interface {
	Serve(listener net.Listener) error
	Shutdown(ctx context.Context) error
	Close() error
}

type nativeBenchmarkServer struct {
	*grpc.Server
}

func (s nativeBenchmarkServer) Close() error {
	s.Stop()
	return nil
}

func (s nativeBenchmarkServer) Shutdown(context.Context) error {
	s.GracefulStop()
	return nil
}

func newBenchmarkFixture(variant benchmarkVariant) (*benchmarkFixture, error) {
	signals := &benchmarkSignals{}
	handler := benchmarkUnaryHandler{
		signals: signals,
		work:    variant.handlerWork,
	}
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, benchmarkUnaryFullMethod, handler.Unary)
	}

	var (
		server         benchmarkServer
		meterProvider  *sdkmetric.MeterProvider
		tracerProvider *sdktrace.TracerProvider
	)

	switch variant.shape {
	case shapeBare:
		native := grpc.NewServer()
		register(native)
		server = nativeBenchmarkServer{Server: native}
	case shapeOTel:
		meterProvider, tracerProvider = signals.newOTelProviders()
		registeredMethods := map[string]struct{}{benchmarkUnaryFullMethod: {}}
		native := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler(
			otelgrpc.WithMeterProvider(meterProvider),
			otelgrpc.WithTracerProvider(tracerProvider),
			otelgrpc.WithPropagators(propagation.TraceContext{}),
			otelgrpc.WithFilter(func(info *stats.RPCTagInfo) bool {
				_, ok := registeredMethods[info.FullMethodName]
				return ok
			}),
		)))
		register(native)
		server = nativeBenchmarkServer{Server: native}
	case shapePolicy:
		// The repository policy chain without the two builtins whose cost the
		// full_* variants isolate. The builtins come from builtinPolicies and the
		// surrounding order from unaryChain — the same two owners NewServer uses —
		// so this shape measures production's chain rather than a copy of it.
		// The only way to leave a builtin out is to name it in
		// policyVariantExcludes.
		admission := newAdmissionPolicy(256, 256, &signals.load)
		measured := make([]builtinPolicy, 0, len(knownBuiltinPolicies))
		for _, policy := range builtinPolicies(
			slog.New(slog.DiscardHandler),
			accessLogPolicy{},
			admission,
			benchmarkUnaryTimeout,
		) {
			if slices.Contains(policyVariantExcludes, policy.name) {
				continue
			}
			measured = append(measured, policy)
		}
		native := grpc.NewServer(grpc.ChainUnaryInterceptor(unaryChain(
			slog.New(slog.DiscardHandler),
			measured,
			[]grpc.UnaryServerInterceptor{signals.policyInterceptor()},
			handlerErrorBoundary(slog.New(slog.DiscardHandler), errorRendering{}),
		)...))
		register(native)
		server = nativeBenchmarkServer{Server: native}
	case shapeFull:
		meterProvider, tracerProvider = signals.newOTelProviders()
		log := slog.New(slog.NewJSONHandler(&signals.log, &slog.HandlerOptions{Level: variant.logLevel}))
		serverConfig := testServerConfig()
		serverConfig.AccessLogSuccessSampleRate = variant.sampleRate
		serverConfig.UnaryTimeout = benchmarkUnaryTimeout
		fullServer, err := NewServer(
			serverConfig,
			Options{
				Logger:         log,
				MeterProvider:  meterProvider,
				TracerProvider: tracerProvider,
				Propagators:    propagation.TraceContext{},
				Load:           &signals.load,
				Services:       []RegisterService{register},
				UnaryPolicy:    []grpc.UnaryServerInterceptor{signals.policyInterceptor()},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("build full benchmark server: %w", err)
		}
		server = fullServer
	default:
		return nil, fmt.Errorf("unknown benchmark shape %q", variant.shape)
	}

	listener := bufconn.Listen(2 << 20)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	// Everything acquired above belongs to the fixture from here, so a failed
	// dial unwinds through the same close() a finished fixture uses. A second
	// unwind written out here is the copy that stops releasing whatever the next
	// field to be added needs releasing.
	fixture := &benchmarkFixture{
		listener:       listener,
		server:         server,
		serveDone:      serveDone,
		signals:        signals,
		meterProvider:  meterProvider,
		tracerProvider: tracerProvider,
	}

	connection, err := grpc.NewClient(
		"passthrough:///grpcx-benchmark",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("build benchmark client: %w", err), fixture.close())
	}

	fixture.connection = connection
	return fixture, nil
}

func (f *benchmarkFixture) invoke(
	ctx context.Context,
	payload []byte,
	captureHeader bool,
) ([]byte, metadata.MD, error) {
	request := wrapperspb.Bytes(payload)
	response := new(wrapperspb.BytesValue)
	var header metadata.MD
	options := []grpc.CallOption(nil)
	if captureHeader {
		options = append(options, grpc.Header(&header))
	}
	if err := f.connection.Invoke(ctx, benchmarkUnaryFullMethod, request, response, options...); err != nil {
		return nil, header, fmt.Errorf("invoke benchmark unary RPC: %w", err)
	}
	return response.GetValue(), header, nil
}

func (f *benchmarkFixture) close() error {
	var closeErrors []error
	// nil while newBenchmarkFixture is unwinding a dial that never returned one.
	if f.connection != nil {
		if err := f.connection.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close benchmark client: %w", err))
		}
	}
	if err := f.stop(context.Background(), false); err != nil {
		closeErrors = append(closeErrors, err)
	}
	if err := f.listener.Close(); err != nil {
		closeErrors = append(closeErrors, fmt.Errorf("close benchmark listener: %w", err))
	}
	if f.meterProvider != nil {
		if err := f.meterProvider.Shutdown(context.Background()); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("shutdown benchmark meter provider: %w", err))
		}
	}
	if f.tracerProvider != nil {
		if err := f.tracerProvider.Shutdown(context.Background()); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("shutdown benchmark tracer provider: %w", err))
		}
	}
	return errors.Join(closeErrors...)
}

func (f *benchmarkFixture) finishRPCs(ctx context.Context) error {
	return f.stop(ctx, true)
}

func (f *benchmarkFixture) stop(ctx context.Context, graceful bool) error {
	f.stopOnce.Do(func() {
		if graceful {
			if err := f.server.Shutdown(ctx); err != nil {
				f.stopErr = fmt.Errorf("shutdown benchmark server: %w", err)
			}
			return
		}
		if err := f.server.Close(); err != nil {
			f.stopErr = fmt.Errorf("close benchmark server: %w", err)
		}
	})

	return errors.Join(f.stopErr, f.waitServe())
}

func (f *benchmarkFixture) waitServe() error {
	f.serveOnce.Do(func() {
		err := <-f.serveDone
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) && !errors.Is(err, net.ErrClosed) {
			f.serveErr = fmt.Errorf("serve benchmark: %w", err)
		}
	})
	return f.serveErr
}

// assertComposition proves the RPC just made actually ran the layers variant
// claims. Each branch below is the shape's own question; everything a single
// variant decides is read from its row rather than named here.
func (f *benchmarkFixture) assertComposition(t *testing.T, variant benchmarkVariant, header metadata.MD) {
	t.Helper()

	switch variant.shape {
	case shapeBare:
		if got := f.signals.load.admitted.Load(); got != 0 {
			t.Fatalf("bare admission count = %d, want 0", got)
		}
	case shapeOTel:
		f.signals.assertRPCMetric(t)
	case shapePolicy:
		f.assertRepositoryChainRan(t, variant.name, header)
	case shapeFull:
		f.assertRepositoryChainRan(t, variant.name, header)
		f.signals.assertRPCMetric(t)
		f.signals.assertAdmissionMetric(t)
		f.assertAccessLog(t, variant)
	}

	if variant.handlerWork {
		const wantChecksum = 15811
		if got := f.signals.handlerChecksum.Load(); got != wantChecksum {
			t.Fatalf("%s handler checksum = %d, want %d", variant.name, got, wantChecksum)
		}
	}
}

// assertRepositoryChainRan proves the layers every shape carrying the repository
// chain must run: correlation published one ID, admission took and released one
// slot, and the supplied policy interceptor was reached.
func (f *benchmarkFixture) assertRepositoryChainRan(t *testing.T, name string, header metadata.MD) {
	t.Helper()

	requestIDs := header.Get(requestIDMetadataKey)
	if len(requestIDs) != 1 || requestIDs[0] == "" {
		t.Fatalf("%s response request IDs = %v, want one non-empty ID", name, requestIDs)
	}
	if got := f.signals.load.admitted.Load(); got != 1 {
		t.Fatalf("%s admitted count = %d, want 1", name, got)
	}
	if got := f.signals.load.released.Load(); got != 1 {
		t.Fatalf("%s released count = %d, want 1", name, got)
	}
	if got := f.signals.policyCalls.Load(); got != 1 {
		t.Fatalf("%s policy count = %d, want 1", name, got)
	}
}

// assertAccessLog holds a full-shape variant to the record its row claims: one
// decodable production record, or none at all when the level or the sample rate
// was set to suppress it.
func (f *benchmarkFixture) assertAccessLog(t *testing.T, variant benchmarkVariant) {
	t.Helper()

	if !variant.wantAccessLogRecord {
		if got := f.signals.log.writeCount(); got != 0 {
			t.Fatalf("%s successful access-log writes = %d, want 0", variant.name, got)
		}
		return
	}

	var record map[string]any
	if err := json.Unmarshal(f.signals.log.firstRecord(), &record); err != nil {
		t.Fatalf("decode %s JSON access record: %v", variant.name, err)
	}
	if got := record["msg"]; got != "grpc_request" {
		t.Fatalf("%s access record msg = %v, want grpc_request", variant.name, got)
	}
	if got := record["rpc.method"]; got != benchmarkUnaryFullMethod {
		t.Fatalf("%s access record rpc.method = %v, want %s", variant.name, got, benchmarkUnaryFullMethod)
	}
}

type benchmarkSignals struct {
	load            benchmarkLoadRecorder
	policyCalls     atomic.Int64
	handlerChecksum atomic.Uint64
	log             firstRecordWriter
	metricReader    *sdkmetric.ManualReader
}

func (s *benchmarkSignals) newOTelProviders() (*sdkmetric.MeterProvider, *sdktrace.TracerProvider) {
	s.metricReader = sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(s.metricReader))
	s.load.init(meterProvider)
	return meterProvider, sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()))
}

func (s *benchmarkSignals) policyInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		request any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		s.policyCalls.Add(1)
		return handler(ctx, request)
	}
}

func (s *benchmarkSignals) assertRPCMetric(t *testing.T) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := s.metricReader.Collect(t.Context(), &resourceMetrics); err != nil {
		t.Fatalf("collect benchmark OTel metrics: %v", err)
	}
	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, recorded := range scopeMetrics.Metrics {
			if metricHasDataPoints(recorded.Data) {
				return
			}
		}
	}
	t.Fatal("OTel benchmark variant recorded no RPC metric data points")
}

func (s *benchmarkSignals) assertAdmissionMetric(t *testing.T) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := s.metricReader.Collect(t.Context(), &resourceMetrics); err != nil {
		t.Fatalf("collect benchmark admission metrics: %v", err)
	}
	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		if scopeMetrics.Scope.Name != telemetry.GRPCServerMeterName {
			continue
		}
		for _, recorded := range scopeMetrics.Metrics {
			if recorded.Name == telemetry.ActiveRPCsInstrument && metricHasDataPoints(recorded.Data) {
				return
			}
		}
	}
	t.Fatalf(
		"full benchmark variant recorded no %s/%s data point",
		telemetry.GRPCServerMeterName,
		telemetry.ActiveRPCsInstrument,
	)
}

func metricHasDataPoints(data metricdata.Aggregation) bool {
	switch value := data.(type) {
	case metricdata.Sum[int64]:
		return len(value.DataPoints) > 0
	case metricdata.Sum[float64]:
		return len(value.DataPoints) > 0
	case metricdata.Histogram[int64]:
		return len(value.DataPoints) > 0
	case metricdata.Histogram[float64]:
		return len(value.DataPoints) > 0
	default:
		return false
	}
}

// benchmarkLoadRecorder counts admission decisions with atomics rather than
// reusing recordingLoad from interceptors_test.go, which counts the same things
// under a mutex.
//
// The difference is the point: this recorder sits in the measured path of every
// full_* variant, where a lock production does not hold would be charged to the
// interceptor chain, and at higher parallelism would be measured as contention
// the service does not have. recordingLoad is driven by unit tests that need an
// exact snapshot instead, which is what the mutex buys there.
type benchmarkLoadRecorder struct {
	admitted       atomic.Int64
	released       atomic.Int64
	shed           atomic.Int64
	active         metric.Int64UpDownCounter
	shedInstrument metric.Int64Counter
}

// init emits the same series internal/infra/telemetry gives a real service, so
// the full_* variants measure the instrumentation cost production pays rather
// than a cheaper stand-in.
func (r *benchmarkLoadRecorder) init(provider metric.MeterProvider) {
	meter := provider.Meter(telemetry.GRPCServerMeterName)
	r.active, _ = meter.Int64UpDownCounter(
		telemetry.ActiveRPCsInstrument,
		metric.WithDescription(telemetry.ActiveRPCsDescription),
		metric.WithUnit(telemetry.RPCsUnit),
	)
	r.shedInstrument, _ = meter.Int64Counter(
		telemetry.ShedRPCsInstrument,
		metric.WithDescription(telemetry.ShedRPCsDescription),
		metric.WithUnit(telemetry.RPCsUnit),
	)
}

func (r *benchmarkLoadRecorder) Admitted(ctx context.Context) func() {
	r.admitted.Add(1)
	if r.active != nil {
		r.active.Add(ctx, 1)
	}
	return func() {
		r.released.Add(1)
		if r.active != nil {
			r.active.Add(ctx, -1)
		}
	}
}

func (r *benchmarkLoadRecorder) Shed(ctx context.Context) {
	r.shed.Add(1)
	if r.shedInstrument != nil {
		r.shedInstrument.Add(ctx, 1)
	}
}

// firstRecordWriter counts access-log writes and keeps the first one, which is
// what lets assertComposition tell a sampled-out variant from a logging one and
// decode the record a logging variant produced.
type firstRecordWriter struct {
	mu     sync.Mutex
	writes int
	first  []byte
}

func (w *firstRecordWriter) Write(record []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writes++
	if w.first == nil {
		w.first = append([]byte(nil), record...)
	}
	return len(record), nil
}

func (w *firstRecordWriter) writeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writes
}

func (w *firstRecordWriter) firstRecord() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.first...)
}

type benchmarkUnaryHandler struct {
	signals *benchmarkSignals
	work    bool
}

func (h benchmarkUnaryHandler) Unary(
	_ context.Context,
	request *wrapperspb.BytesValue,
) (*wrapperspb.BytesValue, error) {
	if h.work {
		var checksum uint64
		for index, value := range request.GetValue() {
			checksum += uint64(value) * uint64(index+1)
		}
		h.signals.handlerChecksum.Store(checksum)
	}
	return wrapperspb.Bytes(request.GetValue()), nil
}
