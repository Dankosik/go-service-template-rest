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

	"github.com/getkin/kin-openapi/openapi3filter"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAuthnMetricAttributesAreBounded(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(t.Context()); err != nil {
			t.Errorf("shutdown metric provider: %v", err)
		}
	})

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
	metrics.recordVerification(t.Context(), TransportHTTP, nil)
	metrics.recordVerification(t.Context(), TransportGRPC, failure(KindUntrustedTransport))
	metrics.recordRefresh(t.Context(), triggerStartup, nil)
	metrics.recordRefresh(t.Context(), triggerKeyMiss, nil)
	metrics.recordRefresh(t.Context(), triggerScheduled, errors.New("poison provider detail"))

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("collect authn metrics: %v", err)
	}
	got := authnMetricSet(t, collected)
	if len(got) != 3 {
		t.Fatalf("authn metric count = %d, want 3: %#v", len(got), got)
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
		{"authn.refresh.trigger": "scheduled", "authn.result": "failure"},
	})

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

// Both cases here reach the counter without ever running a signature check,
// which is where the count is easiest to lose: the HTTP boundary turns a
// credential away before Verify, and a caller hangs up while a key-miss
// refresh is still in flight. The first would go uncounted if an adapter
// stopped routing through recordRejection; the second was counted as "invalid"
// until verificationReason stopped reading a missing Kind as a bad credential.
func TestAuthnMetricsCountRejectionsVerifyNeverSaw(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(t.Context()); err != nil {
			t.Errorf("shutdown metric provider: %v", err)
		}
	})

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

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://service.example/protected",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	_, err = verifier.ResolveHTTP(t.Context(), &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request},
	})
	requireKind(t, err, KindMissing)

	// An unknown key id starts a refresh the scripted provider holds open, so
	// the cancellation lands while Verify is waiting on it.
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, verifyErr := verifier.Verify(
			ctx,
			signToken(t, second, "key-2", "at+jwt", validClaims(now)),
			TransportGRPC,
		)
		result <- verifyErr
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify() error = %v, want cancellation", err)
	}
	close(release)

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("collect authn metrics: %v", err)
	}
	verifications := requireMetric(t, authnMetricSet(t, collected), "authn.verifications", "{verification}")
	verificationSum, ok := verifications.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("verification data = %#v, want int64 sums", verifications.Data)
	}
	requireMetricAttributes(t, verificationSum.DataPoints, []map[string]string{
		{"authn.transport": "http", "authn.result": "failure", "authn.reason": "missing"},
		{"authn.transport": "grpc", "authn.result": "failure", "authn.reason": "canceled"},
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
			principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
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

// TestVerificationSetsCoverEveryReason holds the prebuilt attribute sets to the
// full label set verificationReason can produce.
//
// [verificationSets] prebuilds one per category by walking to lastKind, and
// lastKind is the one constant a new category does not update on its own. Left
// behind, it would leave the new category's series unbuilt, and option's
// on-the-fly fallback would turn that into a silently slower hot path rather
// than a visibly wrong one. The walk below finds the end of the declared run
// independently, the way [TestDocumentedMetricReasonsMatchTheGuide] does.
func TestVerificationSetsCoverEveryReason(t *testing.T) {
	unnamed := NewError(0).Error()
	reasons := []string{
		verificationReason(context.Canceled),
		verificationReason(context.DeadlineExceeded),
		verificationReason(errUnclassified),
	}
	declared := Kind(0)
	for kind := Kind(1); NewError(kind).Error() != unnamed; kind++ {
		declared = kind
		reasons = append(reasons, verificationReason(NewError(kind)))
	}
	if declared != lastKind {
		t.Fatalf(
			"the declared Kind run ends at %d but lastKind is %d; metrics.go prebuilds "+
				"one attribute set per category by walking to lastKind, so the two must agree",
			declared, lastKind,
		)
	}

	sets := newVerificationSets(TransportHTTP)
	// Several errors share one label on purpose — a deadline and a cancellation
	// both record as canceled, and an unclassified error records as unavailable
	// like the Kind of that name — so the series are the distinct labels rather
	// than the errors that reach them. verificationReason owns each of those
	// collapses.
	distinct := make(map[string]struct{}, len(reasons))
	for _, reason := range reasons {
		distinct[reason] = struct{}{}
		if _, prebuilt := sets.failures[reason]; !prebuilt {
			t.Errorf(
				"the %q series is not prebuilt, so every verification recording it "+
					"builds its attribute set on the request path",
				reason,
			)
		}
	}
	if got := len(sets.failures); got != len(distinct) {
		t.Errorf("prebuilt %d failure series for %d distinct reachable labels", got, len(distinct))
	}
}

var _ metric.MeterProvider = failingMetricProvider{}
