package httpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ResponseTooLargeError reports a decoded response body exceeding its configured limit.
type ResponseTooLargeError struct {
	Limit int64
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("outbound HTTP response body exceeds %d bytes", e.Limit)
}

// authorityTransport is the innermost guard: it refuses any request whose scheme
// or authority drifted from the one the client was built for, before a dial can
// happen.
type authorityTransport struct {
	base      http.RoundTripper
	scheme    string
	authority string
}

func (t authorityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil ||
		req.URL.User != nil ||
		!strings.EqualFold(req.URL.Scheme, t.scheme) ||
		!strings.EqualFold(req.URL.Host, t.authority) {
		return nil, ErrTargetDenied
	}
	response, err := t.base.RoundTrip(req)
	if err != nil {
		return response, fmt.Errorf("send outbound HTTP request: %w", err)
	}
	return response, nil
}

type responseLimitTransport struct {
	base  http.RoundTripper
	limit int64
}

func (t responseLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(req)
	if response != nil && response.Body != nil {
		response.Body = &responseBody{
			ReadCloser: http.MaxBytesReader(nil, response.Body, t.limit),
			limit:      t.limit,
		}
	}
	if err != nil {
		return response, fmt.Errorf("receive outbound HTTP response: %w", err)
	}
	return response, nil
}

type responseBody struct {
	io.ReadCloser

	limit int64
}

func (b *responseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		_ = b.Close()
		return n, &ResponseTooLargeError{Limit: b.limit}
	}
	if err == nil {
		return n, nil
	}
	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}
	return n, fmt.Errorf("read outbound HTTP response body: %w", err)
}
