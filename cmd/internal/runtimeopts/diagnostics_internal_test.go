package runtimeopts

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDiagnosticsListenerStopForcesLocalTimeoutAndJoins(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	started := make(chan struct{})
	server := &http.Server{
		ReadHeaderTimeout: time.Second,
		Handler: http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			close(started)
			<-request.Context().Done()
		}),
	}
	served := &DiagnosticsListener{server: server, component: "test", done: make(chan struct{})}
	go func() {
		defer close(served.done)
		if serveErr := server.Serve(listener); !errors.Is(serveErr, http.ErrServerClosed) {
			served.serveErr = serveErr
		}
	}()

	requestDone := make(chan error, 1)
	go func() {
		request, requestErr := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+listener.Addr().String(), http.NoBody)
		if requestErr != nil {
			requestDone <- requestErr
			return
		}
		response, requestErr := http.DefaultClient.Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()
	<-started

	if err := served.Stop(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	select {
	case <-served.done:
	default:
		t.Fatal("Stop() returned before Serve joined")
	}
	if err := <-requestDone; err == nil {
		t.Fatal("forced connection close returned no client error")
	}
}
