package oidcjwt

// The cost of recording one verification, which is the only per-request work
// this package does unconditionally.
//
// [verificationSets] exists to keep that cost down, and a comment cannot hold
// itself to a claim like that. TestPrebuiltAttributeSetsBeatPerCallConstruction
// below is where it is held; the benchmarks report what the path actually costs.
// Both measure against a real SDK meter rather than a no-op one, because the
// no-op counter discards its options and would report the prebuild as free on
// both sides.

import (
	"context"
	"log/slog"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// TestPrebuiltAttributeSetsBeatPerCallConstruction holds the trade
// [verificationSets] is built for.
//
// It compares the two paths rather than pinning an allocation count, because
// the count moves with the Go and OpenTelemetry versions while the trade does
// not, and because the race detector changes both sides alike but neither
// ratio. Deleting the prebuild makes recordVerification take its own
// newVerificationSets fallback, and the two measurements converge — which is
// exactly the regression worth failing on.
func TestPrebuiltAttributeSetsBeatPerCallConstruction(t *testing.T) {
	metrics := verificationCostFixture(t)
	ctx := t.Context()
	err := failure(KindInvalid)

	prebuilt := testing.AllocsPerRun(200, func() {
		metrics.recordVerification(ctx, transportHTTP, err)
	})
	// transportGRPC is prebuilt too, so an unprebuilt carrier has to be named
	// here to measure the fallback recordVerification would take for all of them.
	perCall := testing.AllocsPerRun(200, func() {
		metrics.recordVerification(ctx, transport("unprebuilt"), err)
	})

	if prebuilt >= perCall {
		t.Errorf(
			"prebuilt carrier allocates %.0f per record and an unprebuilt one allocates %.0f; "+
				"the prebuild in newAuthnMetrics is no longer buying anything",
			prebuilt, perCall,
		)
	}
}

func verificationCostFixture(tb testing.TB) authnMetrics {
	tb.Helper()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
	tb.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			tb.Errorf("shutdown metric provider: %v", err)
		}
	})
	return newAuthnMetrics(provider, newDegradedWarning(slog.New(slog.DiscardHandler)))
}

// BenchmarkRecordVerificationSuccess measures the path every accepted request
// takes.
func BenchmarkRecordVerificationSuccess(b *testing.B) {
	metrics := verificationCostFixture(b)
	ctx := b.Context()
	b.ReportAllocs()
	for b.Loop() {
		metrics.recordVerification(ctx, transportHTTP, nil)
	}
}

// BenchmarkRecordVerificationFailure measures the path a refused request takes,
// which is the majority path while a service is being probed for credentials.
// It costs one more allocation than the success path because it carries the
// reason label as well.
func BenchmarkRecordVerificationFailure(b *testing.B) {
	metrics := verificationCostFixture(b)
	ctx := b.Context()
	err := failure(KindInvalid)
	b.ReportAllocs()
	for b.Loop() {
		metrics.recordVerification(ctx, transportHTTP, err)
	}
}
