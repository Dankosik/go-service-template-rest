// Correlation trust boundary, wire half: of the sources that can put a reserved
// key on an outgoing RPC, does only what PropagationPolicy selected actually
// arrive at the server?
//
// It runs real RPCs against a metadata-capture server, so it asserts what
// crossed the boundary rather than which strip seam removed it.
//
// propagation_internal_test.go drives those seams directly, and the third
// source — resolver-supplied address metadata — has its own three files
// starting at resolver_internal_test.go.

package grpcclient_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/grpcclient"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func TestPropagationPoliciesApplyToUnaryAndStreamingRPCs(t *testing.T) {
	unaryMetadata, streamMetadata, target := startMetadataCaptureServer(t)
	recorder, tracerProvider := telemetrytest.NewRecordingTracerProvider(t)

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
			cfg := grpcclient.DefaultConfig(target)
			cfg.HealthCheck = false
			connection, err := grpcclient.New(
				cfg,
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
			assertWireCorrelation(t, "unary", <-unaryMetadata, wireCorrelation{
				traceID:       parentTraceID,
				requestID:     testCase.requestID,
				wantTrace:     testCase.wantTrace,
				wantRequestID: testCase.wantRequestID,
			})
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
			assertWireCorrelation(t, "stream", <-streamMetadata, wireCorrelation{
				traceID:       parentTraceID,
				requestID:     testCase.requestID,
				wantTrace:     testCase.wantTrace,
				wantRequestID: testCase.wantRequestID,
			})
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
	reader, meterProvider := telemetrytest.NewManualMeterProvider(t)
	_, tracerProvider := telemetrytest.NewRecordingTracerProvider(t)

	cfg := grpcclient.DefaultConfig(target)
	cfg.HealthCheck = false
	connection, err := grpcclient.New(
		cfg,
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

	telemetrytest.AssertNoAttributeContains(t, reader, requestID, traceID, "metric-private-parent")
}

func TestPerRPCCredentialsCannotOverrideCorrelationMetadata(t *testing.T) {
	serverCredentials, clientCredentials := testTLSCredentials(t)
	unaryMetadata, streamMetadata, target := startMetadataCaptureServer(
		t,
		grpc.Creds(serverCredentials),
	)
	_, tracerProvider := telemetrytest.NewRecordingTracerProvider(t)
	cfg := grpcclient.DefaultConfig(target)
	cfg.HealthCheck = false
	connection, err := grpcclient.New(
		cfg,
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
	credentialCorrelation := wireCorrelation{
		traceID:       traceID,
		requestID:     requestID,
		wantTrace:     true,
		wantRequestID: true,
	}
	assertWireCorrelation(t, "credential unary", <-unaryMetadata, credentialCorrelation)

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
	assertWireCorrelation(t, "credential stream", <-streamMetadata, credentialCorrelation)
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
			cfg := grpcclient.DefaultConfig(target)
			cfg.HealthCheck = false
			connection, err := grpcclient.New(
				cfg,
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

// wireCorrelation is what one RPC's metadata should carry once it reaches the
// server: the caller's trace, its accepted request ID, or neither.
type wireCorrelation struct {
	traceID       string
	requestID     string
	wantTrace     bool
	wantRequestID bool
}

// assertWireCorrelation owns the whole rule for what may cross this client's
// trust boundary — the selected trace and request ID present, tracestate and
// baggage always absent, and an unrelated header untouched — and returns the
// traceparent it accepted.
//
// It is the only copy of that rule, so a fourth reserved key is one edit rather
// than one per proof. label names the RPC for a caller driving more than one:
// transparent_retry_test.go passes an attempt number and parses the returned
// traceparent for the per-attempt span identity only it asks about.
func assertWireCorrelation(
	t *testing.T,
	label string,
	values metadata.MD,
	want wireCorrelation,
) string {
	t.Helper()

	traceparents := values.Get("traceparent")
	if (len(traceparents) == 1) != want.wantTrace {
		t.Fatalf("%s traceparent = %v, want present %v", label, traceparents, want.wantTrace)
	}
	traceparent := ""
	if want.wantTrace {
		traceparent = traceparents[0]
		if !strings.Contains(traceparent, want.traceID) {
			t.Fatalf("%s traceparent = %q, want trace ID %s", label, traceparent, want.traceID)
		}
	}
	requestIDs := values.Get("x-request-id")
	if (len(requestIDs) == 1) != want.wantRequestID {
		t.Fatalf("%s x-request-id = %v, want present %v", label, requestIDs, want.wantRequestID)
	}
	if want.wantRequestID && requestIDs[0] != want.requestID {
		t.Fatalf("%s x-request-id = %v, want %q", label, requestIDs, want.requestID)
	}
	for _, forbidden := range []string{"tracestate", "baggage"} {
		if got := values.Get(forbidden); len(got) != 0 {
			t.Fatalf("%s %s = %v, want absent", label, forbidden, got)
		}
	}
	if got := values.Get("authorization"); len(got) != 1 || got[0] != "Bearer retained" {
		t.Fatalf("%s authorization = %v, want retained", label, got)
	}
	return traceparent
}

// livePerRPCCredentials is field for field the staticPerRPCCredentials that
// propagation_internal_test.go declares. Neither is a variant of the other: that
// file is package grpcclient and this one is grpcclient_test, so an unexported
// type cannot cross between them. The names differ only to say which side of the
// boundary each drives — this one reaches a real server, that one a seam.
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
