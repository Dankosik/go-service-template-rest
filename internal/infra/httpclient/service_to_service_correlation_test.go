package httpclient_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/dns/dnsmessage"
)

//nolint:paralleltest // Replaces net.DefaultResolver and the global OTel provider with cleanup restoration.
func TestServiceToServiceHTTPCorrelationAndCancellation(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)
	privateAddress := privateTestAddress(t)
	const downstreamHost = "service-b.correlation.internal"
	installTestDNSResolver(t, downstreamHost, privateAddress)

	var serviceALogs, serviceBLogs synchronizedBuffer
	serviceALog := slog.New(logctx.New(slog.NewJSONHandler(&serviceALogs, nil)))
	serviceBLog := slog.New(logctx.New(slog.NewJSONHandler(&serviceBLogs, nil)))
	entered := make(chan struct{})
	serviceBDone := make(chan struct{})
	serviceBReturned := make(chan struct{})

	serviceBRoutes := http.NewServeMux()
	serviceBRoutes.HandleFunc("GET /work", func(w http.ResponseWriter, r *http.Request) {
		serviceBLog.InfoContext(r.Context(), "service-b-handler")
		w.WriteHeader(http.StatusNoContent)
	})
	serviceBRoutes.HandleFunc("POST /cancel", func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		serviceBLog.InfoContext(r.Context(), "service-b-canceled")
		close(serviceBDone)
	})
	serviceBRouter := hardenedTestRouter(t, serviceBLog, "service-b", serviceBRoutes)
	serviceBAddress := startHTTPTestServer(t, privateAddress, http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		serviceBRouter.ServeHTTP(w, r)
		if r.Method == http.MethodPost && r.URL.Path == "/cancel" {
			close(serviceBReturned)
		}
	}))
	_, serviceBPort, err := net.SplitHostPort(serviceBAddress)
	if err != nil {
		t.Fatalf("net.SplitHostPort(service B) error = %v", err)
	}

	downstream, err := httpclient.New(httpclient.Config{
		DependencyName:         "service-b",
		BaseURL:                "http://" + net.JoinHostPort(downstreamHost, serviceBPort),
		TargetClass:            httpclient.PrivateHTTP,
		PrivateHostSuffix:      ".correlation.internal",
		RequestTimeout:         5 * time.Second,
		ResponseHeaderTimeout:  2 * time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        4,
		MaxIdleConnsPerHost:    4,
		Propagation:            httpclient.PropagationTrustedService,
	}, nil)
	if err != nil {
		t.Fatalf("httpclient.New() error = %v", err)
	}
	t.Cleanup(downstream.CloseIdleConnections)

	downstreamErrors := make(chan error, 1)
	serviceACancelDone := make(chan struct{})
	serviceAReturned := make(chan struct{})
	serviceARoutes := http.NewServeMux()
	serviceARoutes.HandleFunc("GET /call", func(w http.ResponseWriter, r *http.Request) {
		serviceALog.InfoContext(r.Context(), "service-a-handler")
		response, callErr := callDownstream(r.Context(), downstream, http.MethodGet, "/work")
		if callErr != nil {
			downstreamErrors <- callErr
			http.Error(w, "downstream unavailable", http.StatusBadGateway)
			return
		}
		if _, copyErr := io.Copy(io.Discard, response.Body); copyErr != nil {
			_ = response.Body.Close()
			http.Error(w, "read downstream response", http.StatusBadGateway)
			return
		}
		if closeErr := response.Body.Close(); closeErr != nil {
			http.Error(w, "close downstream response", http.StatusBadGateway)
			return
		}
		w.WriteHeader(response.StatusCode)
	})
	serviceARoutes.HandleFunc("POST /cancel", func(w http.ResponseWriter, r *http.Request) {
		defer close(serviceACancelDone)
		response, callErr := callDownstream(r.Context(), downstream, http.MethodPost, "/cancel")
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		downstreamErrors <- callErr
		if callErr != nil {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	serviceARouter := hardenedTestRouter(t, serviceALog, "service-a", serviceARoutes)
	serviceAAddress := startHTTPTestServer(t, privateAddress, http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		serviceARouter.ServeHTTP(w, r)
		if r.Method == http.MethodPost && r.URL.Path == "/cancel" {
			close(serviceAReturned)
		}
	}))

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatalf("http.DefaultTransport type = %T, want *http.Transport", http.DefaultTransport)
	}
	baseTransport := defaultTransport.Clone()
	baseTransport.Proxy = nil
	caller := &http.Client{
		Transport: otelhttp.NewTransport(
			baseTransport,
			otelhttp.WithPropagators(propagation.TraceContext{}),
		),
	}
	t.Cleanup(baseTransport.CloseIdleConnections)

	const requestID = "service_chain_request_123"
	spanStart := len(recorder.Ended())
	rootCtx, root := otel.Tracer("service-to-service-http-test").Start(t.Context(), "root")
	rootSpanContext := root.SpanContext()
	request, err := http.NewRequestWithContext(rootCtx, http.MethodGet, "http://"+serviceAAddress+"/call", http.NoBody)
	if err != nil {
		root.End()
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	request.Header.Set("X-Request-ID", requestID)
	response, err := caller.Do(request)
	if err != nil {
		root.End()
		t.Fatalf("service A call error = %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		_ = response.Body.Close()
		root.End()
		t.Fatalf("service A status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	if err := response.Body.Close(); err != nil {
		root.End()
		t.Fatalf("response.Body.Close() error = %v", err)
	}
	root.End()

	assertHTTPServiceSpanChain(t, recorder.Ended()[spanStart:], rootSpanContext)
	assertCorrelationLog(t, serviceALogs.String(), requestID, rootSpanContext.TraceID().String())
	assertCorrelationLog(t, serviceBLogs.String(), requestID, rootSpanContext.TraceID().String())

	cancelCtx, cancel := context.WithCancel(t.Context())
	cancelCtx, cancelRoot := otel.Tracer("service-to-service-http-test").Start(cancelCtx, "cancel-root")
	cancelRequest, err := http.NewRequestWithContext(cancelCtx, http.MethodPost, "http://"+serviceAAddress+"/cancel", http.NoBody)
	if err != nil {
		cancel()
		cancelRoot.End()
		t.Fatalf("http.NewRequestWithContext(cancel) error = %v", err)
	}
	cancelRequest.Header.Set("X-Request-ID", requestID)
	callResult := make(chan error, 1)
	go func() {
		cancelResponse, callErr := caller.Do(cancelRequest)
		if cancelResponse != nil && cancelResponse.Body != nil {
			_ = cancelResponse.Body.Close()
		}
		callResult <- callErr
	}()

	<-entered
	cancel()
	if err := <-callResult; !errors.Is(err, context.Canceled) {
		cancelRoot.End()
		t.Fatalf("caller error = %v, want context.Canceled", err)
	}
	if err := <-downstreamErrors; !errors.Is(err, context.Canceled) {
		cancelRoot.End()
		t.Fatalf("downstream error = %v, want context.Canceled", err)
	}
	<-serviceBDone
	<-serviceACancelDone
	<-serviceBReturned
	<-serviceAReturned
	cancelRoot.End()

	canceledSpans := spansForTrace(recorder.Ended(), cancelRoot.SpanContext().TraceID())
	if len(canceledSpans) < 5 {
		t.Fatalf("ended canceled trace spans = %d, want root plus four transport spans", len(canceledSpans))
	}
}

func hardenedTestRouter(
	t *testing.T,
	log *slog.Logger,
	serverName string,
	routes http.Handler,
) http.Handler {
	t.Helper()

	router, err := httpx.Harden(log, telemetry.New(), httpx.RouterConfig{
		MaxBodyBytes:   1 << 20,
		RequestTimeout: 5 * time.Second,
		MaxInFlight:    4,
		OTelServerName: serverName,
	}, routes)
	if err != nil {
		t.Fatalf("httpx.Harden(%s) error = %v", serverName, err)
	}
	return router
}

func callDownstream(
	ctx context.Context,
	client *httpclient.Client,
	method string,
	path string,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, client.BaseURL()+path, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build downstream request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return response, fmt.Errorf("call downstream: %w", err)
	}
	return response, nil
}

func startHTTPTestServer(t *testing.T, address netip.Addr, handler http.Handler) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(
		t.Context(),
		"tcp",
		net.JoinHostPort(address.String(), "0"),
	)
	if err != nil {
		t.Fatalf("net.Listen(%s) error = %v", address, err)
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second,
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			t.Errorf("Server.Shutdown() error = %v", err)
		}
		if err := <-serveDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server.Serve() error = %v", err)
		}
	})
	return listener.Addr().String()
}

func privateTestAddress(t *testing.T) netip.Addr {
	t.Helper()

	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces() error = %v", err)
	}
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, addressErr := networkInterface.Addrs()
		if addressErr != nil {
			continue
		}
		for _, candidate := range addresses {
			prefix, parseErr := netip.ParsePrefix(candidate.String())
			if parseErr == nil && prefix.Addr().Is4() && prefix.Addr().IsPrivate() {
				return prefix.Addr()
			}
		}
	}
	t.Fatal("no bindable private IPv4 address available for production HTTP-client proof")
	return netip.Addr{}
}

func installTestDNSResolver(t *testing.T, hostname string, address netip.Addr) {
	t.Helper()

	packetConn, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start test DNS server: %v", err)
	}
	dnsDone := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1232)
		for {
			size, peer, readErr := packetConn.ReadFrom(buffer)
			if readErr != nil {
				dnsDone <- readErr
				return
			}
			response, responseErr := privateDNSResponse(buffer[:size], hostname, address)
			if responseErr != nil {
				dnsDone <- responseErr
				return
			}
			if _, writeErr := packetConn.WriteTo(response, peer); writeErr != nil {
				dnsDone <- writeErr
				return
			}
		}
	}()

	previousResolver := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, "udp", packetConn.LocalAddr().String())
		},
	}
	t.Cleanup(func() {
		net.DefaultResolver = previousResolver
		if err := packetConn.Close(); err != nil {
			t.Errorf("close test DNS server: %v", err)
		}
		if err := <-dnsDone; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("test DNS server error = %v", err)
		}
	})
}

func privateDNSResponse(query []byte, hostname string, address netip.Addr) ([]byte, error) {
	var parser dnsmessage.Parser
	header, err := parser.Start(query)
	if err != nil {
		return nil, fmt.Errorf("parse DNS header: %w", err)
	}
	question, err := parser.Question()
	if err != nil {
		return nil, fmt.Errorf("parse DNS question: %w", err)
	}

	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:                 header.ID,
		Response:           true,
		RecursionDesired:   header.RecursionDesired,
		RecursionAvailable: true,
	})
	builder.EnableCompression()
	if err := builder.StartQuestions(); err != nil {
		return nil, fmt.Errorf("start DNS questions: %w", err)
	}
	if err := builder.Question(question); err != nil {
		return nil, fmt.Errorf("write DNS question: %w", err)
	}
	if err := builder.StartAnswers(); err != nil {
		return nil, fmt.Errorf("start DNS answers: %w", err)
	}
	if question.Type == dnsmessage.TypeA &&
		strings.EqualFold(strings.TrimSuffix(question.Name.String(), "."), hostname) {
		if err := builder.AResource(dnsmessage.ResourceHeader{
			Name:  question.Name,
			Type:  dnsmessage.TypeA,
			Class: dnsmessage.ClassINET,
			TTL:   1,
		}, dnsmessage.AResource{A: address.As4()}); err != nil {
			return nil, fmt.Errorf("write DNS A answer: %w", err)
		}
	}
	response, err := builder.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish DNS response: %w", err)
	}
	return response, nil
}

func assertHTTPServiceSpanChain(
	t *testing.T,
	spans []sdktrace.ReadOnlySpan,
	root trace.SpanContext,
) {
	t.Helper()

	topClient := childSpan(t, spans, root.SpanID(), trace.SpanKindClient)
	serviceAServer := childSpan(t, spans, topClient.SpanContext().SpanID(), trace.SpanKindServer)
	downstreamClient := childSpan(t, spans, serviceAServer.SpanContext().SpanID(), trace.SpanKindClient)
	serviceBServer := childSpan(t, spans, downstreamClient.SpanContext().SpanID(), trace.SpanKindServer)
	for _, span := range []sdktrace.ReadOnlySpan{topClient, serviceAServer, downstreamClient, serviceBServer} {
		if span.SpanContext().TraceID() != root.TraceID() {
			t.Fatalf("span %q trace ID = %s, want %s", span.Name(), span.SpanContext().TraceID(), root.TraceID())
		}
	}
}

func childSpan( //nolint:ireturn // The OpenTelemetry recorder exposes spans through this interface.
	t *testing.T,
	spans []sdktrace.ReadOnlySpan,
	parent trace.SpanID,
	kind trace.SpanKind,
) sdktrace.ReadOnlySpan {
	t.Helper()

	for _, span := range spans {
		if span.SpanKind() == kind && span.Parent().SpanID() == parent {
			return span
		}
	}
	t.Fatalf("no %s span with parent %s in %s", kind, parent, spanSummary(spans))
	return nil
}

func spansForTrace(spans []sdktrace.ReadOnlySpan, traceID trace.TraceID) []sdktrace.ReadOnlySpan {
	var matched []sdktrace.ReadOnlySpan
	for _, span := range spans {
		if span.SpanContext().TraceID() == traceID {
			matched = append(matched, span)
		}
	}
	return matched
}

func spanSummary(spans []sdktrace.ReadOnlySpan) string {
	var summary []string
	for _, span := range spans {
		summary = append(summary, span.Name()+"/"+span.SpanKind().String()+"/"+span.Parent().SpanID().String())
	}
	return strings.Join(summary, ",")
}

func assertCorrelationLog(t *testing.T, logs string, requestID string, traceID string) {
	t.Helper()

	for key, value := range map[string]string{
		"request_id": requestID,
		"trace_id":   traceID,
	} {
		encoded := strconv.Quote(key) + ":" + strconv.Quote(value)
		if !strings.Contains(logs, encoded) {
			t.Fatalf("logs do not contain %s: %s", encoded, logs)
		}
	}
}

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronizedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.buffer.Write(value)
	if err != nil {
		return n, fmt.Errorf("write synchronized log buffer: %w", err)
	}
	return n, nil
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
