package oauthintrospection

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"

	// profile:grpc:start
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	// profile:grpc:end
)

func TestCanonicalIntrospectionRequest(t *testing.T) {
	clientID := "id with +%:"
	secret := "secret with +%:"
	token := "token with +%:"
	provider := newLoopbackProvider(t, nil)
	policy := mustPolicy(t, PolicyInput{
		Issuer:       testIssuer,
		Audience:     testAudience,
		Endpoint:     "https://idp.example.com/oauth/introspect",
		TargetClass:  "external-https",
		ClientID:     clientID,
		ClientSecret: secret,
	})
	verifier := newPinnedVerifierWithPolicy(t, provider, policy)
	result, err := verifier.Verify(t.Context(), token)
	if err != nil || result.Principal.Subject != "subject-1" || result.Principal.ClientID != "client-1" {
		t.Fatalf("Verify() = %+v, %v", result, err)
	}
	if got := strconv.FormatInt(provider.calls.Load(), 10); got != "1" {
		t.Fatalf("calls = %s, want 1", got)
	}
	got := provider.last()
	if got.method != http.MethodPost {
		t.Fatalf("method = %q", got.method)
	}
	if got.rawURL != "/oauth/introspect" {
		t.Fatalf("url = %q", got.rawURL)
	}
	if strings.Contains(got.rawURL, token) || strings.Contains(got.rawURL, url.QueryEscape(token)) {
		t.Fatal("token appeared in URL")
	}
	if got.header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("content-type = %q", got.header.Get("Content-Type"))
	}
	if got.header.Get("Accept") != "application/json" {
		t.Fatalf("accept = %q", got.header.Get("Accept"))
	}
	wantBasic := "Basic " + base64.StdEncoding.EncodeToString([]byte(url.QueryEscape(clientID)+":"+url.QueryEscape(secret)))
	if got.header.Get("Authorization") != wantBasic {
		t.Fatalf("authorization = %q", got.header.Get("Authorization"))
	}
	form := mustParseForm(t, got.body)
	if form.Get("token") != token || form.Get("token_type_hint") != "access_token" {
		t.Fatalf("form = %v", form)
	}
}

func TestFixedAuthorityOneAttempt(t *testing.T) {
	redirectHits := atomic.Int64{}
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirectHits.Add(1)
	}))
	t.Cleanup(redirect.Close)

	t.Setenv("HTTPS_PROXY", "https://proxy.invalid")
	t.Setenv("HTTP_PROXY", "http://proxy.invalid")

	constructed, err := New(testPolicy(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(constructed.Close)
	if !httpclient.ProxyDisabled(constructed.client) {
		t.Fatal("production constructor honored an ambient proxy")
	}

	var status int
	var closeBeforeHeaders bool
	provider := newLoopbackProvider(t, func(response http.ResponseWriter, _ *http.Request) {
		if closeBeforeHeaders {
			hj, ok := response.(http.Hijacker)
			if !ok {
				t.Fatal("response is not a hijacker")
			}
			conn, _, hijackErr := hj.Hijack()
			if hijackErr != nil {
				t.Fatalf("Hijack() error = %v", hijackErr)
			}
			_ = conn.Close()
			return
		}
		if status == http.StatusFound {
			response.Header().Set("Location", redirect.URL)
		}
		response.WriteHeader(status)
	})
	verifier := newPinnedVerifier(t, provider)
	if !httpclient.ProxyDisabled(verifier.client) {
		t.Fatal("pinned client enabled a proxy")
	}

	for _, code := range []int{http.StatusFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		status = code
		_, err := verifier.Verify(t.Context(), testToken)
		requireKind(t, err, bearerauthn.KindUnavailable)
	}
	status = 0
	closeBeforeHeaders = true
	_, err = verifier.Verify(t.Context(), testToken)
	requireKind(t, err, bearerauthn.KindUnavailable)
	if provider.calls.Load() != 4 || redirectHits.Load() != 0 {
		t.Fatalf("provider calls = %d, redirect hits = %d", provider.calls.Load(), redirectHits.Load())
	}
}

func TestProviderBoundaryAdmission(t *testing.T) {
	canary := "provider-body-canary"
	for _, testCase := range []struct {
		name   string
		status int
		ctype  string
		body   string
		wantOK bool
	}{
		{name: "json parameters", status: 200, ctype: "application/json; charset=utf-8", body: activeJSON("subject-1", "client-1"), wantOK: true},
		{name: "exact limit", status: 200, ctype: "application/json", body: exactLimitJSON(t), wantOK: true},
		{name: "no content", status: 204, ctype: "application/json", body: canary},
		{name: "found", status: 302, ctype: "application/json", body: canary},
		{name: "unauthorized", status: 401, ctype: "application/json", body: canary},
		{name: "too many", status: 429, ctype: "application/json", body: canary},
		{name: "server", status: 500, ctype: "application/json", body: canary},
		{name: "missing media", status: 200, body: activeJSON("subject-1", "client-1")},
		{name: "wrong media", status: 200, ctype: "text/plain", body: activeJSON("subject-1", "client-1")},
		{name: "malformed media", status: 200, ctype: "application/", body: activeJSON("subject-1", "client-1")},
		{name: "oversize", status: 200, ctype: "application/json", body: strings.Repeat("x", MaxProviderBody+1)},
		{name: "truncated", status: 200, ctype: "application/json", body: `{"active":true,"iss":"`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			provider := newLoopbackProvider(t, func(response http.ResponseWriter, _ *http.Request) {
				if testCase.ctype != "" {
					response.Header().Set("Content-Type", testCase.ctype)
				}
				response.WriteHeader(testCase.status)
				_, _ = io.WriteString(response, testCase.body)
			})
			verifier := newPinnedVerifier(t, provider)
			result, err := verifier.Verify(t.Context(), testToken)
			if testCase.wantOK {
				if err != nil || result.Principal.Subject != "subject-1" {
					t.Fatalf("Verify() = %+v, %v", result, err)
				}
				return
			}
			requireKind(t, err, bearerauthn.KindUnavailable)
			if strings.Contains(err.Error(), canary) {
				t.Fatalf("error disclosed canary: %v", err)
			}
			if provider.calls.Load() != 1 {
				t.Fatalf("calls = %d", provider.calls.Load())
			}
		})
	}

	t.Run("gzip over limit", func(t *testing.T) {
		var encoded bytes.Buffer
		writer := gzip.NewWriter(&encoded)
		_, _ = writer.Write([]byte(strings.Repeat("a", MaxProviderBody+1)))
		_ = writer.Close()
		provider := newLoopbackProvider(t, func(response http.ResponseWriter, _ *http.Request) {
			response.Header().Set("Content-Type", "application/json")
			response.Header().Set("Content-Encoding", "gzip")
			response.WriteHeader(http.StatusOK)
			_, _ = response.Write(encoded.Bytes())
		})
		verifier := newPinnedVerifier(t, provider)
		_, err := verifier.Verify(t.Context(), testToken)
		requireKind(t, err, bearerauthn.KindUnavailable)
	})

	t.Run("tls trust", func(t *testing.T) {
		untrusted := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("untrusted server was reached")
		}))
		t.Cleanup(untrusted.Close)
		policy := testPolicy(t)
		client, err := httpclient.NewExternalHTTPSWithLimits(policy.endpoint, httpclient.ResponseLimits{
			ResponseHeaderTimeout:  ProviderTimeout,
			MaxResponseHeaderBytes: MaxResponseHeaderBytes,
		})
		if err != nil {
			t.Fatalf("NewExternalHTTPSWithLimits() error = %v", err)
		}
		t.Cleanup(client.CloseIdleConnections)
		source, ok := untrusted.Client().Transport.(*http.Transport)
		if !ok {
			t.Fatal("httptest transport type")
		}
		httpclient.BindLoopbackTLS(client, source)
		httpclient.RejectLoopbackTLSTrust(client)
		verifier := newVerifier(policy, client, func() time.Time { return testNow })
		_, err = verifier.Verify(t.Context(), testToken)
		requireKind(t, err, bearerauthn.KindUnavailable)
	})

	t.Run("unreachable host", func(t *testing.T) {
		policy := mustPolicy(t, PolicyInput{
			Issuer:       testIssuer,
			Audience:     testAudience,
			Endpoint:     "https://idp.example.invalid/oauth/introspect",
			TargetClass:  "external-https",
			ClientID:     testClientID,
			ClientSecret: testSecret,
		})
		verifier, err := New(policy)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		t.Cleanup(verifier.Close)
		_, err = verifier.Verify(t.Context(), testToken)
		requireKind(t, err, bearerauthn.KindUnavailable)
	})
}

func exactLimitJSON(t *testing.T) string {
	t.Helper()
	body := activeJSON("subject-1", "client-1")
	pad := MaxProviderBody - len(body) - len(`,"pad":""`)
	if pad < 0 {
		return body
	}
	return strings.TrimSuffix(body, "}") + `,"pad":"` + strings.Repeat("a", pad) + `"}`
}

func TestProviderCancellationClassification(t *testing.T) {
	waitFor := func(t *testing.T, entered <-chan struct{}) {
		t.Helper()
		select {
		case <-entered:
		case <-time.After(7 * time.Second):
			t.Fatal("provider did not receive the request")
		}
	}

	t.Run("caller cancel", func(t *testing.T) {
		entered := make(chan struct{})
		provider := newLoopbackProvider(t, func(_ http.ResponseWriter, request *http.Request) {
			close(entered)
			<-request.Context().Done()
		})
		verifier := newPinnedVerifier(t, provider)
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() {
			_, err := verifier.Verify(ctx, testToken)
			done <- err
		}()
		waitFor(t, entered)
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want canceled", err)
			}
		case <-time.After(7 * time.Second):
			t.Fatal("canceled verify hung")
		}
	})

	t.Run("caller deadline", func(t *testing.T) {
		entered := make(chan struct{})
		provider := newLoopbackProvider(t, func(_ http.ResponseWriter, request *http.Request) {
			close(entered)
			<-request.Context().Done()
		})
		verifier := newPinnedVerifier(t, provider)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := verifier.Verify(ctx, testToken)
			done <- err
		}()
		waitFor(t, entered)
		select {
		case err := <-done:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want deadline", err)
			}
		case <-time.After(7 * time.Second):
			t.Fatal("deadline verify hung")
		}
	})

	t.Run("provider timeout", func(t *testing.T) {
		entered := make(chan struct{})
		provider := newLoopbackProvider(t, func(_ http.ResponseWriter, request *http.Request) {
			close(entered)
			<-request.Context().Done()
		})
		verifier := newPinnedVerifier(t, provider)
		done := make(chan error, 1)
		go func() {
			_, err := verifier.Verify(t.Context(), testToken)
			done <- err
		}()
		waitFor(t, entered)
		select {
		case err := <-done:
			requireKind(t, err, bearerauthn.KindUnavailable)
			if t.Context().Err() != nil {
				t.Fatal("caller context was done")
			}
		case <-time.After(7 * time.Second):
			t.Fatal("provider timeout hung")
		}
	})
}

func TestVerifierCloseIsIdempotentAndReleasesIdleConnections(t *testing.T) {
	var reused atomic.Bool
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			reused.Store(info.Reused)
		},
	}
	provider := newLoopbackProvider(t, nil)
	verifier := newPinnedVerifier(t, provider)
	ctx := httptrace.WithClientTrace(t.Context(), trace)
	if _, err := verifier.Verify(ctx, testToken); err != nil {
		t.Fatalf("first Verify() error = %v", err)
	}
	if reused.Load() {
		t.Fatal("first request reused a connection")
	}
	verifier.Close()
	verifier.Close()
	if _, err := verifier.Verify(ctx, testToken); err != nil {
		t.Fatalf("post-close Verify() error = %v", err)
	}
	if reused.Load() {
		t.Fatal("idle connection was reused after close")
	}
}

func TestUncachedIndependentDecisions(t *testing.T) {
	var next atomic.Int64
	release := make(chan struct{})
	var entered sync.WaitGroup
	provider := newLoopbackProvider(t, func(response http.ResponseWriter, _ *http.Request) {
		n := next.Add(1)
		if n == 3 || n == 4 {
			entered.Done()
			<-release
		}
		response.Header().Set("Content-Type", "application/json")
		switch n {
		case 5:
			response.WriteHeader(http.StatusInternalServerError)
		case 7:
			_, _ = io.WriteString(response, `{"active":false}`)
		default:
			_, _ = io.WriteString(response, activeJSON("subject-1", "client-1"))
		}
	})
	verifier := newPinnedVerifier(t, provider)

	if _, err := verifier.Verify(t.Context(), testToken); err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(t.Context(), testToken); err != nil {
		t.Fatal(err)
	}

	entered.Add(2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			<-start
			_, _ = verifier.Verify(t.Context(), testToken)
		}()
	}
	close(start)
	done := make(chan struct{})
	go func() {
		entered.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(7 * time.Second):
		t.Fatal("concurrent requests did not both enter the provider")
	}
	if provider.peak.Load() < 2 {
		t.Fatalf("concurrent peak = %d, want at least 2", provider.peak.Load())
	}
	close(release)
	wg.Wait()

	if _, err := verifier.Verify(t.Context(), testToken); err == nil {
		t.Fatal("expected provider failure")
	}
	if _, err := verifier.Verify(t.Context(), testToken); err != nil {
		t.Fatalf("recovery error = %v", err)
	}
	if _, err := verifier.Verify(t.Context(), testToken); err == nil {
		t.Fatal("expected inactive")
	}
	if _, err := verifier.Verify(t.Context(), testToken); err != nil {
		t.Fatalf("inactive then active error = %v", err)
	}
	if provider.calls.Load() != 8 {
		t.Fatalf("calls = %d, want 8", provider.calls.Load())
	}
}

// profile:grpc:start

func bearerAuthInput(request *http.Request) *openapi3filter.AuthenticationInput {
	return &openapi3filter.AuthenticationInput{
		SecuritySchemeName: "bearerAuth",
		SecurityScheme:     &openapi3.SecurityScheme{Type: "http", Scheme: "bearer"},
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: request,
		},
	}
}

func invokeGRPC(ctx context.Context, t *testing.T, runtime *bearerauthn.Runtime) (reqctx.Principal, error) {
	t.Helper()
	var principal reqctx.Principal
	_, err := runtime.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/example.v1.API/Get"},
		func(ctx context.Context, _ any) (any, error) {
			var ok bool
			principal, ok = reqctx.PrincipalFromContext(ctx)
			if !ok {
				return struct{}{}, errMissingPrincipal
			}
			return struct{}{}, nil
		})
	return principal, err
}

func grpcIncoming(ctx context.Context, token string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
}

var errMissingPrincipal = context.Canceled

func TestHTTPAndGRPCPrincipalParity(t *testing.T) {
	exp := strconv.FormatInt(testNow.Add(time.Hour).Unix(), 10)
	provider := newLoopbackProvider(t, func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"active":true,"iss":"`+testIssuer+`","aud":"`+testAudience+
			`","exp":`+exp+`,"sub":"subject-1","client_id":"client-1","scope":"admin","username":"u"}`)
	})
	verifier := newPinnedVerifier(t, provider)
	runtime, err := bearerauthn.New(verifier, nil)
	if err != nil {
		t.Fatalf("bearerauthn.New() error = %v", err)
	}

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/private", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+testToken)
	httpPrincipal, err := runtime.ResolveHTTP(t.Context(), bearerAuthInput(request))
	if err != nil {
		t.Fatalf("ResolveHTTP() error = %v", err)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("HTTP forwarded the credential")
	}

	grpcPrincipal, err := invokeGRPC(grpcIncoming(t.Context(), testToken), t, runtime)
	if err != nil {
		t.Fatalf("gRPC error = %v", err)
	}
	if httpPrincipal.Issuer != grpcPrincipal.Issuer || httpPrincipal.Subject != grpcPrincipal.Subject || httpPrincipal.ClientID != grpcPrincipal.ClientID || httpPrincipal.Subject != "subject-1" || httpPrincipal.ClientID != "client-1" {
		t.Fatalf("http=%+v grpc=%+v", httpPrincipal, grpcPrincipal)
	}
	if provider.calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", provider.calls.Load())
	}
}

// profile:grpc:end

func TestIntrospectionDisclosureBoundary(t *testing.T) {
	canaries := []string{"token-canary", "secret-canary"}
	provider := newLoopbackProvider(t, func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(response, strings.Join(canaries, " "))
	})
	policy := mustPolicy(t, PolicyInput{
		Issuer:       testIssuer,
		Audience:     testAudience,
		Endpoint:     "https://idp.example.com/oauth/introspect",
		TargetClass:  "external-https",
		ClientID:     "id-value",
		ClientSecret: canaries[1],
	})
	verifier := newPinnedVerifierWithPolicy(t, provider, policy)
	_, err := verifier.Verify(t.Context(), canaries[0])
	requireKind(t, err, bearerauthn.KindUnavailable)
	rendered := fmt.Sprintf("%s %v %+v", err, err, err)
	for _, canary := range canaries {
		if strings.Contains(rendered, canary) {
			t.Fatalf("disclosed %q", canary)
		}
	}
}
