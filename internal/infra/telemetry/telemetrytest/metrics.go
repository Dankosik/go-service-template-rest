package telemetrytest

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// InstallManualReader installs a process-wide meter provider backed by a
// manual reader so tests can collect metrics on demand. The previous meter
// provider is restored and the temporary provider is shut down when the test
// finishes.
//
// Its only consumer is the container-backed PostgreSQL integration suite, so
// this file is removed with the DATABASE=none profile.
func InstallManualReader(tb testing.TB) *sdkmetric.ManualReader {
	tb.Helper()

	previousMeterProvider := otel.GetMeterProvider()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	tb.Cleanup(func() {
		otel.SetMeterProvider(previousMeterProvider)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), providerShutdownTimeout)
		defer cancel()
		if err := provider.Shutdown(shutdownCtx); err != nil {
			tb.Errorf("shutdown test meter provider: %v", err)
		}
	})
	return reader
}
