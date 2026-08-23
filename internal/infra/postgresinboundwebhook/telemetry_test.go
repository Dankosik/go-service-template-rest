// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestInboundWebhookMetricVocabulary(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	telem := newTelemetry(provider, nil)
	ctx := context.Background()
	for _, outcome := range []string{"accepted", "duplicate", "rejected"} {
		telem.recordIngress(ctx, outcome)
	}
	for _, outcome := range []string{"quarantined", "retrying", "handled", "failed"} {
		telem.recordProcessing(ctx, outcome)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int64{}
	for _, scope := range rm.ScopeMetrics {
		for _, metric := range scope.Metrics {
			sum, ok := metric.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
			for _, point := range sum.DataPoints {
				attrs := point.Attributes.ToSlice()
				if len(attrs) != 1 || string(attrs[0].Key) != outcomeAttr {
					t.Fatalf("unexpected attributes: %#v", attrs)
				}
				seen[metric.Name+"/"+attrs[0].Value.AsString()] += point.Value
			}
		}
	}
	for _, key := range []string{
		ingressInstrument + "/accepted",
		ingressInstrument + "/duplicate",
		ingressInstrument + "/rejected",
		processingInstrument + "/quarantined",
		processingInstrument + "/retrying",
		processingInstrument + "/handled",
		processingInstrument + "/failed",
	} {
		if seen[key] != 1 {
			t.Fatalf("%s = %d, want 1", key, seen[key])
		}
	}
}

// profile:inbound-webhooks-standard:end
