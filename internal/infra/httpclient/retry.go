package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const idempotencyKeyHeader = "Idempotency-Key"

// retryJitterFraction randomizes half of each delay: full jitter makes the first
// retry nearly immediate, and none makes a fleet retry in lockstep.
const retryJitterFraction = 0.5

// RetryPolicy bounds retries for one client. It is disabled by default because
// how many attempts a dependency can absorb is a decision per dependency.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts, not the number of extra ones.
	// Zero or one disables retrying.
	MaxAttempts int
	// BaseDelay is the first backoff interval. Each later attempt doubles it before
	// jitter is applied.
	BaseDelay time.Duration
}

func (p RetryPolicy) enabled() bool {
	return p.MaxAttempts > 1 && p.BaseDelay > 0
}

func (p RetryPolicy) validate() error {
	if p.MaxAttempts < 0 {
		return errors.New("build outbound HTTP client: retry max attempts must be >= 0")
	}
	if p.BaseDelay < 0 {
		return errors.New("build outbound HTTP client: retry base delay must be >= 0")
	}
	// A policy that names attempts without a delay would retry immediately, which
	// is the shape that turns one failure into a burst.
	if p.MaxAttempts > 1 && p.BaseDelay <= 0 {
		return errors.New("build outbound HTTP client: retry base delay is required when max attempts is > 1")
	}
	return nil
}

// retryTransport repeats a request that failed in a way a repeat could fix. Three
// rules keep that safe, each enforced by the function that owns it:
// repeatableRequest, fitsRemainingBudget, and retryAfter.
type retryTransport struct {
	base   http.RoundTripper
	policy RetryPolicy
}

//nolint:wrapcheck // The wrapped transports below already added their context, and retryableResult classifies on error identity.
func (t retryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if !t.policy.enabled() || !repeatableRequest(request) {
		return response, err
	}

	for attempt := 2; attempt <= t.policy.MaxAttempts; attempt++ {
		if !retryableResult(response, err) {
			return response, err
		}

		delay := retryDelay(t.policy.BaseDelay, attempt, response)
		if !fitsRemainingBudget(request, delay) {
			return response, err
		}
		// Only once another attempt is certain, so a caller never receives a
		// response whose body this closed.
		drainResponse(response)

		select {
		case <-request.Context().Done():
			return nil, context.Cause(request.Context())
		case <-time.After(delay):
		}

		response, err = t.retryRequest(request)
	}

	return response, err
}

// retryRequest rebuilds the request. repeatableRequest has already established
// that the body can be rewound.
func (t retryTransport) retryRequest(request *http.Request) (*http.Response, error) {
	next := request.Clone(request.Context())
	if hasBody(request) {
		body, err := request.GetBody()
		if err != nil {
			return nil, fmt.Errorf("rewind outbound HTTP request body: %w", err)
		}
		next.Body = body
	}

	//nolint:wrapcheck // The transport's own error is what the caller classifies.
	return t.base.RoundTrip(next)
}

// repeatableRequest reports whether repeating this request is safe.
//
// Anything but GET, HEAD, or OPTIONS needs an Idempotency-Key: only the caller
// knows whether the upstream operation made that key durable, and without it a
// retried POST the server did process creates a second resource.
//
// A body must also be rewindable, and only GetBody can do that. net/http populates
// it for the body types it knows, so an opaque io.Reader gets one attempt rather
// than a silent retry of an empty body.
func repeatableRequest(request *http.Request) bool {
	if request == nil {
		return false
	}
	if hasBody(request) && request.GetBody == nil {
		return false
	}
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return hasUsableIdempotencyKey(request)
}

// hasBlankIdempotencyKey applies whether or not retry is enabled: a blank key is
// a caller mistake regardless, and rejecting it early avoids a client that only
// reveals the mistake once a retry is attempted.
func hasBlankIdempotencyKey(request *http.Request) bool {
	if request == nil {
		return false
	}
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	present, usable := idempotencyKeyState(request.Header)
	return present && !usable
}

func hasUsableIdempotencyKey(request *http.Request) bool {
	if request == nil {
		return false
	}
	_, usable := idempotencyKeyState(request.Header)
	return usable
}

func idempotencyKeyState(header http.Header) (present bool, usable bool) {
	for name, values := range header {
		if !strings.EqualFold(name, idempotencyKeyHeader) {
			continue
		}
		present = true
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				return true, true
			}
		}
	}
	return present, false
}

func hasBody(request *http.Request) bool {
	return request.Body != nil && request.Body != http.NoBody
}

// retryableResult reports whether another attempt could plausibly succeed. Only
// statuses meaning "not now" qualify. A 500 is deliberately absent: it usually
// means the server did the work and failed partway, and repeating it repeats that.
func retryableResult(response *http.Response, err error) bool {
	if err != nil {
		// A denied target or a body over its cap will fail identically forever.
		return !errors.Is(err, ErrTargetDenied) && !isResponseTooLarge(err)
	}
	if response == nil {
		return false
	}
	switch response.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

func isResponseTooLarge(err error) bool {
	_, ok := errors.AsType[*ResponseTooLargeError](err)
	return ok
}

func retryDelay(base time.Duration, attempt int, response *http.Response) time.Duration {
	if hinted, ok := retryAfter(response, time.Now()); ok {
		return hinted
	}

	// attempt is 2 for the first retry, so the first delay is the base itself.
	backoff := base << (attempt - 2)
	if backoff <= 0 {
		backoff = base // The shift overflowed on an absurd attempt count.
	}
	// #nosec G404 -- Jitter needs spread across replicas, not unpredictability; nothing here is a secret or a token.
	jitter := time.Duration(rand.Float64() * retryJitterFraction * float64(backoff))
	return backoff - time.Duration(retryJitterFraction*float64(backoff)) + jitter
}

// retryAfter reads either Retry-After form. A valid server Date is the reference
// for an HTTP-date hint, so ordinary client/server clock skew does not distort it.
func retryAfter(response *http.Response, now time.Time) (time.Duration, bool) {
	if response == nil {
		return 0, false
	}
	raw := response.Header.Get("Retry-After")
	if raw == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	retryAt, err := http.ParseTime(raw)
	if err != nil {
		return 0, false
	}
	reference := now
	if serverDate, dateErr := http.ParseTime(response.Header.Get("Date")); dateErr == nil {
		reference = serverDate
	}
	delay := retryAt.Sub(reference)
	return delay, delay >= 0
}

// fitsRemainingBudget reports whether the delay leaves room for another attempt.
// The margin is the delay itself: an attempt given less time than it spent
// waiting cannot finish, and a retry outliving its request holds a handler
// goroutine and its in-flight slot for an answer nobody awaits.
func fitsRemainingBudget(request *http.Request, delay time.Duration) bool {
	deadline, ok := request.Context().Deadline()
	if !ok {
		return true
	}
	return time.Until(deadline) > 2*delay
}

// maxDrainBytes bounds the drain read: past this, rereading costs more than the
// one handshake draining exists to save.
const maxDrainBytes = 64 << 10

// drainResponse consumes and closes a response that is about to be replaced, so
// its connection returns to the pool instead of being torn down. The read is what
// matters: net/http discards a connection whose body was closed before EOF, so
// closing alone makes every retry pay a fresh TCP and TLS handshake.
func drainResponse(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxDrainBytes))
	_ = response.Body.Close()
}
