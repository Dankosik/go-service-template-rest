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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
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
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain listener response", got)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if got := string(body); got != "431 Request Header Fields Too Large" {
		t.Fatalf("body = %q, want listener 431 text", got)
	}

	spec, err := openapi.GetSpec()
	if err != nil {
		t.Fatalf("load OpenAPI contract: %v", err)
	}
	spec.Servers = nil
	contractRouter, err := gorillamux.NewRouter(spec)
	if err != nil {
		t.Fatalf("build OpenAPI contract router: %v", err)
	}
	contractRequest := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/orders", nil)
	route, pathParams, err := contractRouter.FindRoute(contractRequest)
	if err != nil {
		t.Fatalf("find webhook contract route: %v", err)
	}
	validation := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    contractRequest,
			PathParams: pathParams,
			Route:      route,
		},
		Status:  resp.StatusCode,
		Header:  resp.Header,
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	validation.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(t.Context(), validation); err != nil {
		t.Fatalf("listener response does not match OpenAPI contract: %v", err)
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
