package oidcjwt

// The cost of recording one verification, which is the only per-request work
// this package does unconditionally.
//
// verificationSets claims a specific allocation count for prebuilding the
// attribute sets, and a comment cannot hold itself to that. These benchmarks
// are where the claim is checked, against a real SDK meter rather than a no-op
// one, because the no-op counter discards its options and would report the
// prebuild as free on both sides.

import (
	"context"
	"log/slog"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func benchmarkAuthnMetrics(b *testing.B) authnMetrics {
	b.Helper()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewManualReader()))
	b.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			b.Errorf("shutdown metric provider: %v", err)
		}
	})
	return newAuthnMetrics(provider, newDegradedWarning(slog.New(slog.DiscardHandler)))
}

// BenchmarkRecordVerificationSuccess measures the path every accepted request
// takes.
func BenchmarkRecordVerificationSuccess(b *testing.B) {
	metrics := benchmarkAuthnMetrics(b)
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
	metrics := benchmarkAuthnMetrics(b)
	ctx := b.Context()
	err := failure(KindInvalid)
	b.ReportAllocs()
	for b.Loop() {
		metrics.recordVerification(ctx, transportHTTP, err)
	}
}
