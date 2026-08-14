package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

type httpDoer interface {
	Do(request *http.Request) (*http.Response, error)
}

type transport struct {
	base         httpDoer
	endpoint     url.URL
	controlLimit int64
	objectLimit  int64
}

type sendState struct {
	wroteHeaders bool
}

type sendStateKey struct{}

type responseState struct{ body io.Closer }

type responseStateKey struct{}

func withSendState(ctx context.Context, state *sendState) context.Context {
	return context.WithValue(ctx, sendStateKey{}, state)
}

func withResponseState(ctx context.Context, state *responseState) context.Context {
	return context.WithValue(ctx, responseStateKey{}, state)
}

func (s *responseState) close() {
	if s.body != nil {
		_ = s.body.Close()
	}
}

func (t *transport) Do(request *http.Request) (*http.Response, error) {
	response, err := t.do(request)
	if response != nil && response.Body != nil {
		limit := t.controlLimit
		if isGetObjectRequest(request) && response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			limit = t.objectLimit
		}
		response.Body = &limitedBody{ReadCloser: response.Body, limit: limit}
	}
	return response, err
}

func isGetObjectRequest(request *http.Request) bool {
	if request.Method != http.MethodGet {
		return false
	}
	query := request.URL.Query()
	return request.URL.RawQuery == "" || (query.Get("x-id") == "GetObject" && len(query) == 1)
}

func (t *transport) do(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil || request.URL.User != nil || request.URL.Scheme != t.endpoint.Scheme || request.URL.Host != t.endpoint.Host {
		return nil, httpclient.ErrTargetDenied
	}
	wroteHeaders := false
	request = request.Clone(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
		WroteHeaders: func() { wroteHeaders = true },
	}))
	response, err := t.base.Do(request)
	if state, ok := request.Context().Value(responseStateKey{}).(*responseState); ok && response != nil {
		state.body = response.Body
	}
	if state, ok := request.Context().Value(sendStateKey{}).(*sendState); ok {
		state.wroteHeaders = wroteHeaders
	}
	if err != nil {
		return response, fmt.Errorf("send S3 request: %w", err)
	}
	return response, nil
}

type limitedBody struct {
	io.ReadCloser

	limit     int64
	closeOnce sync.Once
	closeErr  error
}

func (b *limitedBody) Close() error {
	b.closeOnce.Do(func() { b.closeErr = b.ReadCloser.Close() })
	return b.closeErr
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.limit == 0 {
		var probe [1]byte
		n, err := b.ReadCloser.Read(probe[:])
		if n > 0 {
			_ = b.Close()
			return 0, &bodyTooLargeError{}
		}
		return 0, err //nolint:wrapcheck // Reader callers require the source EOF identity.
	}
	if int64(len(p)) > b.limit {
		p = p[:b.limit]
	}
	n, err := b.ReadCloser.Read(p)
	b.limit -= int64(n)
	if errors.Is(err, io.EOF) {
		_ = b.Close()
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return n, fmt.Errorf("read S3 response: %w", err)
	}
	return n, err //nolint:wrapcheck // Reader callers require the source EOF identity.
}
