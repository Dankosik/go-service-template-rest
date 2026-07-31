package httpclient

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestHTTPPropagationPolicy(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)

	tests := []struct {
		name      string
		policy    PropagationPolicy
		requestID string
		wantTrace bool
		wantID    string
	}{
		{name: "none", policy: PropagationNone, requestID: "request-none"},
		{
			name:      "trace context",
			policy:    PropagationTraceContext,
			requestID: "request-trace",
			wantTrace: true,
		},
		{
			name:      "trusted service",
			policy:    PropagationTrustedService,
			requestID: "request-trusted",
			wantTrace: true,
			wantID:    "request-trusted",
		},
		{
			name:      "trusted service invalid request ID",
			policy:    PropagationTrustedService,
			requestID: "invalid request id",
			wantTrace: true,
		},
		{
			name:      "trusted service missing request ID",
			policy:    PropagationTrustedService,
			wantTrace: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if err := meterProvider.Shutdown(shutdownCtx); err != nil {
					t.Errorf("shutdown metric provider: %v", err)
				}
			})

			observations := make(chan propagationWireObservation, 1)
			client := newPropagationTestClient(
				t,
				test.policy,
				RetryPolicy{},
				meterProvider,
				http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					_, _ = io.Copy(io.Discard, request.Body)
					observations <- propagationWireObservation{
						header:  request.Header.Clone(),
						trailer: request.Trailer.Clone(),
					}
					response.WriteHeader(http.StatusNoContent)
				}),
			)

			ctx := reqctx.ContextWithRequestID(context.Background(), test.requestID)
			ctx, parent := otel.Tracer("httpclient.propagation.test").Start(ctx, "parent")
			before := len(recorder.Ended())
			request, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				client.BaseURL()+"?canary=query-metric-secret",
				strings.NewReader("policy body"),
			)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			seedStaleHTTPPropagation(request.Header)
			request.ContentLength = -1
			request.Trailer = make(http.Header)
			seedStaleHTTPPropagation(request.Trailer)
			request.Trailer.Set("X-Allowed-Trailer", "retained")
			originalHeader := request.Header.Clone()
			originalTrailer := request.Trailer.Clone()

			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			if err := response.Body.Close(); err != nil {
				t.Fatalf("response Body.Close() error = %v", err)
			}
			parent.End()

			wire := <-observations
			assertHTTPPolicyHeader(t, wire.header, test.wantTrace, test.wantID)
			assertNoHTTPPropagationFields(t, wire.trailer)
			assertAllowedHTTPTrailer(t, wire.trailer)
			if !reflect.DeepEqual(request.Header, originalHeader) {
				t.Fatalf("caller header mutated:\ngot  %#v\nwant %#v", request.Header, originalHeader)
			}
			if !reflect.DeepEqual(request.Trailer, originalTrailer) {
				t.Fatalf("caller trailer mutated:\ngot  %#v\nwant %#v", request.Trailer, originalTrailer)
			}

			clientSpans := clientSpansSince(recorder, before)
			if len(clientSpans) != 1 {
				t.Fatalf("client spans = %d, want 1", len(clientSpans))
			}
			if got, want := clientSpans[0].SpanContext().TraceID(), parent.SpanContext().TraceID(); got != want {
				t.Fatalf("client trace ID = %s, want parent trace %s", got, want)
			}
			if test.wantTrace {
				remote := remoteSpanContext(wire.header)
				if got, want := remote.TraceID(), clientSpans[0].SpanContext().TraceID(); got != want {
					t.Fatalf("wire trace ID = %s, want client attempt %s", got, want)
				}
				if got, want := remote.SpanID(), clientSpans[0].SpanContext().SpanID(); got != want {
					t.Fatalf("wire parent span ID = %s, want client attempt %s", got, want)
				}
			}
			assertHTTPMetricsDoNotContain(
				t,
				reader,
				append(
					[]string{
						test.requestID,
						"stale-request-id",
						"stale-baggage",
						"query-metric-secret",
						clientSpans[0].SpanContext().TraceID().String(),
						clientSpans[0].SpanContext().SpanID().String(),
					},
					append(headerValues(wire.header), headerValues(wire.trailer)...)...,
				)...,
			)
		})
	}
}

func TestHTTPPropagationPolicyRetriesEveryAttempt(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)

	for _, test := range []struct {
		name      string
		policy    PropagationPolicy
		wantTrace bool
		wantID    string
	}{
		{name: "none", policy: PropagationNone},
		{name: "trace context", policy: PropagationTraceContext, wantTrace: true},
		{
			name:      "trusted service",
			policy:    PropagationTrustedService,
			wantTrace: true,
			wantID:    "retry-request",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var attempts atomic.Int32
			observations := make(chan propagationWireObservation, 2)
			client := newPropagationTestClient(
				t,
				test.policy,
				RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond},
				nil,
				http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					_, _ = io.Copy(io.Discard, request.Body)
					observations <- propagationWireObservation{
						header:  request.Header.Clone(),
						trailer: request.Trailer.Clone(),
					}
					if attempts.Add(1) == 1 {
						response.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					response.WriteHeader(http.StatusNoContent)
				}),
			)

			ctx := reqctx.ContextWithRequestID(context.Background(), "retry-request")
			ctx, parent := otel.Tracer("httpclient.retry.propagation.test").Start(ctx, "parent")
			before := len(recorder.Ended())
			request, err := http.NewRequestWithContext(
				ctx,
				http.MethodGet,
				client.BaseURL(),
				strings.NewReader("retry body"),
			)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			seedStaleHTTPPropagation(request.Header)
			request.ContentLength = -1
			request.Trailer = make(http.Header)
			seedStaleHTTPPropagation(request.Trailer)
			request.Trailer.Set("X-Allowed-Trailer", "retained")
			originalHeader := request.Header.Clone()
			originalTrailer := request.Trailer.Clone()

			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			_, _ = io.Copy(io.Discard, response.Body)
			if err := response.Body.Close(); err != nil {
				t.Fatalf("response Body.Close() error = %v", err)
			}
			parent.End()

			if got := attempts.Load(); got != 2 {
				t.Fatalf("attempts = %d, want 2", got)
			}
			if !reflect.DeepEqual(request.Header, originalHeader) {
				t.Fatalf("caller header mutated:\ngot  %#v\nwant %#v", request.Header, originalHeader)
			}
			if !reflect.DeepEqual(request.Trailer, originalTrailer) {
				t.Fatalf("caller trailer mutated:\ngot  %#v\nwant %#v", request.Trailer, originalTrailer)
			}
			clientSpans := clientSpansSince(recorder, before)
			if len(clientSpans) != 2 {
				t.Fatalf("client spans = %d, want 2", len(clientSpans))
			}

			spanIDs := make(map[trace.SpanID]struct{}, 2)
			for _, span := range clientSpans {
				if got, want := span.SpanContext().TraceID(), parent.SpanContext().TraceID(); got != want {
					t.Fatalf("client trace ID = %s, want parent trace %s", got, want)
				}
				spanIDs[span.SpanContext().SpanID()] = struct{}{}
			}
			wireSpanIDs := make(map[trace.SpanID]struct{}, 2)
			for attempt := 1; attempt <= 2; attempt++ {
				wire := <-observations
				assertHTTPPolicyHeader(t, wire.header, test.wantTrace, test.wantID)
				assertNoHTTPPropagationFields(t, wire.trailer)
				assertAllowedHTTPTrailer(t, wire.trailer)
				if test.wantTrace {
					remote := remoteSpanContext(wire.header)
					if got, want := remote.TraceID(), parent.SpanContext().TraceID(); got != want {
						t.Fatalf("attempt %d wire trace ID = %s, want %s", attempt, got, want)
					}
					if _, ok := spanIDs[remote.SpanID()]; !ok {
						t.Fatalf(
							"attempt %d wire parent span ID = %s, want one of client attempts %v",
							attempt,
							remote.SpanID(),
							spanIDs,
						)
					}
					wireSpanIDs[remote.SpanID()] = struct{}{}
				}
			}
			if len(spanIDs) != 2 {
				t.Fatalf("distinct client attempt span IDs = %d, want 2", len(spanIDs))
			}
			if test.wantTrace && len(wireSpanIDs) != 2 {
				t.Fatalf("distinct wire parent span IDs = %d, want 2", len(wireSpanIDs))
			}
		})
	}
}

func TestNewRejectsUnknownPropagationPolicy(t *testing.T) {
	t.Parallel()

	cfg := validExternalConfig()
	cfg.Propagation = PropagationPolicy(255)
	if _, err := New(cfg, nil); err == nil {
		t.Fatal("New() error = nil, want unknown propagation policy rejected")
	}
}

func newPropagationTestClient(
	t *testing.T,
	policy PropagationPolicy,
	retry RetryPolicy,
	meterProvider metric.MeterProvider,
	handler http.Handler,
) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split test server address: %v", err)
	}

	cfg := validExternalConfig()
	cfg.BaseURL = "http://provider.railway.internal:" + port
	cfg.TargetClass = PrivateHTTP
	cfg.PrivateHostSuffix = "railway.internal"
	cfg.Propagation = policy
	cfg.Retry = retry
	client, err := New(cfg, meterProvider)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	var dialer net.Dialer
	client.transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, server.Listener.Addr().String())
	}
	return client
}

func seedStaleHTTPPropagation(header http.Header) {
	header["tRaCePaReNt"] = []string{"00-00000000000000000000000000000001-0000000000000001-01"}
	header["TrAcEsTaTe"] = []string{"vendor=stale"}
	header["BaGgAgE"] = []string{"secret=stale-baggage"}
	header["x-ReQuEsT-iD"] = []string{"stale-request-id"}
}

type propagationWireObservation struct {
	header  http.Header
	trailer http.Header
}

func assertHTTPPolicyHeader(t *testing.T, header http.Header, wantTrace bool, wantRequestID string) {
	t.Helper()

	traceparent := headerValueEqualFold(header, "traceparent")
	if wantTrace && traceparent == "" {
		t.Fatal("wire traceparent is empty")
	}
	if !wantTrace && traceparent != "" {
		t.Fatalf("wire traceparent = %q, want absent", traceparent)
	}
	if got := headerValueEqualFold(header, "baggage"); got != "" {
		t.Fatalf("wire baggage = %q, want absent", got)
	}
	if got := headerValueEqualFold(header, requestIDHeader); got != wantRequestID {
		t.Fatalf("wire request ID = %q, want %q", got, wantRequestID)
	}
	if got := headerValueEqualFold(header, "tracestate"); got == "vendor=stale" {
		t.Fatalf("wire tracestate preserved stale caller value %q", got)
	}
}

func assertNoHTTPPropagationFields(t *testing.T, values http.Header) {
	t.Helper()

	for _, reserved := range reservedPropagationHeaders {
		if got := headerValueEqualFold(values, reserved); got != "" {
			t.Fatalf("wire trailer %s = %q, want absent", reserved, got)
		}
	}
}

func assertAllowedHTTPTrailer(t *testing.T, values http.Header) {
	t.Helper()

	if got := values.Get("X-Allowed-Trailer"); got != "retained" {
		t.Fatalf("wire allowed trailer = %q, want retained", got)
	}
}

func headerValueEqualFold(header http.Header, name string) string {
	for key, values := range header {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func headerValues(header http.Header) []string {
	var values []string
	for _, entries := range header {
		values = append(values, entries...)
	}
	return values
}

func remoteSpanContext(header http.Header) trace.SpanContext {
	ctx := propagation.TraceContext{}.Extract(context.Background(), propagation.HeaderCarrier(header))
	return trace.SpanContextFromContext(ctx)
}

func clientSpansSince(
	recorder interface {
		Ended() []sdktrace.ReadOnlySpan
	},
	start int,
) []sdktrace.ReadOnlySpan {
	var spans []sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended()[start:] {
		if span.SpanKind() == trace.SpanKindClient {
			spans = append(spans, span)
		}
	}
	return spans
}

func assertHTTPMetricsDoNotContain(
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
					assertAttributesDoNotContain(t, point.Attributes.ToSlice(), forbidden)
				}
			case metricdata.Histogram[int64]:
				for _, point := range data.DataPoints {
					points++
					assertAttributesDoNotContain(t, point.Attributes.ToSlice(), forbidden)
				}
			}
		}
	}
	if points == 0 {
		t.Fatal("HTTP client metric data points = 0, want recorded metrics")
	}
}

func assertAttributesDoNotContain(t *testing.T, attributes []attribute.KeyValue, forbidden []string) {
	t.Helper()

	for _, attr := range attributes {
		value := attr.Value.String()
		for _, candidate := range forbidden {
			if candidate != "" && (strings.Contains(string(attr.Key), candidate) || strings.Contains(value, candidate)) {
				t.Fatalf(
					"metric attribute %s=%q contains forbidden correlation value %q",
					attr.Key,
					value,
					candidate,
				)
			}
		}
	}
}
