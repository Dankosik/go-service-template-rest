package oauthintrospection

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/authntrust"
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

const (
	testIssuer   = "https://issuer.example.com"
	testAudience = "https://api.example.com"
	testClientID = "rs-client"
	testSecret   = "rs-secret"
	testToken    = "opaque-access-token"
)

var testNow = time.Unix(1_900_000_000, 0).UTC()

type capturedRequest struct {
	method string
	rawURL string
	header http.Header
	body   []byte
}

type loopbackProvider struct {
	server       *httptest.Server
	calls        atomic.Int64
	peak         atomic.Int64
	inFlight     atomic.Int64
	mu           sync.Mutex
	requests     []capturedRequest
	handler      http.HandlerFunc
	received     chan struct{}
	receivedOnce sync.Once
}

func testPolicy(t *testing.T) Policy {
	t.Helper()
	return mustPolicy(t, PolicyInput{
		Issuer:       testIssuer,
		Audience:     testAudience,
		Endpoint:     "https://idp.example.com/oauth/introspect",
		TargetClass:  authntrust.TargetClassExternalHTTPS,
		ClientID:     testClientID,
		ClientSecret: testSecret,
	})
}

func mustPolicy(t *testing.T, input PolicyInput) Policy {
	t.Helper()
	policy, err := NewPolicy(input)
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	return policy
}

func activeJSON(sub, clientID string) string {
	body := `{"active":true,"iss":"` + testIssuer + `","aud":"` + testAudience +
		`","exp":` + strconv.FormatInt(testNow.Add(time.Hour).Unix(), 10)
	if sub != "" {
		body += `,"sub":"` + sub + `"`
	}
	if clientID != "" {
		body += `,"client_id":"` + clientID + `"`
	}
	return body + `}`
}

func newLoopbackProvider(t *testing.T, handler http.HandlerFunc) *loopbackProvider {
	t.Helper()
	provider := &loopbackProvider{received: make(chan struct{}), handler: handler}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		current := provider.inFlight.Add(1)
		defer provider.inFlight.Add(-1)
		for {
			previous := provider.peak.Load()
			if current <= previous || provider.peak.CompareAndSwap(previous, current) {
				break
			}
		}
		provider.calls.Add(1)
		body, _ := io.ReadAll(request.Body)
		_ = request.Body.Close()
		provider.mu.Lock()
		provider.requests = append(provider.requests, capturedRequest{
			method: request.Method,
			rawURL: request.URL.String(),
			header: request.Header.Clone(),
			body:   append([]byte(nil), body...),
		})
		provider.mu.Unlock()
		provider.receivedOnce.Do(func() { close(provider.received) })
		if provider.handler != nil {
			provider.handler(response, request)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, activeJSON("subject-1", "client-1"))
	}))
	server.StartTLS()
	t.Cleanup(server.Close)
	provider.server = server
	return provider
}

func (p *loopbackProvider) last() capturedRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.requests) == 0 {
		return capturedRequest{}
	}
	return p.requests[len(p.requests)-1]
}

func newPinnedVerifier(t *testing.T, provider *loopbackProvider) *Verifier {
	t.Helper()
	return newPinnedVerifierWithPolicy(t, provider, testPolicy(t))
}

func newPinnedVerifierWithPolicy(t *testing.T, provider *loopbackProvider, policy Policy) *Verifier {
	t.Helper()
	client, err := httpclient.NewExternalHTTPSWithLimits(policy.endpoint, httpclient.ResponseLimits{
		ResponseHeaderTimeout:  ProviderTimeout,
		MaxResponseHeaderBytes: MaxResponseHeaderBytes,
	})
	if err != nil {
		t.Fatalf("NewExternalHTTPSWithLimits() error = %v", err)
	}
	t.Cleanup(client.CloseIdleConnections)
	source, ok := provider.server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatal("httptest client transport has unexpected type")
	}
	httpclient.BindLoopbackTLS(client, source)
	if !httpclient.ProxyDisabled(client) {
		t.Fatal("loopback bind enabled a proxy")
	}
	return newVerifier(policy, client, func() time.Time { return testNow })
}

func requireKind(t *testing.T, err error, want bearerauthn.Kind) {
	t.Helper()
	got, ok := bearerauthn.KindOf(err)
	if !ok || got != want {
		t.Fatalf("error = %v, kind = %v, want %v", err, got, want)
	}
}

func mustParseForm(t *testing.T, body []byte) url.Values {
	t.Helper()
	values, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}
	return values
}
