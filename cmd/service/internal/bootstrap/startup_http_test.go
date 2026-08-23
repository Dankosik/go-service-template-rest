package bootstrap

import (
	"io"
	stdlog "log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
)

func TestHTTPServerKeepsTerminalResponseReserve(t *testing.T) {
	t.Parallel()

	const requestTimeout = 100 * time.Millisecond
	handler := httpx.RequestTimeout(requestTimeout, http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	server := newHTTPServer(config.HTTPConfig{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      requestTimeout + time.Second,
		IdleTimeout:       time.Second,
		MaxHeaderBytes:    1024,
	}, handler, stdlog.New(io.Discard, "", 0))

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
	})

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+listener.Addr().String(), http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("GET timed response: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusGatewayTimeout)
	}
	if !strings.Contains(string(body), `"code":"gateway_timeout"`) {
		t.Fatalf("body = %q, want complete gateway_timeout Problem", body)
	}
}
