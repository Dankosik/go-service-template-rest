package grpcclient_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func TestPropagationPoliciesApplyToUnaryAndStreamingRPCs(t *testing.T) {
	unaryMetadata, streamMetadata, target := startMetadataCaptureServer(t)
	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})

	for _, testCase := range []struct {
		name          string
		policy        grpcclient.PropagationPolicy
		requestID     string
		wantTrace     bool
		wantRequestID bool
	}{
		{name: "none", policy: grpcclient.PropagationNone, requestID: "request_none_123"},
		{
			name:      "trace context",
			policy:    grpcclient.PropagationTraceContext,
			requestID: "request_trace_123",
			wantTrace: true,
		},
		{
			name:          "trusted service",
			policy:        grpcclient.PropagationTrustedService,
			requestID:     "request_trusted_123",
			wantTrace:     true,
			wantRequestID: true,
		},
		{
			name:      "trusted service rejects invalid request ID",
			policy:    grpcclient.PropagationTrustedService,
			requestID: "invalid request id",
			wantTrace: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			connection, err := grpcclient.New(
				grpcclient.DefaultConfig(target),
				grpcclient.Options{
					TransportCredentials: insecure.NewCredentials(),
					TracerProvider:       tracerProvider,
					Propagation:          testCase.policy,
				},
			)
			if err != nil {
				t.Fatalf("grpcclient.New() error = %v", err)
			}
			t.Cleanup(func() { _ = connection.Close() })

			callerMetadata := metadata.MD{
				"traceparent":  {"00-11111111111111111111111111111111-2222222222222222-01"},
				"tracestate":   {"vendor=stale"},
				"baggage":      {"private=value"},
				"x-request-id": {"stale-request"},
				"authorization": {
					"Bearer retained",
				},
			}
			ctx := metadata.NewOutgoingContext(t.Context(), callerMetadata)
			ctx = reqctx.ContextWithRequestID(ctx, testCase.requestID)
			ctx, parent := tracerProvider.Tracer("grpcclient-propagation-test").Start(ctx, "parent")
			parentTraceID := parent.SpanContext().TraceID().String()
			endedBefore := len(recorder.Ended())

			client := healthgrpc.NewHealthClient(connection)
			if _, err := client.Check(ctx, &healthgrpc.HealthCheckRequest{}); err != nil {
				parent.End()
				t.Fatalf("Health.Check() error = %v", err)
			}
			assertWireCorrelation(
				t,
				<-unaryMetadata,
				parentTraceID,
				testCase.requestID,
				testCase.wantTrace,
				testCase.wantRequestID,
			)
			if got := len(recorder.Ended()); got <= endedBefore {
				parent.End()
				t.Fatalf("ended spans after unary call = %d, want more than %d", got, endedBefore)
			}

			watchCtx, cancelWatch := context.WithCancel(ctx)
			watch, err := client.Watch(watchCtx, &healthgrpc.HealthCheckRequest{})
			if err != nil {
				cancelWatch()
				parent.End()
				t.Fatalf("Health.Watch() error = %v", err)
			}
			if _, err := watch.Recv(); err != nil {
				cancelWatch()
				parent.End()
				t.Fatalf("Health.Watch().Recv() error = %v", err)
			}
			assertWireCorrelation(
				t,
				<-streamMetadata,
				parentTraceID,
				testCase.requestID,
				testCase.wantTrace,
				testCase.wantRequestID,
			)
			cancelWatch()
			parent.End()

			if got := callerMetadata.Get("traceparent"); len(got) != 1 ||
				got[0] != "00-11111111111111111111111111111111-2222222222222222-01" {
				t.Fatalf("caller traceparent = %v, want unchanged", got)
			}
			if got := callerMetadata.Get("baggage"); len(got) != 1 || got[0] != "private=value" {
				t.Fatalf("caller baggage = %v, want unchanged", got)
			}
		})
	}
}

func TestPropagationMetricAttributesExcludeCorrelationValues(t *testing.T) {
	unaryMetadata, _, target := startMetadataCaptureServer(t)
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("MeterProvider.Shutdown() error = %v", err)
		}
	})
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})

	connection, err := grpcclient.New(
		grpcclient.DefaultConfig(target),
		grpcclient.Options{
			TransportCredentials: insecure.NewCredentials(),
			MeterProvider:        meterProvider,
			TracerProvider:       tracerProvider,
			Propagation:          grpcclient.PropagationTrustedService,
		},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	const requestID = "metric_private_request_123"
	ctx := reqctx.ContextWithRequestID(t.Context(), requestID)
	ctx, parent := tracerProvider.Tracer("grpcclient-metric-test").Start(ctx, "metric-private-parent")
	traceID := parent.SpanContext().TraceID().String()
	if _, err := healthgrpc.NewHealthClient(connection).Check(
		ctx,
		&healthgrpc.HealthCheckRequest{},
	); err != nil {
		parent.End()
		t.Fatalf("Health.Check() error = %v", err)
	}
	parent.End()
	<-unaryMetadata

	assertGRPCMetricsDoNotContain(t, reader, requestID, traceID, "metric-private-parent")
}

func TestPerRPCCredentialsCannotOverrideCorrelationMetadata(t *testing.T) {
	serverCredentials, clientCredentials := grpcTestTLSCredentials(t)
	unaryMetadata, streamMetadata, target := startMetadataCaptureServer(
		t,
		grpc.Creds(serverCredentials),
	)
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("TracerProvider.Shutdown() error = %v", err)
		}
	})
	connection, err := grpcclient.New(
		grpcclient.DefaultConfig(target),
		grpcclient.Options{
			TransportCredentials: clientCredentials,
			TracerProvider:       tracerProvider,
			Propagation:          grpcclient.PropagationTrustedService,
		},
	)
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	credentialMetadata := map[string]string{
		"authorization": "Bearer retained",
		"TraceParent":   "stale-parent",
		"tracestate":    "stale-state",
		"BAGGAGE":       "private=value",
		"x-request-id":  "stale-request",
	}
	perRPC := livePerRPCCredentials{
		values:          credentialMetadata,
		requireSecurity: true,
	}
	const requestID = "credential_request_123"
	ctx := reqctx.ContextWithRequestID(t.Context(), requestID)
	ctx, parent := tracerProvider.Tracer("grpcclient-credential-test").Start(ctx, "parent")
	traceID := parent.SpanContext().TraceID().String()
	client := healthgrpc.NewHealthClient(connection)

	if _, err := client.Check(
		ctx,
		&healthgrpc.HealthCheckRequest{},
		grpc.PerRPCCredentials(perRPC),
	); err != nil {
		parent.End()
		t.Fatalf("Health.Check() error = %v", err)
	}
	assertWireCorrelation(t, <-unaryMetadata, traceID, requestID, true, true)

	watchCtx, cancelWatch := context.WithCancel(ctx)
	watch, err := client.Watch(
		watchCtx,
		&healthgrpc.HealthCheckRequest{},
		grpc.PerRPCCredentials(perRPC),
	)
	if err != nil {
		cancelWatch()
		parent.End()
		t.Fatalf("Health.Watch() error = %v", err)
	}
	if _, err := watch.Recv(); err != nil {
		cancelWatch()
		parent.End()
		t.Fatalf("Health.Watch().Recv() error = %v", err)
	}
	assertWireCorrelation(t, <-streamMetadata, traceID, requestID, true, true)
	cancelWatch()
	parent.End()

	if credentialMetadata["TraceParent"] != "stale-parent" ||
		credentialMetadata["authorization"] != "Bearer retained" {
		t.Fatalf("credential metadata was mutated: %v", credentialMetadata)
	}
}

func TestPerRPCCredentialsSecurityAndErrorFailuresReachNoHandler(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		credential livePerRPCCredentials
		want       string
	}{
		{
			name: "transport security required",
			credential: livePerRPCCredentials{
				values:          map[string]string{"authorization": "Bearer retained"},
				requireSecurity: true,
			},
			want: "secure credentials",
		},
		{
			name: "credential error",
			credential: livePerRPCCredentials{
				err: errors.New("sentinel credential lookup failure"),
			},
			want: "sentinel credential lookup failure",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			unaryMetadata, _, target := startMetadataCaptureServer(t)
			connection, err := grpcclient.New(
				grpcclient.DefaultConfig(target),
				grpcclient.Options{
					TransportCredentials: insecure.NewCredentials(),
					Propagation:          grpcclient.PropagationTrustedService,
				},
			)
			if err != nil {
				t.Fatalf("grpcclient.New() error = %v", err)
			}
			t.Cleanup(func() { _ = connection.Close() })

			callCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			_, err = healthgrpc.NewHealthClient(connection).Check(
				callCtx,
				&healthgrpc.HealthCheckRequest{},
				grpc.PerRPCCredentials(testCase.credential),
			)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(testCase.want)) {
				t.Fatalf("Health.Check() error = %v, want containing %q", err, testCase.want)
			}
			select {
			case got := <-unaryMetadata:
				t.Fatalf("handler observed metadata after credential failure: %v", got)
			default:
			}
		})
	}
}

func assertWireCorrelation(
	t *testing.T,
	values metadata.MD,
	traceID string,
	requestID string,
	wantTrace bool,
	wantRequestID bool,
) {
	t.Helper()

	traceparents := values.Get("traceparent")
	if (len(traceparents) == 1) != wantTrace {
		t.Fatalf("traceparent = %v, want present %v", traceparents, wantTrace)
	}
	if wantTrace && len(traceparents[0]) < len(traceID) ||
		wantTrace && traceparents[0][3:3+len(traceID)] != traceID {
		t.Fatalf("traceparent = %v, want trace ID %s", traceparents, traceID)
	}
	requestIDs := values.Get("x-request-id")
	if (len(requestIDs) == 1) != wantRequestID {
		t.Fatalf("x-request-id = %v, want present %v", requestIDs, wantRequestID)
	}
	if wantRequestID && requestIDs[0] != requestID {
		t.Fatalf("x-request-id = %v, want %q", requestIDs, requestID)
	}
	for _, forbidden := range []string{"tracestate", "baggage"} {
		if got := values.Get(forbidden); len(got) != 0 {
			t.Fatalf("%s = %v, want absent", forbidden, got)
		}
	}
	if got := values.Get("authorization"); len(got) != 1 || got[0] != "Bearer retained" {
		t.Fatalf("authorization = %v, want retained", got)
	}
}

func startMetadataCaptureServer(
	t *testing.T,
	serverOptions ...grpc.ServerOption,
) (<-chan metadata.MD, <-chan metadata.MD, string) {
	t.Helper()

	unaryMetadata := make(chan metadata.MD, 1)
	streamMetadata := make(chan metadata.MD, 1)
	serverOptions = append(serverOptions,
		grpc.UnaryInterceptor(func(
			ctx context.Context,
			request any,
			_ *grpc.UnaryServerInfo,
			handler grpc.UnaryHandler,
		) (any, error) {
			incoming, _ := metadata.FromIncomingContext(ctx)
			unaryMetadata <- incoming.Copy()
			return handler(ctx, request)
		}),
		grpc.StreamInterceptor(func(
			service any,
			stream grpc.ServerStream,
			_ *grpc.StreamServerInfo,
			handler grpc.StreamHandler,
		) error {
			incoming, _ := metadata.FromIncomingContext(stream.Context())
			streamMetadata <- incoming.Copy()
			return handler(service, stream)
		}),
	)
	server := grpc.NewServer(serverOptions...)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(server, healthServer)

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		if err := <-serveDone; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})

	return unaryMetadata, streamMetadata, "passthrough:///" + listener.Addr().String()
}

type livePerRPCCredentials struct {
	values          map[string]string
	err             error
	requireSecurity bool
}

func (c livePerRPCCredentials) GetRequestMetadata(
	context.Context,
	...string,
) (map[string]string, error) {
	return c.values, c.err
}

func (c livePerRPCCredentials) RequireTransportSecurity() bool {
	return c.requireSecurity
}

func grpcTestTLSCredentials( //nolint:ireturn // grpc-go exposes transport credentials as an interface.
	t *testing.T,
) (credentials.TransportCredentials, credentials.TransportCredentials) {
	t.Helper()

	certificateSource := httptest.NewTLSServer(http.NotFoundHandler())
	serverCertificate := certificateSource.TLS.Certificates[0]
	leafCertificate := certificateSource.Certificate()
	certificateSource.Close()

	roots := x509.NewCertPool()
	roots.AddCert(leafCertificate)
	return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{serverCertificate},
			MinVersion:   tls.VersionTLS12,
		}), credentials.NewTLS(&tls.Config{
			RootCAs:    roots,
			ServerName: "example.com",
			MinVersion: tls.VersionTLS12,
		})
}

func assertGRPCMetricsDoNotContain(
	t *testing.T,
	reader *sdkmetric.ManualReader,
	forbidden ...string,
) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &resourceMetrics); err != nil {
		t.Fatalf("ManualReader.Collect() error = %v", err)
	}
	points := 0
	for _, scope := range resourceMetrics.ScopeMetrics {
		for _, metricValue := range scope.Metrics {
			switch data := metricValue.Data.(type) {
			case metricdata.Histogram[float64]:
				for _, point := range data.DataPoints {
					points++
					assertGRPCAttributesDoNotContain(t, point.Attributes.ToSlice(), forbidden)
				}
			case metricdata.Histogram[int64]:
				for _, point := range data.DataPoints {
					points++
					assertGRPCAttributesDoNotContain(t, point.Attributes.ToSlice(), forbidden)
				}
			case metricdata.Sum[int64]:
				for _, point := range data.DataPoints {
					points++
					assertGRPCAttributesDoNotContain(t, point.Attributes.ToSlice(), forbidden)
				}
			}
		}
	}
	if points == 0 {
		t.Fatal("gRPC client metric data points = 0, want recorded metrics")
	}
}

func assertGRPCAttributesDoNotContain(
	t *testing.T,
	attributes []attribute.KeyValue,
	forbidden []string,
) {
	t.Helper()

	for _, attr := range attributes {
		for _, candidate := range forbidden {
			if candidate != "" &&
				(strings.Contains(string(attr.Key), candidate) ||
					strings.Contains(attr.Value.String(), candidate)) {
				t.Fatalf(
					"metric attribute %s=%q contains forbidden correlation value %q",
					attr.Key,
					attr.Value.String(),
					candidate,
				)
			}
		}
	}
}
