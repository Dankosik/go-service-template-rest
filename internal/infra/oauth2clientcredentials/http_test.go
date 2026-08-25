package oauth2clientcredentials

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"golang.org/x/oauth2"
)

type recordingResourceClient struct {
	authorization string
}

func (c *recordingResourceClient) RoundTrip(request *http.Request) (*http.Response, error) {
	c.authorization = request.Header.Get("Authorization")
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: request}, nil
}

func newTestHTTPClient(owner *Client, base http.RoundTripper) *HTTPClient {
	return &HTTPClient{client: newNoRedirectHTTPClient(&oauth2.Transport{
		Source: clientTokenSource{client: owner},
		Base:   base,
	})}
}

func TestHTTPClientAttachesLibraryTokenToRequestCopy(t *testing.T) {
	var calls atomic.Int32
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		calls.Add(1)
		return validTestToken("opaque"), nil
	}, nil)
	t.Cleanup(owner.Close)
	base := new(recordingResourceClient)
	client := newTestHTTPClient(owner, base)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://resource.example.com/items", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	_ = response.Body.Close()
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
	if base.authorization != "Bearer opaque" {
		t.Fatalf("Authorization = %q", base.authorization)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("Do() mutated the caller request")
	}

	request.Header.Set("Authorization", "Bearer caller")
	response, err = client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("Do() competing authorization error = %v", err)
	}
}

func TestHTTPClientRejectsUseAfterOwnerClose(t *testing.T) {
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		return validTestToken("opaque"), nil
	}, nil)
	client := newTestHTTPClient(owner, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: request}, nil
	}))
	owner.Close()
	request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://resource.example.com/items", http.NoBody)
	response, err := client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Do() error = %v, want ErrUnavailable", err)
	}
}

func TestHTTPClientDoesNotFollowRedirect(t *testing.T) {
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		return validTestToken("opaque"), nil
	}, nil)
	t.Cleanup(owner.Close)
	var calls int
	client := newTestHTTPClient(owner, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusTemporaryRedirect,
			Header:     http.Header{"Location": []string{"/redirected"}},
			Body:       http.NoBody,
			Request:    request,
		}, nil
	}))
	request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://resource.example.com/items", http.NoBody)

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Do() status = %d, want %d", response.StatusCode, http.StatusTemporaryRedirect)
	}
	if calls != 1 {
		t.Fatalf("resource calls = %d, want 1", calls)
	}
}

func TestHTTPClientConstructionRequiresBothOwners(t *testing.T) {
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		return validTestToken("opaque"), nil
	}, nil)
	t.Cleanup(owner.Close)
	base, err := httpclient.NewExternalHTTPS("https://resource.example.com")
	if err != nil {
		t.Fatalf("NewExternalHTTPS() error = %v", err)
	}
	defer base.CloseIdleConnections()
	client, err := owner.HTTP(base)
	if err != nil || client == nil {
		t.Fatalf("HTTP() = %#v, %v", client, err)
	}
	if _, err := owner.HTTP(nil); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("HTTP(nil) error = %v", err)
	}
}
