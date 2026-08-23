package telemetrytest

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// The helpers here read what a scrape would see from a manual reader. They are
// shared because the alternative is one metricdata walker per instrumented
// package, and those copies drift on the thing that matters least visibly:
// whether an absent metric fails the test or silently reads as its zero value.
// Everything below fails loudly, so "the gauge reported 0" and "the gauge was
// never published" can never be the same assertion.
//
// Unlike InstallManualReader these touch no process-wide state, so a test using
// them may run in parallel.

// NewManualMeterProvider returns a meter provider whose series a test collects
// on demand through the returned reader.
//
// It installs nothing process-wide, so a test that hands the provider to the
// code under test may run in parallel. Use [InstallManualReader] instead when
// that code reads the global provider, and [NewManualMeter] when it takes a
// meter rather than a provider.
func NewManualMeterProvider(tb testing.TB) (*sdkmetric.ManualReader, *sdkmetric.MeterProvider) {
	tb.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	shutdownAfterTest(tb, "meter", provider.Shutdown)
	return reader, provider
}

// NewManualMeter returns a manual reader and a meter recording into it, scoped
// to the instrumentation scope the package under test uses in production.
// Passing the production scope keeps a collected metric identified the way an
// operator would see it.
//
//nolint:ireturn // metric.Meter is OTel's own interface; a provider returns nothing else.
func NewManualMeter(tb testing.TB, scope string) (*sdkmetric.ManualReader, metric.Meter) {
	tb.Helper()

	reader, provider := NewManualMeterProvider(tb)
	return reader, provider.Meter(scope)
}

// ForEachMetric visits every collected metric across every scope, so a caller
// states what it wants from a metric instead of repeating the collect-and-walk.
// It is also the one place a collection failure is reported.
func ForEachMetric(tb testing.TB, reader *sdkmetric.ManualReader, visit func(metricdata.Metrics)) {
	tb.Helper()

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		tb.Fatalf("collect metrics: %v", err)
	}
	for _, scope := range collected.ScopeMetrics {
		for _, measured := range scope.Metrics {
			visit(measured)
		}
	}
}

// The aggregation accessors assert rather than assume. An instrument that
// changes aggregation is a readable failure naming the metric, not a panic
// inside a test that was reading something else.

func Int64Gauge(tb testing.TB, measured metricdata.Metrics) metricdata.Gauge[int64] {
	tb.Helper()
	gauge, ok := measured.Data.(metricdata.Gauge[int64])
	if !ok {
		tb.Fatalf("%s aggregation = %T, want an int64 gauge", measured.Name, measured.Data)
	}
	return gauge
}

func Float64Gauge(tb testing.TB, measured metricdata.Metrics) metricdata.Gauge[float64] {
	tb.Helper()
	gauge, ok := measured.Data.(metricdata.Gauge[float64])
	if !ok {
		tb.Fatalf("%s aggregation = %T, want a float64 gauge", measured.Name, measured.Data)
	}
	return gauge
}

func Int64Sum(tb testing.TB, measured metricdata.Metrics) metricdata.Sum[int64] {
	tb.Helper()
	sum, ok := measured.Data.(metricdata.Sum[int64])
	if !ok {
		tb.Fatalf("%s aggregation = %T, want an int64 sum", measured.Name, measured.Data)
	}
	return sum
}

func Float64Histogram(tb testing.TB, measured metricdata.Metrics) metricdata.Histogram[float64] {
	tb.Helper()
	histogram, ok := measured.Data.(metricdata.Histogram[float64])
	if !ok {
		tb.Fatalf("%s aggregation = %T, want a float64 histogram", measured.Name, measured.Data)
	}
	return histogram
}

// SinglePoint is the one point an instrument with no attributes carries.
func SinglePoint[N int64 | float64](tb testing.TB, name string, points []metricdata.DataPoint[N]) N {
	tb.Helper()
	if len(points) != 1 {
		tb.Fatalf("%s has %d points, want the single unattributed one", name, len(points))
	}
	return points[0].Value
}

// Int64GaugeValue is the value of a named unattributed int64 gauge. A metric
// that was never published fails here rather than reading back as zero, which
// for a readiness or depth gauge is also a meaningful value.
func Int64GaugeValue(tb testing.TB, reader *sdkmetric.ManualReader, name string) int64 {
	tb.Helper()

	var value int64
	found := false
	ForEachMetric(tb, reader, func(measured metricdata.Metrics) {
		if measured.Name != name {
			return
		}
		value = SinglePoint(tb, name, Int64Gauge(tb, measured).DataPoints)
		found = true
	})
	if !found {
		tb.Fatalf("metric %s was not collected", name)
	}
	return value
}

// Int64SumValue is the value of a named unattributed int64 sum.
func Int64SumValue(tb testing.TB, reader *sdkmetric.ManualReader, name string) int64 {
	tb.Helper()

	var value int64
	found := false
	ForEachMetric(tb, reader, func(measured metricdata.Metrics) {
		if measured.Name != name {
			return
		}
		value = SinglePoint(tb, name, Int64Sum(tb, measured).DataPoints)
		found = true
	})
	if !found {
		tb.Fatalf("metric %s was not collected", name)
	}
	return value
}

// AttributeSets is every point's attribute set for one instrument, whatever its
// aggregation. Use it to assert on an instrument's label cardinality and
// vocabulary without knowing which aggregation it carries.
func AttributeSets(tb testing.TB, aggregation metricdata.Aggregation) []attribute.Set {
	tb.Helper()

	sets := make([]attribute.Set, 0)
	switch data := aggregation.(type) {
	case metricdata.Gauge[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Gauge[float64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Sum[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Sum[float64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Histogram[int64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, point := range data.DataPoints {
			sets = append(sets, point.Attributes)
		}
	default:
		tb.Fatalf("unexpected metric aggregation %T", aggregation)
	}
	return sets
}

// AssertNoAttributeContains fails when any collected metric attribute — key or
// value — contains one of the forbidden substrings. An empty candidate matches
// everything and is skipped, so a caller may pass a correlation value that the
// case under test left absent.
//
// This is a trust-boundary assertion rather than a metrics one. The client
// adapters may label a series with the target and the outcome; a request ID or
// a trace ID reaching a label is both an unbounded-cardinality series and the
// correlation value the propagation policy decided not to emit.
//
// A collection that produced no points fails here. The walk would otherwise pass
// every check below while covering nothing, which is exactly how this assertion
// would keep reporting success after the instruments it guards stopped
// recording. Fail-closed on the aggregation too: AttributeSets refuses one it
// does not know rather than skipping the points it cannot read, because the two
// per-package copies this replaced each skipped a different aggregation and the
// HTTP one therefore never inspected a counter label at all.
func AssertNoAttributeContains(tb testing.TB, reader *sdkmetric.ManualReader, forbidden ...string) {
	tb.Helper()

	points := 0
	ForEachMetric(tb, reader, func(measured metricdata.Metrics) {
		for _, set := range AttributeSets(tb, measured.Data) {
			points++
			for _, labelled := range set.ToSlice() {
				value := labelled.Value.String()
				for _, candidate := range forbidden {
					if candidate == "" {
						continue
					}
					if strings.Contains(string(labelled.Key), candidate) || strings.Contains(value, candidate) {
						tb.Fatalf(
							"%s attribute %s=%q contains forbidden correlation value %q",
							measured.Name,
							labelled.Key,
							value,
							candidate,
						)
					}
				}
			}
		}
	})
	if points == 0 {
		tb.Fatal("collected metric data points = 0, want recorded metrics")
	}
}

// Attribute is one attribute of a point, failing when it is absent rather than
// returning the empty string a missing and an empty label would share.
func Attribute(tb testing.TB, attributes attribute.Set, name string) string {
	tb.Helper()

	value, present := attributes.Value(attribute.Key(name))
	if !present {
		tb.Fatalf("metric point has no %s attribute", name)
	}
	return value.AsString()
}
