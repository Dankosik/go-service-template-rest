package oauth2clientcredentials

// profile:outbound-auth-http:start

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
	"go.opentelemetry.io/otel/metric"
)

const (
	testTokenHost     = "token.auth.internal"
	testResourceHost  = "resource.auth.internal"
	otherResourceHost = "other.auth.internal"
	testBearerToken   = "operation-fixed-token"
)

type httpFixture struct {
	config          Config
	client          *Client
	resource        *httpclient.Client
	authenticated   *HTTPClient
	clock           *movableClock
	providerCalls   *atomic.Int32
	providerHeaders chan http.Header
	resourceCalls   *atomic.Int32
	authorizations  chan string
}

func newHTTPFixture(
	t *testing.T,
	clock *movableClock,
	resourceHandler http.Handler,
	steps []acquisitionStep,
	retry httpclient.RetryPolicy,
) *httpFixture {
	t.Helper()
	return newHTTPFixtureWithTelemetry(t, clock, resourceHandler, steps, retry, nil, nil)
}

func newHTTPFixtureWithMeter(
	t *testing.T,
	clock *movableClock,
	resourceHandler http.Handler,
	steps []acquisitionStep,
	retry httpclient.RetryPolicy,
	meterProvider metric.MeterProvider,
) *httpFixture {
	t.Helper()
	return newHTTPFixtureWithTelemetry(t, clock, resourceHandler, steps, retry, meterProvider, nil)
}

func newHTTPFixtureWithTelemetry(
	t *testing.T,
	clock *movableClock,
	resourceHandler http.Handler,
	steps []acquisitionStep,
	retry httpclient.RetryPolicy,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
) *httpFixture {
	t.Helper()
	address := privateTestAddress(t)
	installPrivateTestResolver(t, map[string]netip.Addr{
		testTokenHost:     address,
		testResourceHost:  address,
		otherResourceHost: address,
	})
	providerCalls := new(atomic.Int32)
	providerHeaders := make(chan http.Header, 8)
	tokenURL := startPrivateHTTPSTestServer(t, address, testTokenHost, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		index := int(providerCalls.Add(1)) - 1
		providerHeaders <- request.Header.Clone()
		if index >= len(steps) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		step := steps[index]
		if username, password, ok := request.BasicAuth(); !ok ||
			username != url.QueryEscape(" client:id+ ") || password != url.QueryEscape(" secret:p@ss+ ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if step.entered != nil {
			close(step.entered)
		}
		if step.release != nil {
			select {
			case <-step.release:
			case <-request.Context().Done():
				return
			}
		}
		if step.err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":%d}`,
			step.token.value,
			int64(step.token.expiresAt.Sub(clock.Now())/time.Second),
		)
	})) + "/oauth/token"
	resourceCalls := new(atomic.Int32)
	authorizations := make(chan string, 8)
	resourceURL := startPrivateHTTPSTestServer(t, address, testResourceHost, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		resourceCalls.Add(1)
		authorizations <- request.Header.Get("Authorization")
		resourceHandler.ServeHTTP(w, request)
	}))
	config := validTestConfig()
	config.TokenEndpoint = tokenURL
	config.TokenTargetClass = httpclient.PrivateHTTPS
	config.TokenPrivateHostSuffix = "auth.internal"
	config.ResourceAuthority = resourceURL
	tokenClient, err := newTokenHTTPClient(config, nil)
	if err != nil {
		t.Fatalf("newTokenHTTPClient() error = %v", err)
	}
	ownerCreated := false
	t.Cleanup(func() {
		if !ownerCreated {
			tokenClient.CloseIdleConnections()
		}
	})
	provider, err := newProvider(config, tokenClient, clock.Now)
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}
	client, err := newClient(config, meterProvider, log, clock.Now, provider.acquire, tokenClient.CloseIdleConnections)
	if err != nil {
		t.Fatalf("newClient() error = %v", err)
	}
	ownerCreated = true
	var resource *httpclient.Client
	t.Cleanup(func() {
		if err := client.Close(context.Background()); err != nil {
			t.Errorf("Client.Close() cleanup error = %v", err)
		}
		if resource != nil {
			resource.CloseIdleConnections()
		}
	})
	resource, err = httpclient.New(httpclient.Config{
		DependencyName:         config.DependencyName,
		BaseURL:                resourceURL,
		TargetClass:            httpclient.PrivateHTTPS,
		PrivateHostSuffix:      "auth.internal",
		RequestTimeout:         2 * time.Second,
		ResponseHeaderTimeout:  time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        2,
		Retry:                  retry,
	}, nil)
	if err != nil {
		t.Fatalf("httpclient.New(resource) error = %v", err)
	}
	authenticated, err := NewHTTPClient(client, resource)
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}
	return &httpFixture{
		config:          config,
		client:          client,
		resource:        resource,
		authenticated:   authenticated,
		clock:           clock,
		providerCalls:   providerCalls,
		providerHeaders: providerHeaders,
		resourceCalls:   resourceCalls,
		authorizations:  authorizations,
	}
}

func reusableAccessToken(clock *movableClock) accessToken {
	return accessToken{value: testBearerToken, expiresAt: clock.Now().Add(time.Minute)}
}

func (f *httpFixture) request(ctx context.Context, t *testing.T) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, f.resource.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatalf("build resource request: %v", err)
	}
	return request
}

func closeResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, response.Body)
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}

func TestHTTPClientResourceAuthorityIsFixed(t *testing.T) {
	t.Parallel()
	t.Run("constructor mismatch", testHTTPClientResourceAuthorityMismatch)
	t.Run("alternate authority and redirect", testHTTPClientAlternateAuthorityAndRedirectDoNotForwardBearer)
}

func testHTTPClientResourceAuthorityMismatch(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	client := requireTestClient(t, validTestConfig(), testClientOptions{
		now:     clock.Now,
		acquire: (&scriptedAcquirer{steps: []acquisitionStep{{token: reusableAccessToken(clock)}}}).acquire,
	})
	base, err := httpclient.New(httpclient.Config{
		DependencyName:         "other",
		BaseURL:                "https://example.com",
		TargetClass:            httpclient.ExternalHTTPS,
		RequestTimeout:         time.Second,
		ResponseHeaderTimeout:  time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		MaxResponseBodyBytes:   1 << 20,
		MaxConnsPerHost:        1,
	}, nil)
	if err != nil {
		t.Fatalf("httpclient.New() error = %v", err)
	}
	t.Cleanup(base.CloseIdleConnections)
	_, err = NewHTTPClient(client, base)
	assertFailureClass(t, err, FailureInvalidConfiguration)
}

func TestHTTPClientAttachesOneOperationToken(t *testing.T) {
	t.Parallel()
	t.Run("attaches fixed token", testHTTPClientAttachesOneOperationToken)
	t.Run("token failure reaches no resource", testHTTPClientTokenFailureReachesNoResource)
}

func testHTTPClientAttachesOneOperationToken(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []acquisitionStep{{token: reusableAccessToken(clock)}}, httpclient.RetryPolicy{})
	request := fixture.request(t.Context(), t)
	response, err := fixture.authenticated.Do(request)
	if err != nil {
		t.Fatalf("HTTPClient.Do() error = %v", err)
	}
	closeResponse(t, response)
	if got := <-fixture.authorizations; got != "Bearer "+testBearerToken {
		t.Fatalf("Authorization = %q, want one fixed bearer", got)
	}
	if got := request.Header.Get("Authorization"); got != "" {
		t.Fatalf("caller Authorization = %q, want unchanged", got)
	}
	if got := fixture.providerCalls.Load(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
}

func TestHTTPClientRejectsCallerAuthorization(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []acquisitionStep{{token: reusableAccessToken(clock)}}, httpclient.RetryPolicy{})
	for _, name := range []string{"Authorization", "authorization", "AUTHORIZATION"} {
		request := fixture.request(t.Context(), t)
		request.Header[name] = []string{"Bearer caller-token"}
		response, err := fixture.authenticated.Do(request)
		closeResponse(t, response)
		assertFailureClass(t, err, FailureInvalidConfiguration)
	}
	if got := fixture.providerCalls.Load(); got != 0 {
		t.Fatalf("provider calls = %d, want 0", got)
	}
	if got := fixture.resourceCalls.Load(); got != 0 {
		t.Fatalf("resource calls = %d, want 0", got)
	}
}

func TestHTTPClientCallerCancellationStopsOnlyItsWait(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	entered := make(chan struct{})
	release := make(chan struct{})
	fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []acquisitionStep{{token: reusableAccessToken(clock), entered: entered, release: release}}, httpclient.RetryPolicy{})
	canceledCtx, cancel := context.WithCancel(t.Context())
	canceledRequest := fixture.request(canceledCtx, t)
	liveRequest := fixture.request(t.Context(), t)
	run := func(request *http.Request, result chan<- error) {
		response, err := fixture.authenticated.Do(request)
		if response != nil && response.Body != nil {
			_, _ = io.Copy(io.Discard, response.Body)
			if closeErr := response.Body.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("close response body: %w", closeErr)
			}
		}
		result <- err
	}
	canceledResult := make(chan error, 1)
	go run(canceledRequest, canceledResult)
	<-entered
	liveResult := make(chan error, 1)
	go run(liveRequest, liveResult)
	cancel()
	assertFailureClass(t, <-canceledResult, FailureCallerCanceled)
	close(release)
	if err := <-liveResult; err != nil {
		t.Fatalf("live HTTPClient.Do() error = %v", err)
	}
	if got := fixture.providerCalls.Load(); got != 1 {
		t.Fatalf("provider calls = %d, want shared 1", got)
	}
	if got := fixture.resourceCalls.Load(); got != 1 {
		t.Fatalf("resource calls = %d, want live caller only", got)
	}
}

func TestHTTPClientPreservesDownstreamAuthResponses(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			clock := newMovableClock(fixedProviderTime)
			fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}), []acquisitionStep{{token: reusableAccessToken(clock)}}, httpclient.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
			response, err := fixture.authenticated.Do(fixture.request(t.Context(), t))
			if err != nil {
				t.Fatalf("HTTPClient.Do() error = %v", err)
			}
			closeResponse(t, response)
			if response.StatusCode != status {
				t.Fatalf("status = %d, want %d", response.StatusCode, status)
			}
			if got := fixture.resourceCalls.Load(); got != 1 {
				t.Fatalf("resource calls = %d, want no replay", got)
			}
			if got := fixture.providerCalls.Load(); got != 1 {
				t.Fatalf("provider calls = %d, want 1", got)
			}
		})
	}
}

func TestHTTPRetryFixesOneTokenAndStopsAtMargin(t *testing.T) {
	t.Parallel()
	t.Run("permitted retry reuses the operation token", func(t *testing.T) {
		t.Parallel()
		clock := newMovableClock(fixedProviderTime)
		var attempts atomic.Int32
		fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}), []acquisitionStep{{token: reusableAccessToken(clock)}}, httpclient.RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond})
		response, err := fixture.authenticated.Do(fixture.request(t.Context(), t))
		if err != nil {
			t.Fatalf("HTTPClient.Do() error = %v", err)
		}
		closeResponse(t, response)
		first, second := <-fixture.authorizations, <-fixture.authorizations
		if first != "Bearer "+testBearerToken || second != first {
			t.Fatalf("attempt bearers = %q, %q, want same operation token", first, second)
		}
		if got := fixture.providerCalls.Load(); got != 1 {
			t.Fatalf("provider calls = %d, want 1", got)
		}
	})

	t.Run("margin stops before attempt two", func(t *testing.T) {
		t.Parallel()
		clock := newMovableClock(fixedProviderTime)
		var advanceOnce sync.Once
		fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			advanceOnce.Do(func() { clock.Advance(50 * time.Second) })
			w.WriteHeader(http.StatusServiceUnavailable)
		}), []acquisitionStep{{token: reusableAccessToken(clock)}}, httpclient.RetryPolicy{MaxAttempts: 2, BaseDelay: time.Millisecond})
		response, err := fixture.authenticated.Do(fixture.request(t.Context(), t))
		closeResponse(t, response)
		assertFailureClass(t, err, FailureTokenUnusable)
		if got := fixture.resourceCalls.Load(); got != 1 {
			t.Fatalf("resource calls = %d, want 1", got)
		}
		if got := fixture.providerCalls.Load(); got != 1 {
			t.Fatalf("provider calls = %d, want no reacquisition", got)
		}
		if got := <-fixture.authorizations; got != "Bearer "+testBearerToken {
			t.Fatalf("first Authorization = %q", got)
		}
		select {
		case extra := <-fixture.authorizations:
			t.Fatalf("unexpected second resource Authorization = %q", extra)
		default:
		}
	})
}

func testHTTPClientTokenFailureReachesNoResource(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []acquisitionStep{{err: failure(FailureProviderUnavailable)}}, httpclient.RetryPolicy{})
	response, err := fixture.authenticated.Do(fixture.request(t.Context(), t))
	closeResponse(t, response)
	assertFailureClass(t, err, FailureProviderUnavailable)
	if got := fixture.resourceCalls.Load(); got != 0 {
		t.Fatalf("resource calls = %d, want 0", got)
	}
}

func testHTTPClientAlternateAuthorityAndRedirectDoNotForwardBearer(t *testing.T) {
	t.Parallel()
	clock := newMovableClock(fixedProviderTime)
	fixture := newHTTPFixture(t, clock, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://"+otherResourceHost+"/stolen")
		w.WriteHeader(http.StatusFound)
	}), []acquisitionStep{{token: reusableAccessToken(clock)}}, httpclient.RetryPolicy{})
	response, err := fixture.authenticated.Do(fixture.request(t.Context(), t))
	if err != nil {
		t.Fatalf("HTTPClient.Do(redirect) error = %v", err)
	}
	closeResponse(t, response)
	if response.StatusCode != http.StatusFound {
		t.Fatalf("redirect status = %d, want unchanged 302", response.StatusCode)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+otherResourceHost+"/stolen", http.NoBody)
	if err != nil {
		t.Fatalf("build alternate request: %v", err)
	}
	response, err = fixture.authenticated.Do(request)
	closeResponse(t, response)
	if !errors.Is(err, httpclient.ErrTargetDenied) {
		t.Fatalf("alternate authority error = %v, want target denial", err)
	}
	if got := fixture.resourceCalls.Load(); got != 1 {
		t.Fatalf("configured resource calls = %d, want redirect source only", got)
	}
	if authorization := <-fixture.authorizations; !strings.HasPrefix(authorization, "Bearer ") {
		t.Fatalf("configured resource Authorization = %q", authorization)
	}
}

// profile:outbound-auth-http:end
