package httpx

import (
	"net/http/httptest"
	"testing"
)

func TestHTTPMethodNormalizationBoundsNonStandardRouteLabels(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("BREW", "/coffee", nil)
	req.Pattern = "BREW /coffee"
	if got := requestMethodLabel(req); got != "OTHER" {
		t.Fatalf("requestMethodLabel() = %q, want %q", got, "OTHER")
	}
	if got := routePathTemplateForRequest(req); got != "/coffee" {
		t.Fatalf("routePathTemplateForRequest() = %q, want %q", got, "/coffee")
	}
}
