package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
)

type Handlers struct {
	Health *health.Service
	Ping   *ping.Service
}

func (h Handlers) Live(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "ok")
}

func (h Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	if h.Health == nil {
		writeText(w, http.StatusOK, "ok")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.Health.Ready(ctx); err != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	writeText(w, http.StatusOK, "ok")
}

func (h Handlers) Pong(w http.ResponseWriter, _ *http.Request) {
	if h.Ping == nil {
		http.Error(w, "ping service is not configured", http.StatusInternalServerError)
		return
	}
	writeText(w, http.StatusOK, h.Ping.Pong())
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
