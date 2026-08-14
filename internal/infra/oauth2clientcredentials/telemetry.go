package oauth2clientcredentials

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

type telemetry struct {
	resolutions        metric.Int64Counter
	providerAttempts   metric.Int64Counter
	providerDuration   metric.Float64Histogram
	resourceRejections metric.Int64Counter
	cache              outcomeMeasurements
	acquisition        outcomeMeasurements
	provider           outcomeMeasurements
	rejections         map[resourceRejection]metric.MeasurementOption
}

type resourceRejection struct {
	transport string
	result    string
}

type outcomeMeasurements struct {
	success         metric.MeasurementOption
	failure         metric.MeasurementOption
	failuresByClass map[FailureClass]metric.MeasurementOption
}

func newTelemetry(provider metric.MeterProvider, dependency string, log *slog.Logger) telemetry {
	if provider == nil {
		provider = metricnoop.NewMeterProvider()
	}
	if log == nil {
		log = slog.Default()
	}
	reportDegraded := sync.OnceFunc(func() {
		log.Warn(
			eventMetricsDegraded,
			fieldComponent, componentOutboundAuth,
			fieldReason, reasonInstrumentFailed,
		)
	})
	meter := provider.Meter(meterName)
	fallback := metricnoop.NewMeterProvider().Meter(meterName)
	resolutions := counterOrNoop(
		meter,
		fallback,
		reportDegraded,
		tokenResolutionsInstrument,
		"{resolution}",
		"Outbound authentication token resolution outcomes.",
	)
	providerAttempts := counterOrNoop(
		meter,
		fallback,
		reportDegraded,
		providerAttemptsInstrument,
		"{attempt}",
		"Outbound authentication provider attempt outcomes.",
	)
	providerDuration, err := meter.Float64Histogram(
		providerDurationInstrument,
		metric.WithUnit("s"),
		metric.WithDescription("Outbound authentication provider attempt duration."),
	)
	if err != nil || providerDuration == nil {
		reportDegraded()
		providerDuration, _ = fallback.Float64Histogram(providerDurationInstrument)
	}
	resourceRejections := counterOrNoop(
		meter,
		fallback,
		reportDegraded,
		resourceRejectionsInstrument,
		"{rejection}",
		"Outbound authentication resource rejection outcomes.",
	)
	return telemetry{
		resolutions:        resolutions,
		providerAttempts:   providerAttempts,
		providerDuration:   providerDuration,
		resourceRejections: resourceRejections,
		cache:              newResolutionMeasurements(dependency, sourceCache),
		acquisition:        newResolutionMeasurements(dependency, sourceAcquisition),
		provider:           newProviderMeasurements(dependency),
		rejections:         newResourceRejectionMeasurements(dependency),
	}
}

func newResourceRejectionMeasurements(dependency string) map[resourceRejection]metric.MeasurementOption {
	measurements := make(map[resourceRejection]metric.MeasurementOption, 4)
	for _, transport := range []string{transportHTTP, transportGRPC} {
		for _, result := range []string{resultUnauthenticated, resultForbidden} {
			measurements[resourceRejection{transport: transport, result: result}] = metric.WithAttributeSet(attribute.NewSet(
				attribute.String(attributeDependency, dependency),
				attribute.String(attributeTransport, transport),
				attribute.String(attributeResult, result),
			))
		}
	}
	return measurements
}

func newResolutionMeasurements(dependency string, source string) outcomeMeasurements {
	base := []attribute.KeyValue{
		attribute.String(attributeDependency, dependency),
		attribute.String(attributeSource, source),
	}
	return newOutcomeMeasurements(base)
}

func newProviderMeasurements(dependency string) outcomeMeasurements {
	return newOutcomeMeasurements([]attribute.KeyValue{
		attribute.String(attributeDependency, dependency),
	})
}

func newOutcomeMeasurements(base []attribute.KeyValue) outcomeMeasurements {
	success := append(slices.Clone(base), attribute.String(attributeResult, resultSuccess))
	failureAttributes := append(slices.Clone(base), attribute.String(attributeResult, resultFailure))
	failuresByClass := make(map[FailureClass]metric.MeasurementOption, len(failureMessages))
	for class := range failureMessages {
		attributes := append(slices.Clone(failureAttributes), attribute.String(attributeFailureClass, string(class)))
		failuresByClass[class] = metric.WithAttributeSet(attribute.NewSet(attributes...))
	}
	return outcomeMeasurements{
		success:         metric.WithAttributeSet(attribute.NewSet(success...)),
		failure:         metric.WithAttributeSet(attribute.NewSet(failureAttributes...)),
		failuresByClass: failuresByClass,
	}
}

//nolint:ireturn // metric.Int64Counter is the instrument interface the OTel API returns.
func counterOrNoop(
	meter metric.Meter,
	fallback metric.Meter,
	reportDegraded func(),
	name string,
	unit string,
	description string,
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

func (t telemetry) recordResolution(ctx context.Context, source string, err error) {
	measurements := t.acquisition
	if source == sourceCache {
		measurements = t.cache
	}
	t.resolutions.Add(ctx, 1, measurements.option(err))
}

func (t telemetry) recordProviderAttempt(ctx context.Context, err error, duration time.Duration) {
	measurement := t.provider.option(err)
	t.providerAttempts.Add(ctx, 1, measurement)
	t.providerDuration.Record(ctx, max(0, duration.Seconds()), measurement)
}

func (t telemetry) recordResourceRejection(ctx context.Context, transport, result string) {
	if measurement, ok := t.rejections[resourceRejection{transport: transport, result: result}]; ok {
		t.resourceRejections.Add(ctx, 1, measurement)
	}
}

//nolint:ireturn // metric.MeasurementOption is the option interface the OTel API takes.
func (m outcomeMeasurements) option(err error) metric.MeasurementOption {
	if err == nil {
		return m.success
	}
	if class, ok := FailureClassOf(err); ok {
		if measurement, exists := m.failuresByClass[class]; exists {
			return measurement
		}
	}
	return m.failure
}
