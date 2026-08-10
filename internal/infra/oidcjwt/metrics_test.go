package oidcjwt

// Proof for metrics.go: the bounded attribute sets authn.verifications and
// authn.jwks.refreshes carry, the reason label errors.go maps every failure
// category to, and that a meter which cannot serve an instrument degrades
// telemetry rather than authentication. That those labels also reach operators
// is proven in docs_test.go.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
)

func TestAuthnMetricAttributesAreBounded(t *testing.T) {
	reader, provider := telemetrytest.NewManualMeterProvider(t)

	now := testNow
	reportDegraded := newDegradedWarning(slog.New(slog.DiscardHandler))
	metrics := newAuthnMetrics(provider, reportDegraded)
	unregister := registerKeyAgeGauge(
		provider,
		func() *keySet { return &keySet{fetchedAt: now.Add(-10 * time.Second)} },
		func() time.Time { return now },
		reportDegraded,
	)
	t.Cleanup(unregister)

	// The constants are the inputs and the literals are the assertions, both
	// here and below: that is what ties a rename of one of these identifiers to
	// the label docs/authentication.md publishes, instead of letting the two
	// drift with every test still green.
	metrics.recordVerification(t.Context(), transportHTTP, nil)
	metrics.recordVerification(t.Context(), transportGRPC, failure(KindUntrustedTransport))
	metrics.recordRefresh(t.Context(), triggerStartup, nil, time.Second)
	metrics.recordRefresh(t.Context(), triggerKeyMiss, nil, 2*time.Second)
	metrics.recordRefresh(t.Context(), triggerScheduled, errProviderTransport, 3*time.Second)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("collect authn metrics: %v", err)
	}
	got := authnMetricSet(t, collected)
	if len(got) != 4 {
		t.Fatalf("authn metric count = %d, want 4: %#v", len(got), got)
	}

	verifications := requireMetric(t, got, "authn.verifications", "{verification}")
	verificationSum, ok := verifications.Data.(metricdata.Sum[int64])
	if !ok || len(verificationSum.DataPoints) != 2 {
		t.Fatalf("verification data = %#v, want two int64 sums", verifications.Data)
	}
	requireMetricAttributes(t, verificationSum.DataPoints, []map[string]string{
		{"authn.transport": "http", "authn.result": "success"},
		{"authn.transport": "grpc", "authn.result": "failure", "authn.reason": "untrusted_transport"},
	})

	refreshes := requireMetric(t, got, "authn.jwks.refreshes", "{refresh}")
	refreshSum, ok := refreshes.Data.(metricdata.Sum[int64])
	if !ok || len(refreshSum.DataPoints) != 3 {
		t.Fatalf("refresh data = %#v, want three int64 sums", refreshes.Data)
	}
	requireMetricAttributes(t, refreshSum.DataPoints, []map[string]string{
		{"authn.refresh.trigger": "startup", "authn.result": "success"},
		{"authn.refresh.trigger": "key_miss", "authn.result": "success"},
		{"authn.refresh.trigger": "scheduled", "authn.result": "failure", "authn.reason": "transport"},
	})

	refreshDuration := requireMetric(t, got, "authn.jwks.refresh.duration", "s")
	durationHistogram, ok := refreshDuration.Data.(metricdata.Histogram[float64])
	if !ok || len(durationHistogram.DataPoints) != 3 {
		t.Fatalf("refresh duration data = %#v, want three float64 histograms", refreshDuration.Data)
	}
	requireHistogramMetricAttributes(t, durationHistogram.DataPoints, []map[string]string{
		{"authn.refresh.trigger": "startup", "authn.result": "success"},
		{"authn.refresh.trigger": "key_miss", "authn.result": "success"},
		{"authn.refresh.trigger": "scheduled", "authn.result": "failure", "authn.reason": "transport"},
	})
	var durationCount uint64
	var durationSum float64
	for _, point := range durationHistogram.DataPoints {
		durationCount += point.Count
		durationSum += point.Sum
	}
	if durationCount != 3 || durationSum != 6 {
		t.Fatalf("refresh duration = (count %d, sum %v), want (3, 6)", durationCount, durationSum)
	}

	age := requireMetric(t, got, "authn.jwks.age", "s")
	ageGauge, ok := age.Data.(metricdata.Gauge[float64])
	if !ok || len(ageGauge.DataPoints) != 1 {
		t.Fatalf("key age data = %#v, want one float64 gauge", age.Data)
	}
	if gotAge := ageGauge.DataPoints[0].Value; gotAge != 10 {
		t.Fatalf("key age = %v, want 10", gotAge)
	}
	if ageGauge.DataPoints[0].Attributes.Len() != 0 {
		t.Fatalf("key age attributes = %v, want none", ageGauge.DataPoints[0].Attributes)
	}
}

// Both cases here reach production metric wiring where it is easiest to lose:
// the HTTP boundary turns a credential away before Verify, while a caller hangs
// up during a key-miss refresh that still has to record its own outcome and
// duration. The first would go uncounted if an adapter stopped routing through
// recordRejection; the second was counted as "invalid" until verificationReason
// stopped reading a missing Kind as a bad credential.
func TestAuthnMetricsCountRejectionsVerifyNeverSaw(t *testing.T) {
	reader, provider := telemetrytest.NewManualMeterProvider(t)

	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	release := make(chan struct{})
	started := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status:  http.StatusOK,
		body:    jwksDocument(t, second, "key-2"),
		wait:    release,
		started: started,
	})}
	verifier := requireTestVerifier(t, testVerifierOptions{
		now:      newTestClock(now).now,
		client:   client,
		provider: provider,
	})

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://service.example/protected", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	_, err = verifier.ResolveHTTP(t.Context(), bearerAuthInput(request))
	requireKind(t, err, KindMissing)

	// A requirement this boundary declines is not a verification outcome, so it
	// must not land beside the two that are. The series count below is what
	// proves that, because requireMetricAttributes only proves the wanted series
	// are present.
	if _, err := verifier.ResolveHTTP(t.Context(), otherSchemeAuthInput(request)); !errors.Is(
		err, errUnsupportedSecurityScheme,
	) {
		t.Fatalf("ResolveHTTP() error = %v, want an unsupported security scheme", err)
	}

	// An unknown key id starts a refresh the scripted provider holds open, so
	// the cancellation lands while Verify is waiting on it.
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, verifyErr := verifier.verify(
			ctx,
			signToken(t, second, "key-2", "at+jwt", validClaims(now)),
			transportGRPC,
		)
		result <- verifyErr
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify() error = %v, want cancellation", err)
	}
	close(release)
	verifier.admission.join()

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("collect authn metrics: %v", err)
	}
	got := authnMetricSet(t, collected)
	verifications := requireMetric(t, got, "authn.verifications", "{verification}")
	verificationSum, ok := verifications.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("verification data = %#v, want int64 sums", verifications.Data)
	}
	requireMetricAttributes(t, verificationSum.DataPoints, []map[string]string{
		{"authn.transport": "http", "authn.result": "failure", "authn.reason": "missing"},
		{"authn.transport": "grpc", "authn.result": "failure", "authn.reason": "canceled"},
	})
	if len(verificationSum.DataPoints) != 2 {
		t.Fatalf(
			"verification series = %d, want exactly the two rejections this boundary answered",
			len(verificationSum.DataPoints),
		)
	}

	refreshDuration := requireMetric(t, got, "authn.jwks.refresh.duration", "s")
	durationHistogram, ok := refreshDuration.Data.(metricdata.Histogram[float64])
	if !ok || len(durationHistogram.DataPoints) != 2 {
		t.Fatalf("refresh duration data = %#v, want startup and key-miss histograms", refreshDuration.Data)
	}
	requireHistogramMetricAttributes(t, durationHistogram.DataPoints, []map[string]string{
		{"authn.refresh.trigger": "startup", "authn.result": "success"},
		{"authn.refresh.trigger": "key_miss", "authn.result": "success"},
	})
}

func TestFailingMeterDegradesTelemetryNotAuthentication(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		failInstruments bool
		failCallback    bool
	}{
		{name: "instrument construction failure", failInstruments: true},
		{name: "callback registration failure", failCallback: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var logs strings.Builder
			log := slog.New(slog.NewJSONHandler(&logs, nil))
			base := metricnoop.NewMeterProvider()
			provider := failingMetricProvider{
				MeterProvider:   base,
				failInstruments: testCase.failInstruments,
				failCallback:    testCase.failCallback,
			}
			now := testNow
			key := loadTestRSAKey(t, testSigningKey)
			verifier := requireTestVerifier(t, testVerifierOptions{
				client:   &scriptedClient{responses: initialResponses(t, key)},
				provider: provider,
				log:      log,
			})

			token := signToken(t, key, "key-1", "at+jwt", validClaims(now))
			principal, err := verifier.verify(t.Context(), token, transportHTTP)
			if err != nil || principal.Subject != "opaque-subject" {
				t.Fatalf("authentication with failing metrics = (%+v, %v), want success", principal, err)
			}
			if count := strings.Count(logs.String(), "authn_metrics_degraded"); count != 1 {
				t.Fatalf("degraded metric warnings = %d, want exactly one; logs=%s", count, logs.String())
			}
			if strings.Contains(logs.String(), "poison metric construction") {
				t.Fatalf("metric construction detail leaked: %s", logs.String())
			}
		})
	}
}

type failingMetricProvider struct {
	metric.MeterProvider

	failInstruments bool
	failCallback    bool
}

//nolint:ireturn // The test double implements metric.MeterProvider's interface-returning method.
func (p failingMetricProvider) Meter(
	name string,
	options ...metric.MeterOption,
) metric.Meter {
	return failingMetricMeter{
		Meter:           p.MeterProvider.Meter(name, options...),
		failInstruments: p.failInstruments,
		failCallback:    p.failCallback,
	}
}

type failingMetricMeter struct {
	metric.Meter

	failInstruments bool
	failCallback    bool
}

//nolint:ireturn // The test double implements metric.Meter's interface-returning method.
func (m failingMetricMeter) Int64Counter(
	name string,
	options ...metric.Int64CounterOption,
) (metric.Int64Counter, error) {
	instrument, err := m.Meter.Int64Counter(name, options...)
	if m.failInstruments {
		return instrument, errors.New("poison metric construction")
	}
	if err != nil {
		return instrument, fmt.Errorf("create base test counter: %w", err)
	}
	return instrument, nil
}

//nolint:ireturn // The test double implements metric.Meter's interface-returning method.
func (m failingMetricMeter) Float64Histogram(
	name string,
	options ...metric.Float64HistogramOption,
) (metric.Float64Histogram, error) {
	instrument, err := m.Meter.Float64Histogram(name, options...)
	if m.failInstruments {
		return instrument, errors.New("poison metric construction")
	}
	if err != nil {
		return instrument, fmt.Errorf("create base test histogram: %w", err)
	}
	return instrument, nil
}

//nolint:ireturn // The test double implements metric.Meter's interface-returning method.
func (m failingMetricMeter) Float64ObservableGauge(
	name string,
	options ...metric.Float64ObservableGaugeOption,
) (metric.Float64ObservableGauge, error) {
	instrument, err := m.Meter.Float64ObservableGauge(name, options...)
	if m.failInstruments {
		return instrument, errors.New("poison metric construction")
	}
	if err != nil {
		return instrument, fmt.Errorf("create base test gauge: %w", err)
	}
	return instrument, nil
}

//nolint:ireturn // The test double implements metric.Meter's interface-returning method.
func (m failingMetricMeter) RegisterCallback(
	callback metric.Callback,
	instruments ...metric.Observable,
) (metric.Registration, error) {
	registration, err := m.Meter.RegisterCallback(callback, instruments...)
	if m.failCallback {
		if registration != nil {
			_ = registration.Unregister()
		}
		return nil, errors.New("poison metric construction")
	}
	if err != nil {
		return registration, fmt.Errorf("register base test callback: %w", err)
	}
	return registration, nil
}

func authnMetricSet(t *testing.T, collected metricdata.ResourceMetrics) map[string]metricdata.Metrics {
	t.Helper()
	result := make(map[string]metricdata.Metrics)
	for _, scope := range collected.ScopeMetrics {
		if scope.Scope.Name != meterName {
			continue
		}
		for _, measured := range scope.Metrics {
			result[measured.Name] = measured
		}
	}
	return result
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
		t.Fatalf("missing authn metric %q: %#v", name, metrics)
	}
	if measured.Unit != unit {
		t.Fatalf("%s unit = %q, want %q", name, measured.Unit, unit)
	}
	return measured
}

func requireMetricAttributes(
	t *testing.T,
	points []metricdata.DataPoint[int64],
	want []map[string]string,
) {
	t.Helper()
	remaining := slices.Clone(want)
	for _, point := range points {
		got := make(map[string]string)
		for _, value := range point.Attributes.ToSlice() {
			got[string(value.Key)] = value.Value.AsString()
		}
		for index, candidate := range remaining {
			if maps.Equal(got, candidate) {
				remaining = slices.Delete(remaining, index, index+1)
				break
			}
		}
	}
	if len(remaining) != 0 {
		t.Fatalf("metric attributes did not contain %#v", remaining)
	}
}

func requireHistogramMetricAttributes(
	t *testing.T,
	points []metricdata.HistogramDataPoint[float64],
	want []map[string]string,
) {
	t.Helper()
	remaining := slices.Clone(want)
	for _, point := range points {
		got := make(map[string]string)
		for _, value := range point.Attributes.ToSlice() {
			got[string(value.Key)] = value.Value.AsString()
		}
		for index, candidate := range remaining {
			if maps.Equal(got, candidate) {
				remaining = slices.Delete(remaining, index, index+1)
				break
			}
		}
	}
	if len(remaining) != 0 {
		t.Fatalf("metric attributes did not contain %#v", remaining)
	}
}

func TestKindDetailsHaveNoGaps(t *testing.T) {
	for index := range kindDetails[1:] {
		kind := Kind(index + 1)
		if _, ok := detailFor(kind); !ok {
			t.Errorf("Kind %d has no message or metric reason", kind)
		}
	}
	for _, kind := range []Kind{0, Kind(len(kindDetails))} {
		err := NewError(kind)
		message := err.Error()
		reason := verificationReason(err)
		if message != "authentication failed" || reason != "unavailable" {
			t.Errorf("unknown Kind %d = (%q, %q), want fail-closed defaults", kind, message, reason)
		}
	}
}

var _ metric.MeterProvider = failingMetricProvider{}
