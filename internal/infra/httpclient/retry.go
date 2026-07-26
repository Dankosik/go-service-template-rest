package httpclient

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// idempotencyKeyHeader marks a request the caller has made safe to repeat. It is
// the same header the inbound middleware keys on, which is deliberate: a caller
// that generated a key for its own request has already decided a repeat is
// harmless.
const idempotencyKeyHeader = "Idempotency-Key"

// retryJitterFraction is how much of each computed delay is randomized. Full
// jitter would make the first retry nearly immediate; none at all would make every
// replica in a fleet retry in lockstep, which is how a provider's ten-second blip
// becomes a synchronized thundering herd.
const retryJitterFraction = 0.5

// RetryPolicy bounds retries for one client.
//
// It is disabled by default, and that is deliberate for a template: retries
// multiply load on a dependency that is already struggling, and how many attempts
// a given provider can absorb is a decision per dependency rather than a default.
// What the template supplies is the policy that is hard to get right.
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

// retryTransport repeats a request that failed in a way a repeat could fix.
//
// Three rules are what make this safe, and they are the three a hand-rolled retry
// loop gets wrong.
//
// Only repeatable requests. A GET or HEAD may be repeated by definition; anything
// else is repeated only when the caller attached an Idempotency-Key, because a
// retried POST that the server did process creates the second resource this
// repository's inbound middleware exists to prevent.
//
// Never past the caller's deadline. The delay and one more attempt have to fit in
// what remains of the request context, or the retry is skipped and the last result
// is returned. A retry that outlives the request that asked for it holds a handler
// goroutine, and its in-flight slot, for work whose answer nobody is waiting for.
//
// Retry-After is honored when the server sent one and it is shorter than the
// deadline. A 429 or 503 with a hint is the server telling the client how long to
// wait; ignoring it is how a client that thinks it is being polite gets rate
// limited harder.
type retryTransport struct {
	base   http.RoundTripper
	policy RetryPolicy
}

// RoundTrip returns the last attempt's result unchanged.
//
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
		// The previous response body is abandoned only once another attempt is
		// certain, so a caller never receives a response whose body this closed.
		drainResponse(response)

		select {
		case <-request.Context().Done():
			return response, err
		case <-time.After(delay):
		}

		response, err = t.retryRequest(request)
	}

	return response, err
}

// retryRequest rebuilds the request for another attempt. repeatableRequest has
// already established that the body can be rewound.
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
// A GET, HEAD, or OPTIONS may be repeated by definition. Anything else is repeated
// only when the caller attached an Idempotency-Key: a retried POST that the server
// did process creates the second resource this repository's inbound middleware
// exists to prevent, and only the caller knows whether it made that safe.
//
// A body that has already been read also has to be rewindable, and only GetBody can
// do that. net/http populates it for the body types it knows; a caller that supplied
// an opaque io.Reader gets one attempt, which is the honest outcome rather than a
// silent retry of an empty body.
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
	return request.Header.Get(idempotencyKeyHeader) != ""
}

func hasBody(request *http.Request) bool {
	return request.Body != nil && request.Body != http.NoBody
}

// retryableResult reports whether another attempt could plausibly succeed.
//
// A transport error before response headers is retryable: the request may never
// have been processed. Among statuses, only the ones that mean "not now" are —
// 429 from a rate limiter, and the three gateway statuses that mean the request
// did not reach a working backend. A 500 is deliberately absent: it usually means
// the server did the work and failed partway, and repeating it repeats that.
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
	var tooLarge *ResponseTooLargeError
	return errors.As(err, &tooLarge)
}

// retryDelay computes the wait before attempt, preferring the server's own hint.
func retryDelay(base time.Duration, attempt int, response *http.Response) time.Duration {
	if hinted, ok := retryAfter(response); ok {
		return hinted
	}

	// Exponential from the base, then jittered. attempt is 2 for the first retry,
	// so the first delay is the base itself.
	backoff := base << (attempt - 2)
	if backoff <= 0 {
		// The shift overflowed, which only happens with an absurd attempt count.
		backoff = base
	}
	// #nosec G404 -- Jitter needs spread across replicas, not unpredictability; nothing here is a secret or a token.
	jitter := time.Duration(rand.Float64() * retryJitterFraction * float64(backoff))
	return backoff - time.Duration(retryJitterFraction*float64(backoff)) + jitter
}

// retryAfter reads the server's hint. Only the delay-seconds form is honored: the
// HTTP-date form requires trusting the client's clock against the server's, and a
// skewed clock turns a one-second hint into a minutes-long stall.
func retryAfter(response *http.Response) (time.Duration, bool) {
	if response == nil {
		return 0, false
	}
	raw := response.Header.Get("Retry-After")
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

// fitsRemainingBudget reports whether the delay leaves room for another attempt.
//
// The margin is the delay itself: an attempt that gets less time than it spent
// waiting is not worth making, and spending the caller's last milliseconds on a
// request that cannot finish is how a retry turns one failure into a slower one.
func fitsRemainingBudget(request *http.Request, delay time.Duration) bool {
	deadline, ok := request.Context().Deadline()
	if !ok {
		return true
	}
	return time.Until(deadline) > 2*delay
}

// maxDrainBytes bounds what is read from a response that is being abandoned. A
// body larger than this is not worth the read: the connection is one handshake,
// and the point of draining is to be cheaper than that.
const maxDrainBytes = 64 << 10

// drainResponse consumes and closes a response that is about to be replaced, so
// its connection returns to the pool instead of being torn down.
//
// The read is the part that matters, and closing alone does not do it. net/http
// hands back a body whose Close reports to the transport's read loop whether the
// body reached EOF; a close before EOF marks the connection unusable and it is
// discarded rather than pooled. So the version that only closed made every retry
// pay a fresh TCP and TLS handshake — against a dependency that had just
// answered 429 or 503, which is exactly when adding connection load is worst,
// and inside the remaining budget of a caller that is already one attempt down.
//
// The body is bounded twice over: responseLimitTransport has already capped it,
// and the limit below caps what is read of that.
func drainResponse(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxDrainBytes))
	_ = response.Body.Close()
}
