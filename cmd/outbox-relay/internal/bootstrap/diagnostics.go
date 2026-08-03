package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func newDiagnosticsServer(addr string, ready func() bool, metrics *telemetry.Metrics) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /health/ready", func(writer http.ResponseWriter, _ *http.Request) {
		if !ready() {
			http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /metrics", metrics.Handler())
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func listenDiagnostics(ctx context.Context, addr string) (net.Listener, error) {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for outbox diagnostics: %w", err)
	}
	return listener, nil
}
