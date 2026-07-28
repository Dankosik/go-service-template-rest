package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
)

var (
	errArticleMissing = errors.New("article not found")
	errPoolSaturated  = errors.New("pool saturated")
	errPoolBackedUp   = errors.New("pool backed up")
)

func classifyTestDomainError(err error) (problem.Mapped, bool) {
	switch {
	case errors.Is(err, errArticleMissing):
		return problem.Mapped{Code: problem.CodeNotFound, Detail: "article was not found"}, true
	case errors.Is(err, errPoolSaturated):
		return problem.Mapped{
			Code:       problem.CodeServiceUnavailable,
			Detail:     "the service is temporarily at capacity",
			RetryAfter: 100 * time.Millisecond,
		}, true
	case errors.Is(err, errPoolBackedUp):
		return problem.Mapped{
			Code:       problem.CodeServiceUnavailable,
			Detail:     "the service is temporarily at capacity",
			RetryAfter: 1100 * time.Millisecond,
		}, true
	default:
		return problem.Mapped{}, false
	}
}

// TestRejectResponseClassifiesDomainErrors is the seam's whole point: one table
// decides what a domain failure looks like to a client, instead of a switch per
// operation building a nested generated literal.
func TestRejectResponseClassifiesDomainErrors(t *testing.T) {
	t.Parallel()

	reject := RejectResponse(classifyTestDomainError)

	for _, tc := range []struct {
		name           string
		err            error
		wantStatus     int
		wantCode       problem.Code
		wantDetail     string
		wantRetryAfter string
	}{
		{
			name:       "classified sentinel",
			err:        fmt.Errorf("get article: %w", errArticleMissing),
			wantStatus: http.StatusNotFound,
			wantCode:   problem.CodeNotFound,
			wantDetail: "article was not found",
		},
		{
			// The hint is what tells a client library to try again. Without it a
			// 503 reads as "down", and a saturated dependency that would have
			// cleared in a second loses the traffic that was about to succeed.
			name:           "classified with a retry hint",
			err:            fmt.Errorf("create article: %w", errPoolSaturated),
			wantStatus:     http.StatusServiceUnavailable,
			wantCode:       problem.CodeServiceUnavailable,
			wantDetail:     "the service is temporarily at capacity",
			wantRetryAfter: "1",
		},
		{
			name:           "classified with a fractional retry hint",
			err:            fmt.Errorf("create article: %w", errPoolBackedUp),
			wantStatus:     http.StatusServiceUnavailable,
			wantCode:       problem.CodeServiceUnavailable,
			wantDetail:     "the service is temporarily at capacity",
			wantRetryAfter: "2",
		},
		{
			// A fault nothing anticipated stays a 500 with no detail. Guessing a
			// friendlier status for it is how a bug starts reading as a client
			// mistake.
			name:       "unclassified error",
			err:        errors.New("something nobody mapped"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   problem.CodeInternalError,
			wantDetail: "request failed",
		},
		{
			// Checked before any mapper, because it is a transport fact: a mapper
			// that classified it would have to repeat the rule, and one that
			// forgot would hide every slow dependency inside the 5xx rate.
			name:       "spent request budget",
			err:        fmt.Errorf("query articles: %w", context.DeadlineExceeded),
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   problem.CodeGatewayTimeout,
			wantDetail: "request exceeded its time budget",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			reject(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/x", nil), tc.err)

			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tc.wantStatus, response.Body.String())
			}
			if got := response.Header().Get("Retry-After"); got != tc.wantRetryAfter {
				t.Fatalf("Retry-After = %q, want %q", got, tc.wantRetryAfter)
			}

			var body struct {
				Code   string `json:"code"`
				Detail string `json:"detail"`
				Status int    `json:"status"`
				Type   string `json:"type"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if body.Code != string(tc.wantCode) || body.Status != tc.wantStatus {
				t.Fatalf("problem = %+v, want code %q and status %d", body, tc.wantCode, tc.wantStatus)
			}
			if body.Detail != tc.wantDetail {
				t.Fatalf("detail = %q, want %q", body.Detail, tc.wantDetail)
			}
			// The type URI comes from the shared catalog rather than the mapper,
			// which is what stopped a 409 from advertising the internal-error
			// class while carrying status 409.
			definition, ok := problem.ForCode(tc.wantCode)
			if !ok {
				t.Fatalf("catalog has no entry for %q", tc.wantCode)
			}
			if body.Type != definition.TypeURI {
				t.Fatalf("type = %q, want %q", body.Type, definition.TypeURI)
			}
		})
	}
}

// TestRejectResponseUsesTheFirstMatchingMapper keeps mapper order meaningful, so
// a service can put a specific classification ahead of a broad one without every
// mapper knowing about the others.
func TestRejectResponseUsesTheFirstMatchingMapper(t *testing.T) {
	t.Parallel()

	specific := func(error) (problem.Mapped, bool) {
		return problem.Mapped{Code: problem.CodeConflict, Detail: "specific"}, true
	}
	broad := func(error) (problem.Mapped, bool) {
		return problem.Mapped{Code: problem.CodeBadRequest, Detail: "broad"}, true
	}

	response := httptest.NewRecorder()
	// The nil is deliberate: a profile that supplies no mapper must not have to
	// be filtered out by the caller.
	RejectResponse(nil, specific, broad)(
		response,
		httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/articles", nil),
		errors.New("boom"),
	)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d from the first matching mapper", response.Code, http.StatusConflict)
	}
}

// TestRejectResponseWithoutMappersKeepsTheTransportFallback pins the behavior a
// service that installs nothing still gets.
func TestRejectResponseWithoutMappersKeepsTheTransportFallback(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	RejectResponse()(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/x", nil), errors.New("boom"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
