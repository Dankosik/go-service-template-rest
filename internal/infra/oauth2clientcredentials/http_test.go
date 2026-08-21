package oauth2clientcredentials

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"golang.org/x/oauth2"
)

type recordingResourceClient struct {
	authorization string
}

func (c *recordingResourceClient) Do(request *http.Request) (*http.Response, error) {
	c.authorization = request.Header.Get("Authorization")
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: request}, nil
}

func TestHTTPClientAttachesLibraryTokenToRequestCopy(t *testing.T) {
	var calls atomic.Int32
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		calls.Add(1)
		return validTestToken("opaque"), nil
	}, nil)
	base := &recordingResourceClient{}
	client := &HTTPClient{client: owner, base: base}
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

func TestHTTPClientCallerCancellationStopsOnlyItsWait(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	owner := newClient(func(context.Context) (*oauth2.Token, error) {
		close(entered)
		<-release
		return validTestToken("opaque"), nil
	}, nil)
	client := &HTTPClient{client: owner, base: &recordingResourceClient{}}
	ctx, cancel := context.WithCancel(context.Background())
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://resource.example.com/items", http.NoBody)
	done := make(chan error, 1)
	go func() {
		response, err := client.Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		done <- err
	}()
	<-entered
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
	close(release)
	if err := owner.Close(t.Context()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
