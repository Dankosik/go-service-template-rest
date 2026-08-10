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
	verifications   metric.Int64Counter
	refreshes       metric.Int64Counter
	refreshDuration metric.Float64Histogram

	// byTransport holds the prebuilt verification attribute sets, one group per
	// carrier. verificationSets owns why they are built once.
	byTransport map[transport]verificationSets
}

// verificationSets holds one prebuilt measurement option per series
// recordVerification can emit for a single carrier.
//
// This is the one hot path in the package: every authenticated request records
// exactly once. Building the attribute set there costs a slice, a sort, and the
// set itself on every call, and the space is closed and tiny — one success plus
// one per label verificationReason returns — so it is built at construction
// instead. TestPrebuiltAttributeSetsBeatPerCallConstruction holds that trade,
// and BenchmarkRecordVerificationFailure reports what the path costs; neither
// number belongs in this comment, where it would rot unread.
type verificationSets struct {
	transport transport
	success   metric.MeasurementOption
	failures  map[string]metric.MeasurementOption
}

func newVerificationSets(transport transport) verificationSets {
	failures := make(map[string]metric.MeasurementOption, len(kindDetails)+1)
	for _, detail := range kindDetails[1:] {
		failures[detail.reason] = failureOption(transport, detail.reason)
	}
	for _, unkinded := range []error{context.Canceled, context.DeadlineExceeded, errUnclassified} {
		reason := verificationReason(unkinded)
		failures[reason] = failureOption(transport, reason)
	}
	return verificationSets{
		transport: transport,
		success: metric.WithAttributeSet(attribute.NewSet(
			attribute.String("authn.transport", string(transport)),
			attribute.String("authn.result", "success"),
		)),
		failures: failures,
	}
}

//nolint:ireturn // metric.MeasurementOption is the option interface the OTel API takes.
func failureOption(transport transport, reason string) metric.MeasurementOption {
	return metric.WithAttributeSet(attribute.NewSet(
		attribute.String("authn.transport", string(transport)),
		attribute.String("authn.result", "failure"),
		attribute.String("authn.reason", reason),
	))
}

// option returns the prebuilt series err belongs to.
//
//nolint:ireturn // metric.MeasurementOption is the option interface the OTel API takes.
func (s verificationSets) option(err error) metric.MeasurementOption {
	if err == nil {
		return s.success
	}
	reason := verificationReason(err)
	if option, ok := s.failures[reason]; ok {
		return option
	}
	// Preserve the count if a future reason misses the prebuilt set.
	return failureOption(s.transport, reason)
}

// newDegradedWarning returns the report function newAuthnMetrics and
// registerKeyAgeGauge share. A meter that cannot serve one instrument usually
// cannot serve any of them, so the warning fires at most once for the whole
// Verifier instead of once per instrument.
func newDegradedWarning(log *slog.Logger) func() {
	if log == nil {
		log = slog.Default()
	}
	return sync.OnceFunc(func() {
		log.Warn(
			"authn_metrics_degraded",
			"component", "authn",
			"reason", "instrument_unavailable",
		)
	})
}

func newAuthnMetrics(provider metric.MeterProvider, reportDegraded func()) authnMetrics {
	if provider == nil {
		provider = metricnoop.NewMeterProvider()
	}
	meter := provider.Meter(meterName)
	fallback := metricnoop.NewMeterProvider().Meter(meterName)
	// Both declared carriers get their sets up front. A transport added in
	// verifier.go and not added here still records — recordVerification builds
	// its group per call instead — so the omission costs allocations rather than
	// a series.
	byTransport := make(map[transport]verificationSets, 2)
	for _, transport := range []transport{transportHTTP, transportGRPC} {
		byTransport[transport] = newVerificationSets(transport)
	}
	refreshDuration, err := meter.Float64Histogram(
		"authn.jwks.refresh.duration",
		metric.WithUnit("s"),
		metric.WithDescription("OIDC JWKS refresh duration."),
	)
	if err != nil || refreshDuration == nil {
		reportDegraded()
		refreshDuration, _ = fallback.Float64Histogram("authn.jwks.refresh.duration")
	}
	return authnMetrics{
		verifications: counterOrNoop(
			meter, fallback, reportDegraded,
			"authn.verifications", "{verification}", "Authentication verification outcomes.",
		),
		refreshes: counterOrNoop(
			meter, fallback, reportDegraded,
			"authn.jwks.refreshes", "{refresh}", "OIDC JWKS refresh outcomes.",
		),
		refreshDuration: refreshDuration,
		byTransport:     byTransport,
	}
}

// counterOrNoop builds one counter, or a no-op one from fallback when the meter
// cannot serve it. Falling back rather than leaving the counter nil is what keeps
// recording a plain call on every path, so a broken meter degrades telemetry
// instead of authentication.
//
// Every counter in this package is built here, so a third one cannot silently
// skip the fallback or the degraded warning.
//
//nolint:ireturn // metric.Int64Counter is the instrument interface the OTel API returns.
func counterOrNoop(
	meter, fallback metric.Meter,
	reportDegraded func(),
	name, unit, description string,
) metric.Int64Counter {
	counter, err := meter.Int64Counter(
		name,
		metric.WithUnit(unit),
		metric.WithDescription(description),
	)
	if err != nil || counter == nil {
		reportDegraded()
		counter, _ = fallback.Int64Counter(name)
	}
	return counter
}

func registerKeyAgeGauge(
	provider metric.MeterProvider,
	current func() *keySet,
	now func() time.Time,
	reportDegraded func(),
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
		reportDegraded()
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
		reportDegraded()
		return func() {}
	}
	return func() {
		_ = registration.Unregister()
	}
}

func (m authnMetrics) recordVerification(ctx context.Context, transport transport, err error) {
	sets, prebuilt := m.byTransport[transport]
	if !prebuilt {
		sets = newVerificationSets(transport)
	}
	m.verifications.Add(ctx, 1, sets.option(err))
}

func (m authnMetrics) recordRefresh(
	ctx context.Context,
	trigger refreshTrigger,
	err error,
	duration time.Duration,
) {
	result := "success"
	attributes := []attribute.KeyValue{
		attribute.String("authn.refresh.trigger", string(trigger)),
	}
	if err != nil {
		result = "failure"
		attributes = append(attributes, attribute.String("authn.reason", providerFailureReason(err)))
	}
	attributes = append(attributes, attribute.String("authn.result", result))
	measurement := metric.WithAttributes(attributes...)
	m.refreshes.Add(ctx, 1, measurement)
	m.refreshDuration.Record(ctx, max(0, duration.Seconds()), measurement)
}

func keyAge(now, fetchedAt time.Time) float64 {
	if fetchedAt.IsZero() || now.Before(fetchedAt) {
		return 0
	}
	return now.Sub(fetchedAt).Seconds()
}
