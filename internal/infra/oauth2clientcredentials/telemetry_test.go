package oauth2clientcredentials

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"

	// profile:outbound-auth-http:start
	"net/http"
	// profile:outbound-auth-http:end
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	// profile:outbound-auth-http:start
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"go.opentelemetry.io/otel"

	// profile:outbound-auth-http:end
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOutboundAuthTelemetryIsCompleteAndBounded(t *testing.T) {
	reader, meterProvider := telemetrytest.NewManualMeterProvider(t)
	// profile:outbound-auth-http:start
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		fixture := newHTTPFixtureWithMeter(t, newMovableClock(fixedProviderTime), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}), []acquisitionStep{{token: accessToken{value: "token", expiresAt: fixedProviderTime.Add(time.Minute)}}}, httpclient.RetryPolicy{}, meterProvider)
		response, err := fixture.authenticated.Do(fixture.request(t.Context(), t))
		if err != nil {
			t.Fatalf("HTTPClient.Do(%d) error = %v", status, err)
		}
		closeResponse(t, response)
		if _, err := fixture.client.resolve(t.Context()); err != nil {
			t.Fatalf("cache resolve after HTTP rejection: %v", err)
		}
	}
	// profile:outbound-auth-http:end
	// profile:outbound-auth-grpc:start
	assertGRPCDownstreamAuthStatus(t, meterProvider)
	// profile:outbound-auth-grpc:end
	for _, class := range []FailureClass{
		FailureInvalidConfiguration, FailureEndpointTrust, FailureProviderTimeout, FailureProviderUnavailable,
		FailureClientRejected, FailureGrantRejected, FailureUnsupportedResponse, FailureTokenUnusable,
	} {
		client := requireTestClient(t, validTestConfig(), testClientOptions{
			now:           func() time.Time { return fixedProviderTime },
			acquire:       func(context.Context) (accessToken, error) { return accessToken{}, failure(class) },
			meterProvider: meterProvider,
		})
		if _, err := client.resolve(t.Context()); err == nil {
			t.Fatalf("resolve() for %s returned nil", class)
		}
	}
	canceled := requireTestClient(t, validTestConfig(), testClientOptions{meterProvider: meterProvider, acquire: func(context.Context) (accessToken, error) {
		return accessToken{}, failure(FailureProviderUnavailable)
	}})
	if _, err := canceled.resolve(canceledContext(context.Canceled)); err == nil {
		t.Fatal("canceled resolve returned nil")
	}
	retired := requireTestClient(t, validTestConfig(), testClientOptions{meterProvider: meterProvider, acquire: func(context.Context) (accessToken, error) {
		return accessToken{}, failure(FailureProviderUnavailable)
	}})
	if err := retired.Close(t.Context()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := retired.resolve(t.Context()); err == nil {
		t.Fatal("retired resolve returned nil")
	}

	metrics := outboundAuthMetrics(t, reader)
	if len(metrics) != 4 {
		t.Fatalf("outbound auth metrics = %v, want exactly four instruments", metrics)
	}
	rejections := telemetrytest.Int64Sum(t, requireMetric(t, metrics, resourceRejectionsInstrument, "{rejection}")).DataPoints
	var transports []string
	//nolint:gocritic // Each transport remains independently removable by its profile marker.
	// profile:outbound-auth-http:start
	transports = append(transports, transportHTTP)
	// profile:outbound-auth-http:end
	// profile:outbound-auth-grpc:start
	transports = append(transports, transportGRPC)
	// profile:outbound-auth-grpc:end
	if want := len(transports) * 2; len(rejections) != want {
		t.Fatalf("resource rejection points = %d, want %d", len(rejections), want)
	}
	for _, transport := range transports {
		for _, result := range []string{resultUnauthenticated, resultForbidden} {
			found := false
			for _, point := range rejections {
				if telemetrytest.Attribute(t, point.Attributes, attributeTransport) == transport &&
					telemetrytest.Attribute(t, point.Attributes, attributeResult) == result {
					if point.Value != 1 {
						t.Fatalf("%s/%s count = %d, want 1", transport, result, point.Value)
					}
					requirePointSet(t, point.Attributes, map[string]string{
						attributeDependency: validTestConfig().DependencyName,
						attributeTransport:  transport,
						attributeResult:     result,
					})
					found = true
				}
			}
			if !found {
				t.Fatalf("missing resource rejection %s/%s", transport, result)
			}
		}
	}
	requireClosedResolutionAndProviderMatrix(t, metrics)
}

// profile:outbound-auth-http:start
func TestOutboundAuthForbiddenValuesNeverReachSignals(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)
	reader, meterProvider := telemetrytest.NewManualMeterProvider(t)
	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, nil))
	clock := newMovableClock(fixedProviderTime)
	fixture := newHTTPFixtureWithTelemetry(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}), []acquisitionStep{{token: accessToken{value: forbiddenCanary, expiresAt: fixedProviderTime.Add(time.Minute)}}}, httpclient.RetryPolicy{}, meterProvider, log)
	ctx, span := otel.Tracer("outbound-auth-canary").Start(t.Context(), forbiddenCanary)
	defer span.End()
	request := fixture.request(ctx, t)
	request.Header.Set("Traceparent", forbiddenCanary)
	request.Header.Set("Baggage", forbiddenCanary)
	request.Header.Set("x-request-id", forbiddenCanary)
	response, err := fixture.authenticated.Do(request)
	if err != nil {
		t.Fatalf("HTTPClient.Do() error = %v", err)
	}
	closeResponse(t, response)
	providerHeaders := <-fixture.providerHeaders
	for _, name := range []string{"traceparent", "baggage", "x-request-id"} {
		if value := providerHeaders.Get(name); value != "" {
			t.Fatalf("provider %s = %q, want empty", name, value)
		}
	}
	for _, span := range recorder.Ended() {
		contents := fmt.Sprint(span.Name(), span.Attributes(), span.Events())
		if strings.Contains(contents, forbiddenCanary) || strings.Contains(contents, testTokenHost) {
			t.Fatal("credential or provider value reached an HTTP span")
		}
	}
	for _, measured := range outboundAuthMetrics(t, reader) {
		if strings.Contains(fmt.Sprint(measured), forbiddenCanary) {
			t.Fatal("credential canary reached a metric")
		}
	}
	if strings.Contains(output.String(), forbiddenCanary) || strings.Contains(output.String(), testTokenHost) {
		t.Fatal("credential or provider value reached a log")
	}
	if err := failure(FailureProviderUnavailable); strings.Contains(err.Error(), forbiddenCanary) {
		t.Fatal("provider canary reached an auth error")
	}
}

// profile:outbound-auth-http:end

func TestOutboundAuthProviderOutageIsOperationLocal(t *testing.T) {
	reader, meterProvider := telemetrytest.NewManualMeterProvider(t)
	provider := &scriptedAcquirer{steps: []acquisitionStep{{
		err: failure(FailureProviderUnavailable),
	}}}
	client := requireTestClient(t, validTestConfig(), testClientOptions{
		now:           func() time.Time { return fixedProviderTime },
		acquire:       provider.acquire,
		meterProvider: meterProvider,
	})

	if _, err := client.resolve(t.Context()); err == nil {
		t.Fatal("resolve() during provider outage returned no error")
	} else {
		assertFailureClass(t, err, FailureProviderUnavailable)
	}
	if got := provider.Calls(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}

	metrics := outboundAuthMetrics(t, reader)
	requireCounterPoint(t, metrics, tokenResolutionsInstrument, "{resolution}", 1, map[string]string{
		attributeDependency:   validTestConfig().DependencyName,
		attributeSource:       sourceAcquisition,
		attributeResult:       resultFailure,
		attributeFailureClass: string(FailureProviderUnavailable),
	})
	requireCounterPoint(t, metrics, providerAttemptsInstrument, "{attempt}", 1, map[string]string{
		attributeDependency:   validTestConfig().DependencyName,
		attributeResult:       resultFailure,
		attributeFailureClass: string(FailureProviderUnavailable),
	})
	requireHistogramPoint(t, metrics, providerDurationInstrument, "s", map[string]string{
		attributeDependency:   validTestConfig().DependencyName,
		attributeResult:       resultFailure,
		attributeFailureClass: string(FailureProviderUnavailable),
	})
}

func TestTelemetryRecordsCacheAndAcquisition(t *testing.T) {
	reader, meterProvider := telemetrytest.NewManualMeterProvider(t)
	provider := &scriptedAcquirer{steps: []acquisitionStep{{
		token: accessToken{value: "token", expiresAt: fixedProviderTime.Add(time.Minute)},
	}}}
	client := requireTestClient(t, validTestConfig(), testClientOptions{
		now:           func() time.Time { return fixedProviderTime },
		acquire:       provider.acquire,
		meterProvider: meterProvider,
	})
	requireOperationToken(t, client)
	requireOperationToken(t, client)

	metrics := outboundAuthMetrics(t, reader)
	resolution := requireMetric(t, metrics, tokenResolutionsInstrument, "{resolution}")
	points := telemetrytest.Int64Sum(t, resolution).DataPoints
	if len(points) != 2 {
		t.Fatalf("resolution points = %d, want acquisition and cache", len(points))
	}
	wantSources := map[string]bool{sourceAcquisition: true, sourceCache: true}
	for _, point := range points {
		if point.Value != 1 {
			t.Fatalf("resolution point value = %d, want 1", point.Value)
		}
		source := telemetrytest.Attribute(t, point.Attributes, attributeSource)
		if !wantSources[source] {
			t.Fatalf("resolution source = %q, want acquisition or cache", source)
		}
		delete(wantSources, source)
		requirePointSet(t, point.Attributes, map[string]string{
			attributeDependency: validTestConfig().DependencyName,
			attributeSource:     source,
			attributeResult:     resultSuccess,
		})
	}
	if len(wantSources) != 0 {
		t.Fatalf("missing resolution sources: %v", wantSources)
	}
	requireCounterPoint(t, metrics, providerAttemptsInstrument, "{attempt}", 1, map[string]string{
		attributeDependency: validTestConfig().DependencyName,
		attributeResult:     resultSuccess,
	})
	requireHistogramPoint(t, metrics, providerDurationInstrument, "s", map[string]string{
		attributeDependency: validTestConfig().DependencyName,
		attributeResult:     resultSuccess,
	})
}

func TestOutboundAuthTelemetryFailureDegradesToNoop(t *testing.T) {
	var output bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&output, nil))
	base := metricnoop.NewMeterProvider()
	meterProvider := failingMeterProvider{MeterProvider: base}
	provider := &scriptedAcquirer{steps: []acquisitionStep{{
		token: accessToken{value: "token", expiresAt: fixedProviderTime.Add(time.Minute)},
	}}}
	client := requireTestClient(t, validTestConfig(), testClientOptions{
		now:           func() time.Time { return fixedProviderTime },
		acquire:       provider.acquire,
		meterProvider: meterProvider,
		log:           log,
	})
	if value, err := requireOperationToken(t, client).authorization(); err != nil || value != "token" {
		t.Fatalf("authorization with failed telemetry = %q, %v", value, err)
	}
	if count := strings.Count(output.String(), eventMetricsDegraded); count != 1 {
		t.Fatalf("degraded warnings = %d, want 1; logs=%s", count, output.String())
	}
	if strings.Contains(output.String(), "poison metric construction") {
		t.Fatalf("metric error leaked: %s", output.String())
	}
}

type failingMeterProvider struct {
	metric.MeterProvider
}

//nolint:ireturn // The test double implements metric.MeterProvider.
func (p failingMeterProvider) Meter(name string, options ...metric.MeterOption) metric.Meter {
	return failingMeter{Meter: p.MeterProvider.Meter(name, options...)}
}

type failingMeter struct {
	metric.Meter
}

//nolint:ireturn // The test double implements metric.Meter.
func (m failingMeter) Int64Counter(
	name string,
	options ...metric.Int64CounterOption,
) (metric.Int64Counter, error) {
	instrument, err := m.Meter.Int64Counter(name, options...)
	if err != nil {
		return instrument, fmt.Errorf("create base test counter: %w", err)
	}
	return instrument, errors.New("poison metric construction")
}

//nolint:ireturn // The test double implements metric.Meter.
func (m failingMeter) Float64Histogram(
	name string,
	options ...metric.Float64HistogramOption,
) (metric.Float64Histogram, error) {
	instrument, err := m.Meter.Float64Histogram(name, options...)
	if err != nil {
		return instrument, fmt.Errorf("create base test histogram: %w", err)
	}
	return instrument, errors.New("poison metric construction")
}

func outboundAuthMetrics(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()
	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	if len(collected.ScopeMetrics) != 1 || collected.ScopeMetrics[0].Scope.Name != meterName {
		t.Fatalf("metric scopes = %#v, want only %q", collected.ScopeMetrics, meterName)
	}
	metrics := make(map[string]metricdata.Metrics)
	for _, measured := range collected.ScopeMetrics[0].Metrics {
		metrics[measured.Name] = measured
	}
	return metrics
}

func requireMetric(
	t *testing.T,
	metrics map[string]metricdata.Metrics,
	name string,
	unit string,
) metricdata.Metrics {
	t.Helper()
	measured, ok := metrics[name]
	if !ok {
		t.Fatalf("missing metric %q", name)
	}
	if measured.Unit != unit {
		t.Fatalf("%s unit = %q, want %q", name, measured.Unit, unit)
	}
	return measured
}

func requireCounterPoint(
	t *testing.T,
	metrics map[string]metricdata.Metrics,
	name string,
	unit string,
	want int64,
	attributes map[string]string,
) {
	t.Helper()
	points := telemetrytest.Int64Sum(t, requireMetric(t, metrics, name, unit)).DataPoints
	if len(points) != 1 || points[0].Value != want {
		t.Fatalf("%s points = %#v, want one point with value %d", name, points, want)
	}
	requirePointSet(t, points[0].Attributes, attributes)
}

func requireHistogramPoint(
	t *testing.T,
	metrics map[string]metricdata.Metrics,
	name string,
	unit string,
	attributes map[string]string,
) {
	t.Helper()
	points := telemetrytest.Float64Histogram(t, requireMetric(t, metrics, name, unit)).DataPoints
	if len(points) != 1 || points[0].Count != 1 {
		t.Fatalf("%s points = %#v, want one sample", name, points)
	}
	requirePointSet(t, points[0].Attributes, attributes)
}

func requirePointSet(t *testing.T, got attribute.Set, want map[string]string) {
	t.Helper()
	if got.Len() != len(want) {
		t.Fatalf("metric attributes = %v, want %v", got.ToSlice(), want)
	}
	for key, value := range want {
		if gotValue := telemetrytest.Attribute(t, got, key); gotValue != value {
			t.Fatalf("metric attribute %s = %q, want %q", key, gotValue, value)
		}
	}
}

func requireClosedResolutionAndProviderMatrix(t *testing.T, metrics map[string]metricdata.Metrics) {
	t.Helper()
	var successes int64
	// profile:outbound-auth-http:start
	successes += 2
	// profile:outbound-auth-http:end
	// profile:outbound-auth-grpc:start
	successes++
	// profile:outbound-auth-grpc:end
	wantResolutions := map[string]int64{
		"acquisition/success": successes, "cache/success": successes,
		"acquisition/" + string(FailureInvalidConfiguration): 1,
		"acquisition/" + string(FailureEndpointTrust):        1,
		"acquisition/" + string(FailureProviderTimeout):      1,
		"acquisition/" + string(FailureProviderUnavailable):  2,
		"acquisition/" + string(FailureClientRejected):       1,
		"acquisition/" + string(FailureGrantRejected):        1,
		"acquisition/" + string(FailureUnsupportedResponse):  1,
		"acquisition/" + string(FailureTokenUnusable):        1,
		"acquisition/" + string(FailureCallerCanceled):       1,
	}
	wantProvider := map[string]int64{
		"success":                           successes,
		string(FailureInvalidConfiguration): 1, string(FailureEndpointTrust): 1,
		string(FailureProviderTimeout): 1, string(FailureProviderUnavailable): 1,
		string(FailureClientRejected): 1, string(FailureGrantRejected): 1,
		string(FailureUnsupportedResponse): 1, string(FailureTokenUnusable): 1,
	}
	wantProviderAttempts := maps.Clone(wantProvider)
	wantProviderDurations := maps.Clone(wantProvider)
	for _, point := range telemetrytest.Int64Sum(t, requireMetric(t, metrics, tokenResolutionsInstrument, "{resolution}")).DataPoints {
		requireClosedOutcomeAttributes(t, point.Attributes, true)
		key := telemetrytest.Attribute(t, point.Attributes, attributeSource) + "/" + telemetrytest.Attribute(t, point.Attributes, attributeResult)
		if telemetrytest.Attribute(t, point.Attributes, attributeResult) == resultFailure {
			key = telemetrytest.Attribute(t, point.Attributes, attributeSource) + "/" + telemetrytest.Attribute(t, point.Attributes, attributeFailureClass)
		}
		if want, ok := wantResolutions[key]; !ok || point.Value != want {
			t.Fatalf("resolution %s = %d, want %d", key, point.Value, want)
		}
		delete(wantResolutions, key)
	}
	if len(wantResolutions) != 0 {
		t.Fatalf("missing resolution points: %v", wantResolutions)
	}
	for _, point := range telemetrytest.Int64Sum(t, requireMetric(t, metrics, providerAttemptsInstrument, "{attempt}")).DataPoints {
		requireClosedOutcomeAttributes(t, point.Attributes, false)
		key := telemetrytest.Attribute(t, point.Attributes, attributeResult)
		if key == resultFailure {
			key = telemetrytest.Attribute(t, point.Attributes, attributeFailureClass)
		}
		if want, ok := wantProviderAttempts[key]; !ok || point.Value != want {
			t.Fatalf("provider attempts %s = %d, want %d", key, point.Value, want)
		}
		delete(wantProviderAttempts, key)
	}
	if len(wantProviderAttempts) != 0 {
		t.Fatalf("missing provider attempt points: %v", wantProviderAttempts)
	}
	for _, point := range telemetrytest.Float64Histogram(t, requireMetric(t, metrics, providerDurationInstrument, "s")).DataPoints {
		requireClosedOutcomeAttributes(t, point.Attributes, false)
		key := telemetrytest.Attribute(t, point.Attributes, attributeResult)
		if key == resultFailure {
			key = telemetrytest.Attribute(t, point.Attributes, attributeFailureClass)
		}
		if want, ok := wantProviderDurations[key]; !ok || point.Count != uint64(want) {
			t.Fatalf("provider duration %s = %d, want %d", key, point.Count, want)
		}
		delete(wantProviderDurations, key)
	}
	if len(wantProviderDurations) != 0 {
		t.Fatalf("missing provider duration points: %v", wantProviderDurations)
	}
}

func requireClosedOutcomeAttributes(t *testing.T, attributes attribute.Set, resolution bool) {
	t.Helper()
	result := telemetrytest.Attribute(t, attributes, attributeResult)
	want := map[string]string{attributeDependency: validTestConfig().DependencyName, attributeResult: result}
	if resolution {
		source := telemetrytest.Attribute(t, attributes, attributeSource)
		if source != sourceCache && source != sourceAcquisition {
			t.Fatalf("resolution source = %q", source)
		}
		want[attributeSource] = source
	}
	if result == resultFailure {
		class := FailureClass(telemetrytest.Attribute(t, attributes, attributeFailureClass))
		if !validFailureClass(class) {
			t.Fatalf("failure class = %q", class)
		}
		want[attributeFailureClass] = string(class)
	} else if result != resultSuccess {
		t.Fatalf("outcome result = %q", result)
	}
	requirePointSet(t, attributes, want)
}

var (
	_ metric.MeterProvider = failingMeterProvider{}
	_ metric.Meter         = failingMeter{}
)
