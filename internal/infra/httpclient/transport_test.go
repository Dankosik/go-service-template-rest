package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestResponseLimitTransportBoundsDecodedBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      func(t *testing.T) io.ReadCloser
		limit     int64
		want      string
		wantLarge bool
	}{
		{
			name:  "exact limit",
			body:  func(*testing.T) io.ReadCloser { return io.NopCloser(strings.NewReader("hello")) },
			limit: 5,
			want:  "hello",
		},
		{
			name:      "plain overflow",
			body:      func(*testing.T) io.ReadCloser { return io.NopCloser(strings.NewReader("hello")) },
			limit:     4,
			want:      "hell",
			wantLarge: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var closed atomic.Bool
			transport := responseLimitTransport{
				base: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body: &trackedBody{
							ReadCloser: test.body(t),
							closed:     &closed,
						},
					}, nil
				}),
				limit: test.limit,
			}

			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			response, err := transport.RoundTrip(request)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			if got := string(body); got != test.want {
				t.Fatalf("body = %q, want %q", got, test.want)
			}
			tooLarge, got := errors.AsType[*ResponseTooLargeError](readErr)
			if got != test.wantLarge {
				t.Fatalf("ResponseTooLargeError present = %t, want %t; error = %v", got, test.wantLarge, readErr)
			}
			if tooLarge != nil && tooLarge.Limit != test.limit {
				t.Fatalf("ResponseTooLargeError.Limit = %d, want %d", tooLarge.Limit, test.limit)
			}
			if test.wantLarge && !closed.Load() {
				t.Fatal("overflow did not close the underlying response body")
			}
			if err := response.Body.Close(); err != nil {
				t.Fatalf("Body.Close() error = %v", err)
			}
			if !closed.Load() {
				t.Fatal("underlying response body was not closed")
			}
		})
	}
}
