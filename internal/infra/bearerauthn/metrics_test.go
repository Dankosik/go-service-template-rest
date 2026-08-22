package bearerauthn

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	// profile:grpc:start
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	// profile:grpc:end
)

func TestVerificationMetricVocabulary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		headers    []string
		err        error
		wantResult string
		wantReason string
	}{
		{name: "success", headers: []string{"Bearer token"}, wantResult: "success"},
		{name: "missing", wantResult: "failure", wantReason: "missing"},
		{name: "malformed", headers: []string{"Basic token"}, wantResult: "failure", wantReason: "malformed"},
		{name: "oversize", headers: []string{"Bearer " + strings.Repeat("x", MaxTokenBytes+1)}, wantResult: "failure", wantReason: "oversize"},
		{name: "invalid", headers: []string{"Bearer token"}, err: NewError(KindInvalid), wantResult: "failure", wantReason: "invalid"},
		{name: "unavailable", headers: []string{"Bearer token"}, err: NewError(KindUnavailable), wantResult: "failure", wantReason: "unavailable"},
		{name: "canceled", headers: []string{"Bearer token"}, err: fmt.Errorf("wait: %w", context.Canceled), wantResult: "failure", wantReason: "invalid"},
		{name: "deadline", headers: []string{"Bearer token"}, err: fmt.Errorf("wait: %w", context.DeadlineExceeded), wantResult: "failure", wantReason: "invalid"},
	}

	transports := []string{"http"}
	// profile:grpc:start
	transports = append(transports, "grpc")
	// profile:grpc:end

	for _, transportName := range transports {
		for _, testCase := range cases {
			t.Run(transportName+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
				t.Cleanup(func() { _ = provider.Shutdown(t.Context()) })
				runtime, err := New(&fakeVerifier{err: testCase.err}, provider)
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				switch transportName {
				case "http":
					request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
					for _, value := range testCase.headers {
						request.Header.Add("Authorization", value)
					}
					_, _ = runtime.ResolveHTTP(t.Context(), bearerAuthInput(request))
				// profile:grpc:start
				case "grpc":
					ctx := t.Context()
					if len(testCase.headers) > 0 {
						ctx = metadata.NewIncomingContext(ctx, metadata.MD{"authorization": testCase.headers})
					}
					_, _ = runtime.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc.API/Call"},
						func(context.Context, any) (any, error) { return struct{}{}, nil })
				// profile:grpc:end
				default:
					t.Fatalf("unexpected transport %q", transportName)
				}

				points := collectVerificationPoints(t, reader)
				if len(points) != 1 {
					t.Fatalf("verification points = %d, want 1", len(points))
				}
				attrs := points[0].Attributes
				gotTransport, _ := attrs.Value(attribute.Key("authn.transport"))
				gotResult, _ := attrs.Value(attribute.Key("authn.result"))
				if gotTransport.AsString() != transportName || gotResult.AsString() != testCase.wantResult {
					t.Fatalf("transport/result = %s/%s, want %s/%s", gotTransport.AsString(), gotResult.AsString(), transportName, testCase.wantResult)
				}
				reason, present := attrs.Value(attribute.Key("authn.reason"))
				if testCase.wantReason == "" {
					if present {
						t.Fatalf("unexpected reason %q", reason.AsString())
					}
				} else if !present || reason.AsString() != testCase.wantReason {
					t.Fatalf("reason = %q present=%v, want %q", reason.AsString(), present, testCase.wantReason)
				}
				for _, labelled := range attrs.ToSlice() {
					if labelled.Key != "authn.transport" && labelled.Key != "authn.result" && labelled.Key != "authn.reason" {
						t.Fatalf("unexpected attribute %s", labelled.Key)
					}
					if strings.Contains(labelled.Value.AsString(), "canary") ||
						strings.Contains(labelled.Value.AsString(), "poison") {
						t.Fatalf("metric leaked canary %q", labelled.Value.AsString())
					}
				}
			})
		}
	}
}

func collectVerificationPoints(t *testing.T, reader *sdkmetric.ManualReader) []metricdata.DataPoint[int64] {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	var points []metricdata.DataPoint[int64]
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			if measured.Name == "authn.jwks.refresh_failures" {
				t.Fatal("introspection-specific or JWT refresh metric published by shared runtime")
			}
			if measured.Name != "authn.verifications" {
				continue
			}
			sum, ok := measured.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("authn.verifications aggregation = %T, want sum", measured.Data)
			}
			points = append(points, sum.DataPoints...)
		}
	}
	return points
}
