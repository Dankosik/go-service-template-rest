package oidcjwt

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAuthnMetrics(t *testing.T) {
	t.Run("bounded signal contract", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		t.Cleanup(func() {
			if err := provider.Shutdown(t.Context()); err != nil {
				t.Errorf("shutdown metric provider: %v", err)
			}
		})

		now := time.Unix(1_900_000_000, 0)
		warning := newAuthnMetricWarning(slog.New(slog.DiscardHandler))
		metrics := newAuthnMetrics(provider, warning)
		unregister := registerKeyAgeGauge(
			provider,
			func() *keySet { return &keySet{fetchedAt: now.Add(-10 * time.Second)} },
			func() time.Time { return now },
			warning,
		)
		t.Cleanup(unregister)

		metrics.recordVerification(t.Context(), TransportHTTP, nil)
		metrics.recordVerification(t.Context(), TransportGRPC, failure(KindUntrustedTransport))
		metrics.recordRefresh(t.Context(), "startup", nil)
		metrics.recordRefresh(t.Context(), "scheduled", errors.New("poison provider detail"))

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
			{"authn.transport": "grpc", "authn.result": "failure", "authn.reason": "malformed"},
		})

		refreshes := requireMetric(t, got, "authn.jwks.refreshes", "{refresh}")
		refreshSum, ok := refreshes.Data.(metricdata.Sum[int64])
		if !ok || len(refreshSum.DataPoints) != 2 {
			t.Fatalf("refresh data = %#v, want two int64 sums", refreshes.Data)
		}
		requireMetricAttributes(t, refreshSum.DataPoints, []map[string]string{
			{"authn.refresh.trigger": "startup", "authn.result": "success"},
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
	})

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
			now := time.Unix(1_900_000_000, 0)
			key := loadTestRSAKey(t, "test-key-1.pem")
			client := &scriptedClient{responses: initialResponses(t, key)}
			verifier, err := newVerifier(
				t.Context(),
				testPolicy(t),
				func(string) (providerClient, error) {
					return providerClient{request: client, close: func() {}}, nil
				},
				func() time.Time { return now },
				provider,
				log,
			)
			if err != nil {
				t.Fatalf("newVerifier() with failing metrics error = %v", err)
			}
			t.Cleanup(verifier.Close)

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
	remaining := make([]map[string]string, len(want))
	copy(remaining, want)
	for _, point := range points {
		got := make(map[string]string)
		for _, value := range point.Attributes.ToSlice() {
			got[string(value.Key)] = value.Value.AsString()
		}
		for index, candidate := range remaining {
			if mapsEqual(got, candidate) {
				remaining = append(remaining[:index], remaining[index+1:]...)
				break
			}
		}
	}
	if len(remaining) != 0 {
		t.Fatalf("metric attributes did not contain %#v", remaining)
	}
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

var _ metric.MeterProvider = failingMetricProvider{}
