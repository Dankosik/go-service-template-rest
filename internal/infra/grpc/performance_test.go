package grpcx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"testing"

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

type benchmarkVariant string

const (
	benchmarkBare            benchmarkVariant = "bare"
	benchmarkOTel            benchmarkVariant = "otel"
	benchmarkPolicy          benchmarkVariant = "policy"
	benchmarkFullLogDisabled benchmarkVariant = "full_log_disabled"
	benchmarkFullUnsampled   benchmarkVariant = "full_success_unsampled"
	benchmarkFullJSON        benchmarkVariant = "full_json"
	benchmarkFullHandlerWork benchmarkVariant = "full_handler_work"
)

var benchmarkVariants = []benchmarkVariant{
	benchmarkBare,
	benchmarkOTel,
	benchmarkPolicy,
	benchmarkFullLogDisabled,
	benchmarkFullUnsampled,
	benchmarkFullJSON,
	benchmarkFullHandlerWork,
}

func TestGRPCBenchmarkVariantsComposeExpectedLayers(t *testing.T) {
	for _, variant := range benchmarkVariants {
		t.Run(string(variant), func(t *testing.T) {
			fixture, err := newBenchmarkFixture(variant)
			if err != nil {
				t.Fatalf("newBenchmarkFixture(%q) error = %v", variant, err)
			}
			t.Cleanup(func() {
				if err := fixture.close(); err != nil {
					t.Errorf("benchmark fixture close error = %v", err)
				}
			})

			payload := []byte("composition-probe")
			response, header, err := fixture.invoke(t.Context(), payload, true)
			if err != nil {
				t.Fatalf("invoke(%q) error = %v", variant, err)
			}
			if !bytes.Equal(response, payload) {
				t.Fatalf("invoke(%q) response = %q, want %q", variant, response, payload)
			}
			if err := fixture.finishRPCs(t.Context()); err != nil {
				t.Fatalf("finish benchmark RPCs for %q: %v", variant, err)
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
		b.Run(string(variant), func(b *testing.B) {
			for _, payloadCase := range payloads {
				b.Run(payloadCase.name, func(b *testing.B) {
					fixture, err := newBenchmarkFixture(variant)
					if err != nil {
						b.Fatalf("newBenchmarkFixture(%q) error = %v", variant, err)
					}
					b.Cleanup(func() {
						if err := fixture.close(); err != nil {
							b.Errorf("benchmark fixture close error = %v", err)
						}
					})

					payload := bytes.Repeat([]byte{'x'}, payloadCase.size)
					response, _, err := fixture.invoke(b.Context(), payload, false)
					if err != nil {
						b.Fatalf("untimed invoke(%q) error = %v", variant, err)
					}
					if !bytes.Equal(response, payload) {
						b.Fatalf("untimed invoke(%q) returned the wrong payload", variant)
					}

					b.ReportAllocs()
					b.SetBytes(int64(len(payload)))
					b.ResetTimer()
					for b.Loop() {
						response, _, err := fixture.invoke(b.Context(), payload, false)
						if err != nil {
							b.Fatalf("invoke(%q) error = %v", variant, err)
						}
						if !bytes.Equal(response, payload) {
							b.Fatalf("invoke(%q) returned the wrong payload", variant)
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
			interceptor := accessLogUnaryInterceptor(log, testCase.policy)
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
		work:    variant == benchmarkFullHandlerWork,
	}
	register := func(registrar grpc.ServiceRegistrar) {
		registrar.RegisterService(&benchmarkUnaryServiceDesc, handler)
	}

	var (
		server         benchmarkServer
		meterProvider  *sdkmetric.MeterProvider
		tracerProvider *sdktrace.TracerProvider
	)

	switch variant {
	case benchmarkBare:
		native := grpc.NewServer()
		register(native)
		server = nativeBenchmarkServer{Server: native}
	case benchmarkOTel:
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
	case benchmarkPolicy:
		admission := newAdmissionLimiter(256, &signals.load)
		native := grpc.NewServer(grpc.ChainUnaryInterceptor(
			correlationUnaryInterceptor(),
			admissionUnaryInterceptor(admission),
			policyErrorBoundaryUnaryInterceptor(),
			signals.policyInterceptor(),
			errorMappingUnaryInterceptor(nil),
		))
		register(native)
		server = nativeBenchmarkServer{Server: native}
	case benchmarkFullLogDisabled, benchmarkFullUnsampled, benchmarkFullJSON, benchmarkFullHandlerWork:
		meterProvider, tracerProvider = signals.newOTelProviders()
		level := slog.LevelInfo
		if variant == benchmarkFullLogDisabled {
			level = slog.LevelError
		}
		log := slog.New(slog.NewJSONHandler(&signals.log, &slog.HandlerOptions{Level: level}))
		serverConfig := testServerConfig()
		if variant == benchmarkFullUnsampled {
			serverConfig.AccessLogSuccessSampleRate = 0
		}
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
		return nil, fmt.Errorf("unknown benchmark variant %q", variant)
	}

	listener := bufconn.Listen(2 << 20)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	connection, err := grpc.NewClient(
		"passthrough:///grpcx-benchmark",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = server.Close()
		<-serveDone
		_ = listener.Close()
		if meterProvider != nil {
			_ = meterProvider.Shutdown(context.Background())
		}
		if tracerProvider != nil {
			_ = tracerProvider.Shutdown(context.Background())
		}
		return nil, fmt.Errorf("build benchmark client: %w", err)
	}

	return &benchmarkFixture{
		connection:     connection,
		listener:       listener,
		server:         server,
		serveDone:      serveDone,
		signals:        signals,
		meterProvider:  meterProvider,
		tracerProvider: tracerProvider,
	}, nil
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
	if err := f.connection.Close(); err != nil {
		closeErrors = append(closeErrors, fmt.Errorf("close benchmark client: %w", err))
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

func (f *benchmarkFixture) assertComposition(t *testing.T, variant benchmarkVariant, header metadata.MD) {
	t.Helper()

	switch variant {
	case benchmarkBare:
		if got := f.signals.load.admitted.Load(); got != 0 {
			t.Fatalf("bare admission count = %d, want 0", got)
		}
	case benchmarkOTel:
		f.signals.assertRPCMetric(t)
	case benchmarkPolicy, benchmarkFullLogDisabled, benchmarkFullUnsampled, benchmarkFullJSON, benchmarkFullHandlerWork:
		requestIDs := header.Get(requestIDMetadataKey)
		if len(requestIDs) != 1 || requestIDs[0] == "" {
			t.Fatalf("%s response request IDs = %v, want one non-empty ID", variant, requestIDs)
		}
		if got := f.signals.load.admitted.Load(); got != 1 {
			t.Fatalf("%s admitted count = %d, want 1", variant, got)
		}
		if got := f.signals.load.released.Load(); got != 1 {
			t.Fatalf("%s released count = %d, want 1", variant, got)
		}
		if got := f.signals.policyCalls.Load(); got != 1 {
			t.Fatalf("%s policy count = %d, want 1", variant, got)
		}
		if variant != benchmarkPolicy {
			f.signals.assertRPCMetric(t)
			f.signals.assertAdmissionMetric(t)
		}
	}

	switch variant {
	case benchmarkBare, benchmarkOTel, benchmarkPolicy:
	case benchmarkFullLogDisabled, benchmarkFullUnsampled:
		if got := f.signals.log.writeCount(); got != 0 {
			t.Fatalf("%s successful access-log writes = %d, want 0", variant, got)
		}
	case benchmarkFullJSON, benchmarkFullHandlerWork:
		var record map[string]any
		if err := json.Unmarshal(f.signals.log.firstRecord(), &record); err != nil {
			t.Fatalf("decode production JSON access record: %v", err)
		}
		if got := record["msg"]; got != "grpc_request" {
			t.Fatalf("JSON access record msg = %v, want grpc_request", got)
		}
		if got := record["rpc.method"]; got != benchmarkUnaryFullMethod {
			t.Fatalf("JSON access record rpc.method = %v, want %s", got, benchmarkUnaryFullMethod)
		}
	}
	if variant == benchmarkFullHandlerWork {
		const wantChecksum = 15811
		if got := f.signals.handlerChecksum.Load(); got != wantChecksum {
			t.Fatalf("full_handler_work checksum = %d, want %d", got, wantChecksum)
		}
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
		for _, metric := range scopeMetrics.Metrics {
			if metricHasDataPoints(metric.Data) {
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
		if scopeMetrics.Scope.Name != "service.grpc.server" {
			continue
		}
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name == "rpc.server.active_requests" && metricHasDataPoints(metric.Data) {
				return
			}
		}
	}
	t.Fatal("full benchmark variant recorded no service.grpc.server/rpc.server.active_requests data point")
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

type benchmarkLoadRecorder struct {
	admitted       atomic.Int64
	released       atomic.Int64
	shed           atomic.Int64
	active         metric.Int64UpDownCounter
	shedInstrument metric.Int64Counter
}

func (r *benchmarkLoadRecorder) init(provider metric.MeterProvider) {
	meter := provider.Meter("service.grpc.server")
	r.active, _ = meter.Int64UpDownCounter(
		"rpc.server.active_requests",
		metric.WithDescription(
			"RPCs currently executing a handler, against the grpc.server.max_concurrent_rpcs limit.",
		),
		metric.WithUnit("{rpc}"),
	)
	r.shedInstrument, _ = meter.Int64Counter(
		"rpc.server.shed_requests",
		metric.WithDescription(
			"RPCs rejected before running a handler because the process RPC limit was reached.",
		),
		metric.WithUnit("{rpc}"),
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

type benchmarkUnaryService interface {
	Unary(ctx context.Context, request *wrapperspb.BytesValue) (*wrapperspb.BytesValue, error)
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

func benchmarkUnaryMethodHandler(
	service any,
	ctx context.Context, //nolint:revive // grpc.MethodDesc requires the service argument before context.
	decode func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	request := new(wrapperspb.BytesValue)
	if err := decode(request); err != nil {
		return nil, err
	}
	typedService, ok := service.(benchmarkUnaryService)
	if !ok {
		return nil, errors.New("benchmark unary service has unexpected type")
	}
	if interceptor == nil {
		response, err := typedService.Unary(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("run benchmark unary service: %w", err)
		}
		return response, nil
	}
	info := &grpc.UnaryServerInfo{
		Server:     service,
		FullMethod: benchmarkUnaryFullMethod,
	}
	handler := func(ctx context.Context, request any) (any, error) {
		typedRequest, ok := request.(*wrapperspb.BytesValue)
		if !ok {
			return nil, errors.New("benchmark unary request has unexpected type")
		}
		response, err := typedService.Unary(ctx, typedRequest)
		if err != nil {
			return nil, fmt.Errorf("run intercepted benchmark unary service: %w", err)
		}
		return response, nil
	}
	return interceptor(ctx, request, info, handler)
}

var benchmarkUnaryServiceDesc = grpc.ServiceDesc{
	ServiceName: "grpcx.benchmark.Service",
	HandlerType: (*benchmarkUnaryService)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Unary",
			Handler:    benchmarkUnaryMethodHandler,
		},
	},
}
