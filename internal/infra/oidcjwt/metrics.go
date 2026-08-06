package oidcjwt

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

const meterName = "service.authn"

// Transport names the carrier a verification arrived on. It is a named type
// rather than a plain string so the accepted set is closed by the compiler, and
// each value is published verbatim as the authn.transport metric attribute,
// which is why it lives beside recordVerification rather than with the
// [Verifier] whose method carries it. Adding a third is an extension the package
// documentation owns.
type Transport string

const (
	TransportHTTP Transport = "http"
	TransportGRPC Transport = "grpc"
)

type authnMetrics struct {
	verifications metric.Int64Counter
	refreshes     metric.Int64Counter
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
	return authnMetrics{
		verifications: counterOrNoop(
			meter, fallback, reportDegraded,
			"authn.verifications", "{verification}", "Authentication verification outcomes.",
		),
		refreshes: counterOrNoop(
			meter, fallback, reportDegraded,
			"authn.jwks.refreshes", "{refresh}", "OIDC JWKS refresh outcomes.",
		),
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
	attributes := []attribute.KeyValue{
		attribute.String("authn.transport", string(transport)),
	}
	if err == nil {
		attributes = append(attributes, attribute.String("authn.result", "success"))
	} else {
		attributes = append(
			attributes,
			attribute.String("authn.result", "failure"),
			attribute.String("authn.reason", verificationReason(err)),
		)
	}
	m.verifications.Add(ctx, 1, metric.WithAttributes(attributes...))
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

// verificationReason names one failure category for the verification counter.
// The returned set is closed and one label per Kind, which keeps cardinality
// fixed while leaving each cause separately alertable — untrusted_transport in
// particular, because it means the deployment's trusted_proxy_cidrs no longer
// matches the real peer and every request is failing, not that one client sent
// a bad header.
//
// canceled is the one label with no Kind behind it. [Verifier.Verify] returns
// the caller's own cancellation or deadline unchanged so each transport can
// answer with the caller's status rather than an authentication outcome, and
// that error carries no Kind. Counting it as invalid would report a client that
// hung up mid-refresh as a client that presented a bad credential, in the one
// series operators are told to expect noise in. Anything else arriving without a
// Kind is unclassified, and unavailable is the answer that does not blame the
// caller for it.
func verificationReason(err error) string {
	kind, ok := KindOf(err)
	if !ok {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "canceled"
		}
		return "unavailable"
	}
	// In [Kind] declaration order, as errors.go is, so the label table and the
	// message table can be read against each other line by line.
	switch kind {
	case KindMissing:
		return "missing"
	case KindMalformed:
		return "malformed"
	case KindOversize:
		return "oversize"
	case KindInvalid:
		return "invalid"
	case KindUnavailable:
		return "unavailable"
	case KindUntrustedTransport:
		return "untrusted_transport"
	default:
		return "unavailable"
	}
}

func keyAge(now, fetchedAt time.Time) float64 {
	if fetchedAt.IsZero() || now.Before(fetchedAt) {
		return 0
	}
	return now.Sub(fetchedAt).Seconds()
}
