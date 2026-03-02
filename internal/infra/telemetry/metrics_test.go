package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsExposeHTTPDurationHistogram(t *testing.T) {
	m := New()
	m.ObserveHTTPRequestDuration(http.MethodGet, "GET /api/v1/ping", http.StatusOK, 25*time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	resp := httptest.NewRecorder()
	m.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}

	body := resp.Body.String()
	if !strings.Contains(body, "http_request_duration_seconds_bucket") {
		t.Fatalf("metrics output does not contain duration histogram buckets")
	}
	if !strings.Contains(body, "http_request_duration_seconds_sum") {
		t.Fatalf("metrics output does not contain duration histogram sum")
	}
	if !strings.Contains(body, "http_request_duration_seconds_count") {
		t.Fatalf("metrics output does not contain duration histogram count")
	}
	if !strings.Contains(body, `method="GET",route="GET /api/v1/ping",status_code="200"`) {
		t.Fatalf("metrics output does not contain expected labels for ping endpoint")
	}
}
