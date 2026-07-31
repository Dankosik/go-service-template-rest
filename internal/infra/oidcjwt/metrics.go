package oidcjwt

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

const meterName = "service.authn"

type authnMetrics struct {
	verifications metric.Int64Counter
	refreshes     metric.Int64Counter
}

type authnMetricWarning struct {
	once sync.Once
	log  *slog.Logger
}

func newAuthnMetricWarning(log *slog.Logger) *authnMetricWarning {
	if log == nil {
		log = slog.Default()
	}
	return &authnMetricWarning{log: log}
}

func (w *authnMetricWarning) report() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		w.log.Warn(
			"authn_metrics_degraded",
			"component", "authn",
			"reason", "instrument_unavailable",
		)
	})
}

func newAuthnMetrics(provider metric.MeterProvider, warning *authnMetricWarning) authnMetrics {
	if provider == nil {
		provider = metricnoop.NewMeterProvider()
	}
	meter := provider.Meter(meterName)
	verifications, err := meter.Int64Counter(
		"authn.verifications",
		metric.WithUnit("{verification}"),
		metric.WithDescription("Authentication verification outcomes."),
	)
	if err != nil || verifications == nil {
		warning.report()
		verifications, _ = metricnoop.NewMeterProvider().Meter(meterName).Int64Counter("authn.verifications")
	}
	refreshes, err := meter.Int64Counter(
		"authn.jwks.refreshes",
		metric.WithUnit("{refresh}"),
		metric.WithDescription("OIDC JWKS refresh outcomes."),
	)
	if err != nil || refreshes == nil {
		warning.report()
		refreshes, _ = metricnoop.NewMeterProvider().Meter(meterName).Int64Counter("authn.jwks.refreshes")
	}
	return authnMetrics{verifications: verifications, refreshes: refreshes}
}

func registerKeyAgeGauge(
	provider metric.MeterProvider,
	current func() *keySet,
	now func() time.Time,
	warning *authnMetricWarning,
) func() {
	if provider == nil {
		provider = metricnoop.NewMeterProvider()
	}
	meter := provider.Meter(meterName)
	gauge, err := meter.Float64ObservableGauge(
		"authn.jwks.age",
		metric.WithUnit("s"),
		metric.WithDescription("Age of the current completely validated JWKS."),
	)
	if err != nil || gauge == nil {
		warning.report()
		return func() {}
	}
	registration, err := meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		keys := current()
		if keys != nil {
			observer.ObserveFloat64(gauge, keyAge(now(), keys.fetchedAt))
		}
		return nil
	}, gauge)
	if err != nil || registration == nil {
		warning.report()
		return func() {}
	}
	return func() {
		_ = registration.Unregister()
	}
}

func (m authnMetrics) recordVerification(ctx context.Context, transport string, err error) {
	attributes := []attribute.KeyValue{
		attribute.String("authn.transport", transport),
	}
	if err == nil {
		attributes = append(attributes, attribute.String("authn.result", "success"))
	} else {
		kind, _ := KindOf(err)
		attributes = append(
			attributes,
			attribute.String("authn.result", "failure"),
			attribute.String("authn.reason", metricReason(kind)),
		)
	}
	m.verifications.Add(ctx, 1, metric.WithAttributes(attributes...))
}

func (m authnMetrics) recordRefresh(ctx context.Context, trigger string, err error) {
	result := "success"
	if err != nil {
		result = "failure"
	}
	m.refreshes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("authn.refresh.trigger", trigger),
		attribute.String("authn.result", result),
	))
}

func metricReason(kind Kind) string {
	switch kind {
	case KindMissing:
		return "missing"
	case KindMalformed, KindUntrustedTransport:
		return "malformed"
	case KindOversize:
		return "oversize"
	case KindUnavailable:
		return "unavailable"
	case KindInvalid:
		return "invalid"
	default:
		return "invalid"
	}
}

func keyAge(now, fetchedAt time.Time) float64 {
	if fetchedAt.IsZero() || now.Before(fetchedAt) {
		return 0
	}
	return now.Sub(fetchedAt).Seconds()
}
