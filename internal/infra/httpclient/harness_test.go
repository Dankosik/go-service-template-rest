package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

func validExternalConfig() Config {
	return Config{
		DependencyName:         "provider",
		BaseURL:                "https://example.com",
		TargetClass:            ExternalHTTPS,
		RequestTimeout:         time.Second,
		ResponseHeaderTimeout:  500 * time.Millisecond,
		MaxResponseHeaderBytes: 32 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        8,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackedBody struct {
	io.ReadCloser

	closed *atomic.Bool
}

func (b *trackedBody) Close() error {
	b.closed.Store(true)
	if err := b.ReadCloser.Close(); err != nil {
		return fmt.Errorf("close tracked response body: %w", err)
	}
	return nil
}

func responseWithBody(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
