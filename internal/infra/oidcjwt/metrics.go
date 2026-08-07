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

	// byTransport holds the prebuilt verification attribute sets, one group per
	// carrier. verificationSets owns why they are built once.
	byTransport map[Transport]verificationSets
}

// verificationSets holds one prebuilt measurement option per series
// recordVerification can emit for a single carrier.
//
// This is the one hot path in the package: every authenticated request records
// exactly once. Building the attribute set there costs a slice, a sort, and the
// set itself on every call, and the space is closed and tiny — one success plus
// one per label verificationReason returns — so it is built at construction
// instead. Measured on the failure path, that is 6 allocations and 624 B per
// verification against 2 and 24 B.
//
// The labels come from verificationReason rather than from a list beside it,
// which is what keeps that function the only table over [Kind]: the walk below
// covers every declared category, and the labels no Kind produces are asked for
// by the errors that produce them.
type verificationSets struct {
	transport Transport
	success   metric.MeasurementOption
	failures  map[string]metric.MeasurementOption
}

func newVerificationSets(transport Transport) verificationSets {
	failures := make(map[string]metric.MeasurementOption, int(lastKind)+1)
	for kind := Kind(1); kind <= lastKind; kind++ {
		reason := verificationReason(failure(kind))
		failures[reason] = failureOption(transport, reason)
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
func failureOption(transport Transport, reason string) metric.MeasurementOption {
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
	// Unreachable while newVerificationSets covers every label
	// verificationReason returns, which is what lastKind keeps true. Building one
	// here makes a label it somehow missed cost allocations rather than a lost
	// count.
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
	// Both declared carriers get their sets up front. A [Transport] added in
	// verifier.go and not added here still records — recordVerification builds
	// its group per call instead — so the omission costs allocations rather than
	// a series.
	byTransport := make(map[Transport]verificationSets, 2)
	for _, transport := range []Transport{TransportHTTP, TransportGRPC} {
		byTransport[transport] = newVerificationSets(transport)
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
		byTransport: byTransport,
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

func (m authnMetrics) recordVerification(ctx context.Context, transport Transport, err error) {
	sets, prebuilt := m.byTransport[transport]
	if !prebuilt {
		sets = newVerificationSets(transport)
	}
	m.verifications.Add(ctx, 1, sets.option(err))
}

func (m authnMetrics) recordRefresh(ctx context.Context, trigger refreshTrigger, err error) {
	result := "success"
	if err != nil {
		result = "failure"
	}
	m.refreshes.Add(ctx, 1, metric.WithAttributes(
		attribute.String("authn.refresh.trigger", string(trigger)),
		attribute.String("authn.result", result),
	))
}

func keyAge(now, fetchedAt time.Time) float64 {
	if fetchedAt.IsZero() || now.Before(fetchedAt) {
		return 0
	}
	return now.Sub(fetchedAt).Seconds()
}
