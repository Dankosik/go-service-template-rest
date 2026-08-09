package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// InstrumentSet registers a package's metric instruments and keeps the first
// failure, so a constructor reads as the list of instruments it owns.
//
//	set := telemetry.NewInstrumentSet(meter)
//	set.Int64Counter(&s.publishes, "messaging.publish.operations")
//	set.Float64Histogram(&s.duration, "messaging.publish.duration", metric.WithUnit("s"))
//	if err := set.Err(); err != nil {
//		return nil, err
//	}
//
// The failure names the metric itself. The per-instrument error branches this
// replaced each spelled that name a second time in English, and nothing kept the
// two in step — so a rename left the operator an error naming a metric no
// dashboard has.
//
// Registration stops at the first failure. A meter that cannot serve one
// instrument does not serve the next either, and continuing would report the
// last name rather than the one that broke.
type InstrumentSet struct {
	meter metric.Meter
	err   error
}

func NewInstrumentSet(meter metric.Meter) *InstrumentSet {
	return &InstrumentSet{meter: meter}
}

// MeterOrGlobal returns meter, or one from the global provider named for scope
// when the caller supplied none.
//
// An adapter takes its meter as an option so a test can hand it a manual reader,
// and falls back to the global provider the composition root installed. Both
// adapters that do this wrote the fallback themselves, which is one place each
// for a third to copy it from and name the wrong scope while doing so.
//
//nolint:ireturn // metric.Meter is OTel's own interface; a provider returns nothing else.
func MeterOrGlobal(meter metric.Meter, scope string) metric.Meter {
	if meter != nil {
		return meter
	}
	return otel.GetMeterProvider().Meter(scope)
}

// Err reports the first registration that failed, or nil when every instrument
// was created.
func (s *InstrumentSet) Err() error {
	return s.err
}

// register is the whole of what every method below does. The five kinds differ
// only in the instrument type and its option type, which is why this is one
// generic function and not five bodies: a method cannot take type parameters, so
// each method stays as the typed name a caller reads and delegates here.
func register[Instrument, Option any](
	set *InstrumentSet,
	target *Instrument,
	name string,
	create func(string, ...Option) (Instrument, error),
	options []Option,
) {
	if set.err != nil {
		return
	}
	instrument, err := create(name, options...)
	if err != nil {
		set.err = fmt.Errorf("create %s metric: %w", name, err)
		return
	}
	*target = instrument
}

func (s *InstrumentSet) Int64Counter(target *metric.Int64Counter, name string, options ...metric.Int64CounterOption) {
	register(s, target, name, s.meter.Int64Counter, options)
}

func (s *InstrumentSet) Int64UpDownCounter(
	target *metric.Int64UpDownCounter,
	name string,
	options ...metric.Int64UpDownCounterOption,
) {
	register(s, target, name, s.meter.Int64UpDownCounter, options)
}

func (s *InstrumentSet) Int64ObservableGauge(
	target *metric.Int64ObservableGauge,
	name string,
	options ...metric.Int64ObservableGaugeOption,
) {
	register(s, target, name, s.meter.Int64ObservableGauge, options)
}

func (s *InstrumentSet) Float64Histogram(
	target *metric.Float64Histogram,
	name string,
	options ...metric.Float64HistogramOption,
) {
	register(s, target, name, s.meter.Float64Histogram, options)
}

func (s *InstrumentSet) Float64ObservableGauge(
	target *metric.Float64ObservableGauge,
	name string,
	options ...metric.Float64ObservableGaugeOption,
) {
	register(s, target, name, s.meter.Float64ObservableGauge, options)
}
