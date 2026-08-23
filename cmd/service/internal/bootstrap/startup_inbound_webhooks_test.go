// profile:inbound-webhooks-standard:start
package bootstrap

import (
	"bufio"
	"bytes"
	"context"
	"io"
	stdlog "log"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestInboundWebhookServiceStartup(t *testing.T) {
	t.Parallel()

	receiver, err := initInboundWebhookReceiver(config.Config{}, nil, telemetry.New(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("empty inbound config err=%v", err)
	}
	if _, ok := receiver.(inboundwebhook.NoopReceiver); !ok {
		t.Fatalf("empty inbound config receiver = %T", receiver)
	}
	_, err = initInboundWebhookReceiver(config.Config{
		InboundWebhooks: config.InboundWebhooksConfig{Endpoints: `{"endpoints":[{"endpoint_id":"orders","active_key_reference":"active"}]}`},
	}, nil, telemetry.New(), slog.New(slog.DiscardHandler))
	if err == nil || !strings.Contains(err.Error(), "postgres") {
		t.Fatalf("missing postgres error = %v", err)
	}
}

func TestInboundWebhookHeaderOverflowUsesListener431(t *testing.T) {
	t.Parallel()

	handler, err := newHTTPHandler(
		config.Config{HTTP: config.HTTPConfig{MaxBodyBytes: 1024, RequestTimeout: time.Second, MaxInFlight: 1}},
		slog.New(slog.DiscardHandler),
		telemetry.New(),
		nil,
		httpRuntimeBindings{
			Handlers: httpx.Handlers{
				Health:         health.New(),
				ReadinessGate:  func(context.Context) error { return nil },
				API:            newServiceAPI(),
				InboundWebhook: nil,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	srv := newHTTPServer(config.HTTPConfig{
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		MaxHeaderBytes:    256,
	}, handler, stdlog.New(io.Discard, "", 0))
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(context.Background(), "tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	request := "POST /webhooks/orders HTTP/1.1\r\nHost: localhost\r\nX-Overflow: " + strings.Repeat("a", 8192) + "\r\n\r\n"
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestInboundWebhookRequestBufferBudget(t *testing.T) {
	t.Parallel()

	if got := requestBufferCopyCount(); got != 2 {
		t.Fatalf("copy count = %d, want 2", got)
	}
	cfg := config.Config{
		HTTP:    config.HTTPConfig{MaxInFlight: 1, MaxBodyBytes: 200},
		Runtime: config.RuntimeConfig{MemoryLimitRatio: 0.9},
	}
	var logged bytes.Buffer
	reportRequestBufferBudget(slog.New(slog.NewJSONHandler(&logged, nil)), cfg, 1000)
	if !strings.Contains(logged.String(), `"request_buffers.worst_case_bytes":400`) {
		t.Fatalf("log = %s", logged.String())
	}
}

// profile:inbound-webhooks-standard:end
