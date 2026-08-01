package natsjs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationScope = "service.messaging.nats"

type Role string

const (
	RoleProducer Role = "producer"
	RoleWorker   Role = "worker"
)

type Observability struct {
	Logger *slog.Logger
	Meter  metric.Meter
	Tracer trace.Tracer
}

type signals struct {
	log                   *slog.Logger
	tracer                trace.Tracer
	publishOperations     metric.Int64Counter
	publishDuration       metric.Float64Histogram
	connectionEvents      metric.Int64Counter
	readiness             metric.Int64ObservableGauge
	fetchMessages         metric.Int64Counter
	fetchBytes            metric.Int64Counter
	consumeActive         metric.Int64UpDownCounter
	handlerOperations     metric.Int64Counter
	handlerDuration       metric.Float64Histogram
	redeliveries          metric.Int64Counter
	retries               metric.Int64Counter
	dlqTransfers          metric.Int64Counter
	drainOperations       metric.Int64Counter
	forcedShutdowns       metric.Int64Counter
	readinessRegistration metric.Registration
	closeOnce             sync.Once
}

func newSignals(obs Observability, role Role, readiness func() bool) (*signals, error) {
	if role != RoleProducer && role != RoleWorker {
		return nil, fmt.Errorf("%w: invalid messaging role", ErrRejected)
	}
	if obs.Logger == nil {
		obs.Logger = slog.Default()
	}
	if obs.Meter == nil {
		obs.Meter = otel.GetMeterProvider().Meter(instrumentationScope)
	}
	if obs.Tracer == nil {
		obs.Tracer = otel.GetTracerProvider().Tracer(instrumentationScope)
	}
	s := &signals{log: obs.Logger, tracer: obs.Tracer}
	if err := s.registerMetrics(obs.Meter); err != nil {
		return nil, err
	}
	registration, err := obs.Meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		value := int64(0)
		if readiness() {
			value = 1
		}
		observer.ObserveInt64(s.readiness, value, metric.WithAttributes(attribute.String("role", string(role))))
		return nil
	}, s.readiness)
	if err != nil {
		return nil, fmt.Errorf("register messaging readiness metric: %w", err)
	}
	s.readinessRegistration = registration
	return s, nil
}

func (s *signals) registerMetrics(meter metric.Meter) error {
	var err error
	if s.publishOperations, err = meter.Int64Counter("messaging.publish.operations"); err != nil {
		return fmt.Errorf("create messaging publish operations metric: %w", err)
	}
	if s.publishDuration, err = meter.Float64Histogram("messaging.publish.duration", metric.WithUnit("s")); err != nil {
		return fmt.Errorf("create messaging publish duration metric: %w", err)
	}
	if s.connectionEvents, err = meter.Int64Counter("messaging.connection.events"); err != nil {
		return fmt.Errorf("create messaging connection events metric: %w", err)
	}
	if s.readiness, err = meter.Int64ObservableGauge("messaging.readiness"); err != nil {
		return fmt.Errorf("create messaging readiness metric: %w", err)
	}
	if s.fetchMessages, err = meter.Int64Counter("messaging.fetch.messages"); err != nil {
		return fmt.Errorf("create messaging fetch messages metric: %w", err)
	}
	if s.fetchBytes, err = meter.Int64Counter("messaging.fetch.bytes", metric.WithUnit("By")); err != nil {
		return fmt.Errorf("create messaging fetch bytes metric: %w", err)
	}
	if s.consumeActive, err = meter.Int64UpDownCounter("messaging.consume.active"); err != nil {
		return fmt.Errorf("create messaging active consumers metric: %w", err)
	}
	if s.handlerOperations, err = meter.Int64Counter("messaging.handler.operations"); err != nil {
		return fmt.Errorf("create messaging handler operations metric: %w", err)
	}
	if s.handlerDuration, err = meter.Float64Histogram("messaging.handler.duration", metric.WithUnit("s")); err != nil {
		return fmt.Errorf("create messaging handler duration metric: %w", err)
	}
	if s.redeliveries, err = meter.Int64Counter("messaging.redeliveries"); err != nil {
		return fmt.Errorf("create messaging redeliveries metric: %w", err)
	}
	if s.retries, err = meter.Int64Counter("messaging.retries"); err != nil {
		return fmt.Errorf("create messaging retries metric: %w", err)
	}
	if s.dlqTransfers, err = meter.Int64Counter("messaging.dlq.transfers"); err != nil {
		return fmt.Errorf("create messaging dead-letter transfers metric: %w", err)
	}
	if s.drainOperations, err = meter.Int64Counter("messaging.drain.operations"); err != nil {
		return fmt.Errorf("create messaging drain operations metric: %w", err)
	}
	if s.forcedShutdowns, err = meter.Int64Counter("messaging.forced_shutdowns"); err != nil {
		return fmt.Errorf("create messaging forced shutdowns metric: %w", err)
	}
	return nil
}

func (s *signals) close() {
	if s != nil && s.readinessRegistration != nil {
		s.closeOnce.Do(func() { _ = s.readinessRegistration.Unregister() })
	}
}

func (s *signals) publish(ctx context.Context, event Event, outcome, reason string, started time.Time) {
	duration := time.Since(started).Seconds()
	attrs := metric.WithAttributes(attribute.String("outcome", outcome))
	s.publishOperations.Add(ctx, 1, attrs)
	s.publishDuration.Record(ctx, duration, attrs)
	s.log.InfoContext(ctx, "messaging_publish",
		"operation", "publish", "message_id", event.MessageID, "subject", event.Subject,
		"outcome", outcome, "duration_seconds", duration, "reason", reason,
	)
}

func (s *signals) connection(ctx context.Context, event string) {
	s.connectionEvents.Add(ctx, 1, metric.WithAttributes(attribute.String("event", event)))
	s.log.InfoContext(ctx, "messaging_connection", "operation", "connection", "outcome", event)
}

func (s *signals) handler(ctx context.Context, msg Message, outcome, reason string, started time.Time) {
	duration := time.Since(started).Seconds()
	attrs := metric.WithAttributes(attribute.String("outcome", outcome))
	s.handlerOperations.Add(ctx, 1, attrs)
	s.handlerDuration.Record(ctx, duration, attrs)
	s.log.InfoContext(ctx, "messaging_delivery",
		"operation", "consume", "message_id", msg.MessageID(), "subject", msg.Subject(),
		"consumer", msg.Metadata().Consumer, "attempt", msg.Metadata().NumDelivered,
		"outcome", outcome, "duration_seconds", duration, "reason", reason,
	)
}
