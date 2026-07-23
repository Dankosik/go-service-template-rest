package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestAccessLogPreservesFirstFinalStatus(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := AccessLog(log, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestAccessLogPreservesFlusherInterface(t *testing.T) {
	t.Parallel()

	var directFlusher bool
	var flushErr error
	handler := AccessLog(nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
