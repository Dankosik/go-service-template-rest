package httpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// This file owns one thing: bounding the response body. The fixed-target
// guarantee itself — validateTarget, enforceDialAddress, and
// authorityTransport.RoundTrip — lives entirely in target_policy.go.

// ResponseTooLargeError reports a decoded response body exceeding its configured limit.
type ResponseTooLargeError struct {
	Limit int64
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("outbound HTTP response body exceeds %d bytes", e.Limit)
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
