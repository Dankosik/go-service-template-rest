package httpx

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Dankosik/billing-service/internal/api"
	"github.com/golang-jwt/jwt/v4"
)

const (
	serviceBearerAuthScheme = "Bearer "

	scopeAccountsResolve    = "billing.accounts.resolve"
	scopeBalancesRead       = "billing.balances.read"
	scopeUsageRead          = "billing.usage.read"
	scopeUsageWrite         = "billing.usage.write"
	scopeMicroleaseRead     = "billing.microleases.read"
	scopeMicroleaseWrite    = "billing.microleases.write"
	scopeOperationsRead     = "billing.operations.read"
	scopeReconciliationRead = "billing.reconciliation.read"
	scopeAdminRead          = "billing.admin.read"
)

var (
	ErrServiceAuthMissing = errors.New("service authentication missing")
	ErrServiceAuthInvalid = errors.New("service authentication invalid")
)

type ServiceAuthConfig struct {
	Enabled  bool
	Issuer   string
	Audience string
	JWKSURL  string
}

type ServicePrincipal struct {
	Subject string
	Scopes  []string
}

func (p ServicePrincipal) HasScope(scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return true
	}
	return slices.Contains(p.Scopes, scope)
}

type ServiceAuthenticator interface {
	AuthenticateService(context.Context, *http.Request) (ServicePrincipal, error)
}

type servicePrincipalContextKey struct{}

type serviceJWTClaims struct {
	Scope  string   `json:"scope,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

type RemoteJWKSAuthenticator struct {
	issuer   string
	audience string
	jwksURL  string
	client   *http.Client

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
}

func NewRemoteJWKSAuthenticator(cfg ServiceAuthConfig) (*RemoteJWKSAuthenticator, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("service auth is disabled")
	}
	if strings.TrimSpace(cfg.Issuer) == "" {
		return nil, fmt.Errorf("service auth issuer is required")
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		return nil, fmt.Errorf("service auth audience is required")
	}
	jwksURL := strings.TrimSpace(cfg.JWKSURL)
	parsed, err := url.Parse(jwksURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("service auth jwks url is invalid")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, fmt.Errorf("service auth jwks url is invalid")
	}
	return &RemoteJWKSAuthenticator{
		issuer:   strings.TrimSpace(cfg.Issuer),
		audience: strings.TrimSpace(cfg.Audience),
		jwksURL:  parsed.String(),
		client:   &http.Client{Timeout: 2 * time.Second},
		keys:     make(map[string]*rsa.PublicKey),
	}, nil
}

func (a *RemoteJWKSAuthenticator) AuthenticateService(ctx context.Context, r *http.Request) (ServicePrincipal, error) {
	if a == nil {
		return ServicePrincipal{}, ErrServiceAuthInvalid
	}
	tokenValue := bearerTokenFromRequest(r)
	if tokenValue == "" {
		return ServicePrincipal{}, ErrServiceAuthMissing
	}
	claims := serviceJWTClaims{}
	token, err := jwt.ParseWithClaims(tokenValue, &claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("%w: unsupported signing method", ErrServiceAuthInvalid)
		}
		kid, _ := token.Header["kid"].(string)
		if strings.TrimSpace(kid) == "" {
			return nil, fmt.Errorf("%w: missing key id", ErrServiceAuthInvalid)
		}
		key, keyErr := a.publicKey(ctx, kid)
		if keyErr != nil {
			return nil, keyErr
		}
		return key, nil
	})
	if err != nil || token == nil || !token.Valid {
		return ServicePrincipal{}, fmt.Errorf("%w: token validation failed", ErrServiceAuthInvalid)
	}
	if !claims.VerifyIssuer(a.issuer, true) {
		return ServicePrincipal{}, fmt.Errorf("%w: issuer mismatch", ErrServiceAuthInvalid)
	}
	if !claims.VerifyAudience(a.audience, true) {
		return ServicePrincipal{}, fmt.Errorf("%w: audience mismatch", ErrServiceAuthInvalid)
	}
	scopes := serviceAuthScopes(claims)
	return ServicePrincipal{Subject: claims.Subject, Scopes: scopes}, nil
}

func (a *RemoteJWKSAuthenticator) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	a.mu.RLock()
	key := a.keys[kid]
	a.mu.RUnlock()
	if key != nil {
		return key, nil
	}
	if err := a.refreshKeys(ctx); err != nil {
		return nil, err
	}
	a.mu.RLock()
	key = a.keys[kid]
	a.mu.RUnlock()
	if key == nil {
		return nil, fmt.Errorf("%w: unknown key id", ErrServiceAuthInvalid)
	}
	return key, nil
}

func (a *RemoteJWKSAuthenticator) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.jwksURL, nil) // #nosec G704 -- service-auth JWKS URL is startup-validated service config, not request input.
	if err != nil {
		return fmt.Errorf("%w: build jwks request: %w", ErrServiceAuthInvalid, err)
	}
	resp, err := a.client.Do(req) // #nosec G704 -- outbound target is the validated JWKS endpoint above.
	if err != nil {
		return fmt.Errorf("%w: fetch jwks: %w", ErrServiceAuthInvalid, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: jwks status %d", ErrServiceAuthInvalid, resp.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("%w: decode jwks: %w", ErrServiceAuthInvalid, err)
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, jwk := range doc.Keys {
		key, err := jwk.rsaPublicKey()
		if err != nil {
			return err
		}
		keys[jwk.Kid] = key
	}
	if len(keys) == 0 {
		return fmt.Errorf("%w: empty jwks", ErrServiceAuthInvalid)
	}
	a.mu.Lock()
	a.keys = keys
	a.mu.Unlock()
	return nil
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use,omitempty"`
	Alg string `json:"alg,omitempty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (k jwkKey) rsaPublicKey() (*rsa.PublicKey, error) {
	if k.Kty != "RSA" || strings.TrimSpace(k.Kid) == "" {
		return nil, fmt.Errorf("%w: unsupported jwk", ErrServiceAuthInvalid)
	}
	if k.Alg != "" && k.Alg != jwt.SigningMethodRS256.Alg() {
		return nil, fmt.Errorf("%w: unsupported jwk alg", ErrServiceAuthInvalid)
	}
	modulusBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid jwk modulus", ErrServiceAuthInvalid)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid jwk exponent", ErrServiceAuthInvalid)
	}
	exponent := 0
	for _, b := range exponentBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, fmt.Errorf("%w: invalid jwk exponent", ErrServiceAuthInvalid)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulusBytes), E: exponent}, nil
}

func serviceAuthScopes(claims serviceJWTClaims) []string {
	scopeSet := make(map[string]struct{})
	for _, scope := range claims.Scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopeSet[scope] = struct{}{}
		}
	}
	for scope := range strings.FieldsSeq(claims.Scope) {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopeSet[scope] = struct{}{}
		}
	}
	scopes := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	return scopes
}

func bearerTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, serviceBearerAuthScheme) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, serviceBearerAuthScheme))
}

func protectedServiceAuthMiddleware(auth ServiceAuthenticator) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requiredScope := requiredServiceScope(r)
			if requiredScope == "" {
				next.ServeHTTP(w, r)
				return
			}
			if auth == nil {
				writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "service authentication required")
				return
			}
			principal, err := auth.AuthenticateService(r.Context(), r)
			if err != nil {
				writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "service authentication required")
				return
			}
			if !principal.HasScope(requiredScope) {
				writeProblem(w, r, http.StatusForbidden, "forbidden", "required service scope is missing")
				return
			}
			ctx := context.WithValue(r.Context(), servicePrincipalContextKey{}, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requiredServiceScope(r *http.Request) string {
	if r == nil {
		return ""
	}
	switch r.Method + " " + strings.TrimSpace(r.URL.Path) {
	case "POST /internal/billing/v1/accounts/resolve":
		return scopeAccountsResolve
	case "POST /internal/billing/v1/usage/reservations",
		"POST /internal/billing/v1/usage/finalizations",
		"POST /internal/billing/v1/usage/write-offs",
		"POST /internal/billing/v1/usage/reversals":
		return scopeUsageWrite
	case "POST /internal/billing/v1/usage/readback":
		return scopeUsageRead
	case "POST /internal/billing/v1/microleases/issue", "POST /internal/billing/v1/microleases/close":
		return scopeMicroleaseWrite
	case "POST /internal/billing/v1/microleases/readback":
		return scopeMicroleaseRead
	case "POST /internal/billing/v1/operations/readback":
		return scopeOperationsRead
	case "GET /internal/billing/v1/reconciliation/cases":
		return scopeReconciliationRead
	}
	path := strings.TrimSpace(r.URL.Path)
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/internal/billing/v1/accounts/") && strings.HasSuffix(path, "/balance"):
		return scopeBalancesRead
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/internal/billing/v1/admin/accounts/") &&
		(strings.HasSuffix(path, "/ledger") || strings.HasSuffix(path, "/exposure")):
		return scopeAdminRead
	default:
		return ""
	}
}
