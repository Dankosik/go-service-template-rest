package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/problem"
)

var (
	errArticleMissing = errors.New("article not found")
	errArticleExists  = errors.New("article already exists")
	errPoolSaturated  = errors.New("pool saturated")
	errPoolBackedUp   = errors.New("pool backed up")
)

func classifyTestDomainError(err error) (failure.Classification, bool) {
	switch {
	case errors.Is(err, errArticleMissing):
		return failure.Classification{Code: failure.CodeNotFound, Detail: "article was not found"}, true
	case errors.Is(err, errArticleExists):
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "article already exists"}, true
	case errors.Is(err, errPoolSaturated):
		return failure.Classification{
			Code:       failure.CodeServiceUnavailable,
			Detail:     "the service is temporarily at capacity",
			RetryAfter: 100 * time.Millisecond,
		}, true
	case errors.Is(err, errPoolBackedUp):
		return failure.Classification{
			Code:       failure.CodeServiceUnavailable,
			Detail:     "the service is temporarily at capacity",
			RetryAfter: 1100 * time.Millisecond,
		}, true
	default:
		return failure.Classification{}, false
	}
}

// TestRejectResponseClassifiesDomainErrors is the seam's whole point: one table
// decides what a domain failure looks like to a client, instead of a switch per
// operation building a nested generated literal.
func TestRejectResponseClassifiesDomainErrors(t *testing.T) {
	t.Parallel()

	// Discarded here: what this table pins is the response, and the record the
	// unclassified case writes has its own test below.
	reject := RejectResponse(slog.New(slog.DiscardHandler), classifyTestDomainError)

	for _, tc := range []struct {
		name           string
		err            error
		wantStatus     int
		wantCode       problem.Code
		wantDetail     string
		wantRetryAfter string
	}{
		{
			name:       "classified already exists",
			err:        fmt.Errorf("create article: %w", errArticleExists),
			wantStatus: http.StatusConflict,
			wantCode:   problem.CodeAlreadyExists,
			wantDetail: "article already exists",
		},
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
		{
			// Same reason as the budget above, for the condition that reaches this
			// handler far more often: a caller that hung up. Answering it as an
			// unclassified 500 puts every abandoned request in the error rate.
			name:       "caller cancellation",
			err:        fmt.Errorf("query articles: %w", context.Canceled),
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   problem.CodeGatewayTimeout,
			wantDetail: "request was canceled by the caller",
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

	specific := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "specific"}, true
	}
	broad := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeBadRequest, Detail: "broad"}, true
	}

	response := httptest.NewRecorder()
	// The nil mapper after the logger is deliberate: a profile that supplies no
	// mapper must not have to be filtered out by the caller.
	RejectResponse(slog.New(slog.DiscardHandler), nil, specific, broad)(
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
	RejectResponse(slog.New(slog.DiscardHandler))(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/x", nil), errors.New("boom"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

// TestRejectResponseRecordsTheUnclassifiedFailure is the test that fails if the
// 500 path ever goes quiet again.
//
// The client-visible half of this behavior — a detail-free "request failed" — is
// deliberate and already pinned above. The cost of that sanitization is that the
// response is evidence of nothing, so this record is what stands between an
// operator and reproducing the request to find out what broke.
//
// It asserts both halves of the bargain: the chain names the dependency that
// failed, and the error's own text never appears. The secret here stands in for
// what a real handler's error carries — a DSN with a password, a token, a row.
func TestRejectResponseRecordsTheUnclassifiedFailure(t *testing.T) {
	t.Parallel()

	const secretCanary = "password=response-path-secret"

	var logged bytes.Buffer
	cause := fmt.Errorf("create article: %w", &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New(secretCanary),
	})

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/articles", nil)
	request.Pattern = "POST /api/v1/articles"
	RejectResponse(newTestServiceLogger(&logged), classifyTestDomainError)(httptest.NewRecorder(), request, cause)

	if strings.Contains(logged.String(), secretCanary) {
		t.Fatalf("record discloses the error's own text: %s", logged.String())
	}

	var record struct {
		Level      string `json:"level"`
		Message    string `json:"msg"`
		Route      string `json:"route"`
		ErrorChain string `json:"error_chain"`
	}
	if err := json.Unmarshal(logged.Bytes(), &record); err != nil {
		t.Fatalf("decode record %q: %v", logged.String(), err)
	}
	if record.Level != slog.LevelError.String() {
		t.Errorf("level = %q, want %q: an unanticipated fault is not routine traffic", record.Level, slog.LevelError)
	}
	// The wrapper alone is what a %T record would have said, and it is the same
	// for every failure in the repository. Naming the type underneath is the
	// whole point of the chain.
	if !strings.Contains(record.ErrorChain, "*net.OpError") {
		t.Errorf("error_chain = %q, want the wrapped dependency type", record.ErrorChain)
	}
	if record.Route != "POST /api/v1/articles" {
		t.Errorf("route = %q, want the matched template", record.Route)
	}
	if record.Message == "" {
		t.Error("record has no message")
	}
}

// TestRejectResponseDoesNotRecordAClassifiedFailure keeps the error stream
// usable. A 404 is an answer this service chose, and the access log already
// carries its problem code; repeating it at ERROR is how an error rate stops
// meaning anything.
func TestRejectResponseDoesNotRecordAClassifiedFailure(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/x", nil)
	RejectResponse(newTestServiceLogger(&logged), classifyTestDomainError)(
		httptest.NewRecorder(),
		request,
		fmt.Errorf("get article: %w", errArticleMissing),
	)

	if logged.Len() != 0 {
		t.Fatalf("classified failure wrote %q, want no record", logged.String())
	}
}

// TestRejectResponseDoesNotRecordACallerAbort is the other half of the
// cancellation fix, and the half that actually costs an operator something.
//
// The status was the visible symptom; this is the one that fires during an
// incident. A caller hanging up is not a fault this service can act on, so it
// must not spend an ERROR record — at the volume a disconnecting client
// produces, that record is what makes the error stream unreadable exactly when
// it is being read.
func TestRejectResponseDoesNotRecordACallerAbort(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "caller cancellation", err: fmt.Errorf("query articles: %w", context.Canceled)},
		{name: "spent request budget", err: fmt.Errorf("query articles: %w", context.DeadlineExceeded)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var logged bytes.Buffer
			RejectResponse(newTestServiceLogger(&logged), classifyTestDomainError)(
				httptest.NewRecorder(),
				httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/x", nil),
				tc.err,
			)

			if logged.Len() != 0 {
				t.Fatalf("caller-owned outcome wrote %q, want no record", logged.String())
			}
		})
	}
}
