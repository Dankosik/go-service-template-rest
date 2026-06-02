package httpx

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestRemoteJWKSAuthenticatorValidatesSignedServiceTokensAndScopes(t *testing.T) {
	t.Parallel()

	key := mustTestRSAKey(t)
	hits := 0
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if err := json.NewEncoder(w).Encode(jwksDocument{Keys: []jwkKey{testJWK("kid-1", key)}}); err != nil {
			t.Fatalf("encode jwks: %v", err)
		}
	}))
	t.Cleanup(jwks.Close)

	auth, err := NewRemoteJWKSAuthenticator(ServiceAuthConfig{
		Enabled:  true,
		Issuer:   "https://issuer.example",
		Audience: "billing-service",
		JWKSURL:  jwks.URL,
	})
	if err != nil {
		t.Fatalf("NewRemoteJWKSAuthenticator() error = %v", err)
	}
	auth.client = jwks.Client()
	token := mustServiceToken(t, key, "kid-1", serviceJWTClaims{
		Scope:  "billing.usage.write billing.usage.read",
		Scopes: []string{"billing.admin.read", "billing.usage.write"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://issuer.example",
			Audience:  jwt.ClaimStrings{"billing-service"},
			Subject:   "worker-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/usage/reservations", nil)
	req.Header.Set("Authorization", serviceBearerAuthScheme+token)
	principal, err := auth.AuthenticateService(context.Background(), req)
	if err != nil {
		t.Fatalf("AuthenticateService(valid token) error = %v", err)
	}
	if principal.Subject != "worker-1" || !slices.Equal(principal.Scopes, []string{"billing.admin.read", "billing.usage.read", "billing.usage.write"}) {
		t.Fatalf("principal = %+v, want sorted deduped scopes", principal)
	}

	if _, err := auth.AuthenticateService(context.Background(), req); err != nil {
		t.Fatalf("AuthenticateService(cached key) error = %v", err)
	}
	if hits != 1 {
		t.Fatalf("JWKS fetches = %d, want cached key after first fetch", hits)
	}
}

func TestRemoteJWKSAuthenticatorRejectsUnsafeConfigAndTokens(t *testing.T) {
	t.Parallel()

	for _, cfg := range []ServiceAuthConfig{
		{},
		{Enabled: true, Audience: "billing-service", JWKSURL: "https://issuer.example/jwks"},
		{Enabled: true, Issuer: "issuer", JWKSURL: "https://issuer.example/jwks"},
		{Enabled: true, Issuer: "issuer", Audience: "billing-service", JWKSURL: "://bad"},
		{Enabled: true, Issuer: "issuer", Audience: "billing-service", JWKSURL: "file:///tmp/jwks"},
	} {
		if _, err := NewRemoteJWKSAuthenticator(cfg); err == nil {
			t.Fatalf("NewRemoteJWKSAuthenticator(%+v) error = nil, want invalid config rejection", cfg)
		}
	}

	auth, err := NewRemoteJWKSAuthenticator(ServiceAuthConfig{
		Enabled:  true,
		Issuer:   "issuer",
		Audience: "billing-service",
		JWKSURL:  "http://127.0.0.1/jwks",
	})
	if err != nil {
		t.Fatalf("NewRemoteJWKSAuthenticator(valid) error = %v", err)
	}
	if _, err := auth.AuthenticateService(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil)); err == nil {
		t.Fatal("AuthenticateService(missing bearer) error = nil, want missing auth")
	}

	hsToken := jwt.NewWithClaims(jwt.SigningMethodHS256, serviceJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "issuer",
			Audience:  jwt.ClaimStrings{"billing-service"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	hsToken.Header["kid"] = "kid-1"
	signed, err := hsToken.SignedString([]byte("test-signing-key"))
	if err != nil {
		t.Fatalf("sign hs token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", serviceBearerAuthScheme+signed)
	if _, err := auth.AuthenticateService(context.Background(), req); err == nil {
		t.Fatal("AuthenticateService(unsupported alg) error = nil, want invalid auth")
	}
}

func TestServiceAuthMiddlewareEnforcesRouteScopesAndPrincipalContext(t *testing.T) {
	t.Parallel()

	auth := &fakeServiceAuthenticator{principal: ServicePrincipal{Subject: "svc", Scopes: []string{scopeUsageWrite}}}
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		principal, ok := r.Context().Value(servicePrincipalContextKey{}).(ServicePrincipal)
		if !ok || principal.Subject != "svc" {
			t.Fatalf("principal context = %+v, %v; want service principal", principal, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/usage/finalizations", nil)
	resp := httptest.NewRecorder()
	protectedServiceAuthMiddleware(auth)(next).ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent || !nextCalled {
		t.Fatalf("authorized response = %d next=%v, want no-content and next call", resp.Code, nextCalled)
	}

	resp = httptest.NewRecorder()
	protectedServiceAuthMiddleware(auth)(next).ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/internal/billing/v1/admin/accounts/acct/ledger", nil))
	if resp.Code != http.StatusForbidden {
		t.Fatalf("missing scope response = %d, want forbidden", resp.Code)
	}

	resp = httptest.NewRecorder()
	protectedServiceAuthMiddleware(nil)(next).ServeHTTP(resp, httptest.NewRequest(http.MethodPost, "/internal/billing/v1/accounts/resolve", nil))
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("nil auth response = %d, want unauthorized", resp.Code)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	resp = httptest.NewRecorder()
	nextCalled = false
	protectedServiceAuthMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(resp, publicReq)
	if resp.Code != http.StatusAccepted || !nextCalled {
		t.Fatalf("public route response = %d next=%v, want bypassed auth", resp.Code, nextCalled)
	}
}

func TestRequiredServiceScopesCoverBillingAuthorityRoutes(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"POST /internal/billing/v1/accounts/resolve":            scopeAccountsResolve,
		"GET /internal/billing/v1/accounts/acct/balance":        scopeBalancesRead,
		"POST /internal/billing/v1/usage/reservations":          scopeUsageWrite,
		"POST /internal/billing/v1/usage/readback":              scopeUsageRead,
		"POST /internal/billing/v1/microleases/issue":           scopeMicroleaseWrite,
		"POST /internal/billing/v1/microleases/readback":        scopeMicroleaseRead,
		"POST /internal/billing/v1/operations/readback":         scopeOperationsRead,
		"GET /internal/billing/v1/reconciliation/cases":         scopeReconciliationRead,
		"GET /internal/billing/v1/admin/accounts/acct/ledger":   scopeAdminRead,
		"GET /internal/billing/v1/admin/accounts/acct/exposure": scopeAdminRead,
		"GET /api/v1/ping": "",
	}
	for route, want := range cases {
		method, path, _ := strings.Cut(route, " ")
		req := httptest.NewRequest(method, path, nil)
		if got := requiredServiceScope(req); got != want {
			t.Fatalf("requiredServiceScope(%s) = %q, want %q", route, got, want)
		}
	}
	if got := bearerTokenFromRequest(nil); got != "" {
		t.Fatalf("bearerTokenFromRequest(nil) = %q, want empty", got)
	}
}

func TestAuthorityProblemResponsesMapApprovedStatuses(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), requestIDContextKey{}, "req-test")
	tests := []struct {
		name     string
		statuses []int
		call     func(context.Context, error) any
	}{
		{"resolve", []int{400, 401, 403, 423, 503, 500}, func(ctx context.Context, err error) any { return resolveBillingAccountProblemResponse(ctx, err) }},
		{"balance", []int{400, 401, 403, 423, 503, 500}, func(ctx context.Context, err error) any { return readBillingBalanceProblemResponse(ctx, err) }},
		{"reserve", []int{400, 401, 403, 409, 422, 423, 429, 503, 500}, func(ctx context.Context, err error) any { return reserveUsageProblemResponse(ctx, err) }},
		{"finalize", []int{400, 401, 403, 409, 422, 423, 503, 500}, func(ctx context.Context, err error) any { return finalizeUsageProblemResponse(ctx, err) }},
		{"writeoff", []int{400, 401, 403, 409, 422, 423, 503, 500}, func(ctx context.Context, err error) any { return writeOffUsageProblemResponse(ctx, err) }},
		{"reverse", []int{400, 401, 403, 409, 422, 423, 503, 500}, func(ctx context.Context, err error) any { return reverseUsageProblemResponse(ctx, err) }},
		{"usage_readback", []int{400, 401, 403, 503, 500}, func(ctx context.Context, err error) any { return readUsageOperationProblemResponse(ctx, err) }},
		{"reconciliation", []int{400, 401, 403, 503, 500}, func(ctx context.Context, err error) any { return listReconciliationCasesProblemResponse(ctx, err) }},
		{"admin_ledger", []int{400, 401, 403, 503, 500}, func(ctx context.Context, err error) any { return readAdminLedgerProblemResponse(ctx, err) }},
		{"admin_exposure", []int{400, 401, 403, 503, 500}, func(ctx context.Context, err error) any { return readAdminExposureProblemResponse(ctx, err) }},
	}
	for _, tt := range tests {
		for _, status := range tt.statuses {
			got := tt.call(ctx, NewProblemError(status, "safe title", "safe detail"))
			if got == nil || !strings.Contains(reflect.TypeOf(got).String(), reflect.TypeOf(got).Name()) {
				t.Fatalf("%s status %d returned invalid response %#v", tt.name, status, got)
			}
		}
		got := tt.call(ctx, NewProblemError(http.StatusTeapot, "", ""))
		if got == nil {
			t.Fatalf("%s default status returned nil", tt.name)
		}
	}
}

func TestMicroleaseProblemResponsesMapApprovedStatuses(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), requestIDContextKey{}, "req-test")
	tests := []struct {
		name     string
		statuses []int
		call     func(context.Context, error) any
	}{
		{"issue", []int{400, 401, 403, 409, 422, 423, 429, 503, 500}, func(ctx context.Context, err error) any { return issueMicroleaseProblemResponse(ctx, err) }},
		{"read", []int{400, 401, 403, 503, 500}, func(ctx context.Context, err error) any { return readMicroleaseProblemResponse(ctx, err) }},
		{"close", []int{400, 401, 403, 409, 422, 423, 503, 500}, func(ctx context.Context, err error) any { return closeMicroleaseProblemResponse(ctx, err) }},
		{"operation", []int{400, 401, 403, 503, 500}, func(ctx context.Context, err error) any { return readBillingOperationProblemResponse(ctx, err) }},
	}
	for _, tt := range tests {
		for _, status := range tt.statuses {
			got := tt.call(ctx, NewProblemError(status, "safe title", "safe detail"))
			if got == nil {
				t.Fatalf("%s status %d returned nil", tt.name, status)
			}
		}
		if got := tt.call(ctx, errors.New("repository unavailable")); got == nil {
			t.Fatalf("%s default error returned nil", tt.name)
		}
	}
}

type fakeServiceAuthenticator struct {
	principal ServicePrincipal
	err       error
}

func (a *fakeServiceAuthenticator) AuthenticateService(context.Context, *http.Request) (ServicePrincipal, error) {
	return a.principal, a.err
}

func mustTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	return key
}

func mustServiceToken(t *testing.T, key *rsa.PrivateKey, kid string, claims serviceJWTClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign service token: %v", err)
	}
	return signed
}

func testJWK(kid string, key *rsa.PrivateKey) jwkKey {
	return jwkKey{
		Kty: "RSA",
		Kid: kid,
		Alg: jwt.SigningMethodRS256.Alg(),
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}
