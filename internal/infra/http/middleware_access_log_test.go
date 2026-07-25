package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLogPreservesFirstFinalStatus(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := AccessLog(log, true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	resp := doRequest(handler, http.MethodGet, "/status")

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &event); err != nil {
		t.Fatalf("unmarshal access log: %v", err)
	}
	status, ok := event["status"].(float64)
	if !ok {
		t.Fatalf("logged status type = %T, want float64", event["status"])
	}
	if got := int(status); got != http.StatusNoContent {
		t.Fatalf("logged status = %d, want %d", got, http.StatusNoContent)
	}
}

func TestAccessLogSkipsHealthProbesByDefault(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := mustNewRouter(t, log, Handlers{}, nil, RouterConfig{})

	for _, path := range []string{"/health/live", "/health/ready"} {
		if resp := doRequest(handler, http.MethodGet, path); resp.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, resp.Code, http.StatusOK)
		}
	}

	if out.Len() != 0 {
		t.Fatalf("access log = %q, want health probes excluded", out.String())
	}
}

func TestAccessLogRecordsHealthProbesWhenEnabled(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := mustNewRouter(t, log, Handlers{}, nil, RouterConfig{
		LogHealthProbes: true,
	})

	if resp := doRequest(handler, http.MethodGet, "/health/ready"); resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if !strings.Contains(out.String(), "/health/ready") {
		t.Fatalf("access log = %q, want the health probe recorded", out.String())
	}
}

func TestAccessLogRecordsUnmatchedHealthLookalikePath(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := mustNewRouter(t, log, Handlers{}, nil, RouterConfig{})

	// The skip is route-based, so a path that only looks like a probe is still
	// recorded. Otherwise scanning traffic could hide behind the exclusion.
	if resp := doRequest(handler, http.MethodGet, "/health/live/../../admin"); resp.Code == 0 {
		t.Fatal("request was not served")
	}
	if out.Len() == 0 {
		t.Fatal("access log is empty, want the unmatched request recorded")
	}
}

func TestAccessLogPreservesFlusherInterface(t *testing.T) {
	t.Parallel()

	var directFlusher bool
	var flushErr error
	handler := AccessLog(nil, true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		directFlusher = true
		flusher.Flush()
		flushErr = http.NewResponseController(w).Flush()
	}))

	resp := doRequest(handler, http.MethodGet, "/flush")

	if !directFlusher {
		t.Fatal("wrapped ResponseWriter does not implement http.Flusher")
	}
	if flushErr != nil {
		t.Fatalf("ResponseController.Flush() error = %v, want nil", flushErr)
	}
	if !resp.Flushed {
		t.Fatalf("ResponseRecorder.Flushed = false, want true")
	}
}

func BenchmarkAccessLog(b *testing.B) {
	for _, tc := range []struct {
		name  string
		level slog.Level
	}{
		{name: "enabled", level: slog.LevelInfo},
		{name: "disabled", level: slog.LevelWarn},
	} {
		b.Run(tc.name, func(b *testing.B) {
			//nolint:sloglint // This benchmark needs a level-selectable handler.
			log := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: tc.level}))
			handler := AccessLog(log, true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
			request.Pattern = "GET /health/live"
			response := httptest.NewRecorder()

			b.ReportAllocs()
			for b.Loop() {
				handler.ServeHTTP(response, request)
			}
		})
	}
}
