// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

const (
	ingressMeterName     = "service.inbound_webhooks.ingress"
	processingMeterName  = "service.inbound_webhooks.processing"
	ingressInstrument    = "inbound_webhooks.ingress_outcomes"
	processingInstrument = "inbound_webhooks.processing_outcomes"
	outcomeAttr          = "outcome"
)

type telemetry struct {
	ingress    metric.Int64Counter
	processing metric.Int64Counter
	log        *slog.Logger
}

func newTelemetry(meter metric.MeterProvider, log *slog.Logger) telemetry {
	if meter == nil {
		meter = metricnoop.NewMeterProvider()
	}
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	ingress, _ := meter.Meter(ingressMeterName).Int64Counter(
		ingressInstrument,
		metric.WithDescription("Inbound webhook ingress outcomes."),
		metric.WithUnit("{outcome}"),
	)
	processing, _ := meter.Meter(processingMeterName).Int64Counter(
		processingInstrument,
		metric.WithDescription("Inbound webhook processing outcomes."),
		metric.WithUnit("{outcome}"),
	)
	return telemetry{ingress: ingress, processing: processing, log: log}
}

func (t telemetry) recordIngress(ctx context.Context, outcome string) {
	if t.ingress == nil {
		return
	}
	t.ingress.Add(ctx, 1, metric.WithAttributes(attribute.String(outcomeAttr, outcome)))
}

func (t telemetry) recordProcessing(ctx context.Context, outcome string) {
	if t.processing == nil {
		return
	}
	t.processing.Add(ctx, 1, metric.WithAttributes(attribute.String(outcomeAttr, outcome)))
}

func (t telemetry) logFailure(ctx context.Context, receiptID, class string) {
	t.log.LogAttrs(ctx, slog.LevelWarn, "inbound_webhook_processing_failed",
		slog.String("receipt_id", receiptID),
		slog.String("failure_class", class),
	)
}

// profile:inbound-webhooks-standard:end
