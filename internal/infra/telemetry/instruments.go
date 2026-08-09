package telemetry

import (
	"fmt"

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

// Err reports the first registration that failed, or nil when every instrument
// was created.
func (s *InstrumentSet) Err() error {
	return s.err
}

func (s *InstrumentSet) Int64Counter(target *metric.Int64Counter, name string, options ...metric.Int64CounterOption) {
	if s.err != nil {
		return
	}
	instrument, err := s.meter.Int64Counter(name, options...)
	if err != nil {
		s.err = fmt.Errorf("create %s metric: %w", name, err)
		return
	}
	*target = instrument
}

func (s *InstrumentSet) Int64UpDownCounter(
	target *metric.Int64UpDownCounter,
	name string,
	options ...metric.Int64UpDownCounterOption,
) {
	if s.err != nil {
		return
	}
	instrument, err := s.meter.Int64UpDownCounter(name, options...)
	if err != nil {
		s.err = fmt.Errorf("create %s metric: %w", name, err)
		return
	}
	*target = instrument
}

func (s *InstrumentSet) Int64ObservableGauge(
	target *metric.Int64ObservableGauge,
	name string,
	options ...metric.Int64ObservableGaugeOption,
) {
	if s.err != nil {
		return
	}
	instrument, err := s.meter.Int64ObservableGauge(name, options...)
	if err != nil {
		s.err = fmt.Errorf("create %s metric: %w", name, err)
		return
	}
	*target = instrument
}

func (s *InstrumentSet) Float64Histogram(
	target *metric.Float64Histogram,
	name string,
	options ...metric.Float64HistogramOption,
) {
	if s.err != nil {
		return
	}
	instrument, err := s.meter.Float64Histogram(name, options...)
	if err != nil {
		s.err = fmt.Errorf("create %s metric: %w", name, err)
		return
	}
	*target = instrument
}

func (s *InstrumentSet) Float64ObservableGauge(
	target *metric.Float64ObservableGauge,
	name string,
	options ...metric.Float64ObservableGaugeOption,
) {
	if s.err != nil {
		return
	}
	instrument, err := s.meter.Float64ObservableGauge(name, options...)
	if err != nil {
		s.err = fmt.Errorf("create %s metric: %w", name, err)
		return
	}
	*target = instrument
}
