package oidcjwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	jose "github.com/go-jose/go-jose/v4"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	testIssuer   = "https://issuer.example.com"
	testAudience = "service-api"
	testJWKSURI  = "https://keys.example.net/jwks"
)

type scriptedResponse struct {
	status  int
	body    []byte
	err     error
	wait    <-chan struct{}
	started chan<- struct{}
	panic   any
}

type scriptedClient struct {
	mu        sync.Mutex
	responses []scriptedResponse
	calls     int
}

func (c *scriptedClient) Do(request *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.calls++
	if len(c.responses) == 0 {
		c.mu.Unlock()
		return nil, errors.New("poison network error")
	}
	response := c.responses[0]
	c.responses = c.responses[1:]
	c.mu.Unlock()
	if response.started != nil {
		close(response.started)
	}
	if response.wait != nil {
		select {
		case <-response.wait:
		case <-request.Context().Done():
			return nil, fmt.Errorf("scripted provider request: %w", request.Context().Err())
		}
	}
	select {
	case <-request.Context().Done():
		return nil, fmt.Errorf("scripted provider request: %w", request.Context().Err())
	default:
	}
	if response.panic != nil {
		panic(response.panic)
	}
	if response.err != nil {
		return nil, response.err
	}
	return &http.Response{
		StatusCode: response.status,
		Body:       io.NopCloser(strings.NewReader(string(response.body))),
		Header:     make(http.Header),
	}, nil
}

func (c *scriptedClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type tokenClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub,omitempty"`
	Audience  any      `json:"aud"`
	ClientID  string   `json:"client_id"`
	JWTID     string   `json:"jti"`
	Expires   int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore *int64   `json:"nbf,omitempty"`
	Scope     []string `json:"scope,omitempty"`
	Roles     []string `json:"roles,omitempty"`
}

func TestVerify_Claims(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	otherKey := loadTestRSAKey(t, "test-key-2.pem")
	base := validClaims(now)

	tests := []struct {
		name     string
		key      *rsa.PrivateKey
		typ      string
		claims   tokenClaims
		wantKind Kind
	}{
		{name: "valid token", key: key, typ: "at+jwt", claims: base},
		{name: "valid scalar audience", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Audience = testAudience })},
		{name: "media type typ", key: key, typ: "APPLICATION/AT+JWT", claims: base},
		{name: "wrong issuer", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Issuer = "https://wrong.example" }), wantKind: KindInvalid},
		{name: "wrong audience", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Audience = "other" }), wantKind: KindInvalid},
		{name: "empty audience", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Audience = "" }), wantKind: KindInvalid},
		{name: "missing audience", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Audience = nil }), wantKind: KindInvalid},
		{name: "duplicate audience", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Audience = []string{testAudience, testAudience} }), wantKind: KindInvalid},
		{name: "wrong typ", key: key, typ: "JWT", claims: base, wantKind: KindInvalid},
		{name: "invalid signature", key: otherKey, typ: "at+jwt", claims: base, wantKind: KindInvalid},
		{name: "expired", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Expires = now.Add(-ClockSkew).Unix() }), wantKind: KindInvalid},
		{name: "premature", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { value := now.Add(ClockSkew + time.Second).Unix(); c.NotBefore = &value }), wantKind: KindInvalid},
		{name: "future issued", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.IssuedAt = now.Add(ClockSkew + time.Second).Unix() }), wantKind: KindInvalid},
		{name: "missing subject", key: key, typ: "at+jwt", claims: mutateClaims(base, func(c *tokenClaims) { c.Subject = "" }), wantKind: KindInvalid},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			verifier := newTestVerifier(t, now, key)
			token := signToken(t, testCase.key, "key-1", testCase.typ, testCase.claims)
			principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
			requireKind(t, err, testCase.wantKind)
			if testCase.wantKind == 0 {
				if principal.Subject != base.Subject || len(principal.Scopes) != 0 {
					t.Fatalf("principal = %+v, want opaque subject and no scopes", principal)
				}
			}
		})
	}
}

func TestPolicyRejectsDuplicateTrustedProxyCIDRs(t *testing.T) {
	_, err := NewPolicy(testIssuer, testAudience, "127.0.0.0/8,127.0.0.1/8")
	if err == nil {
		t.Fatal("NewPolicy() error = nil, want duplicate CIDR rejection")
	}
}

func TestVerify_Serialization(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	client := &scriptedClient{responses: initialResponses(t, key)}
	verifier := newTestVerifierWithClient(t, now, client)
	valid := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	parts := strings.Split(valid, ".")
	validPayload, err := json.Marshal(validClaims(now))
	if err != nil {
		t.Fatal(err)
	}

	duplicateHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","alg":"RS256","typ":"at+jwt","kid":"key-1"}`))
	disallowedHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"at+jwt","kid":"key-1"}`))
	rs384Header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS384","typ":"at+jwt","kid":"key-1"}`))
	unsignedHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"at+jwt","kid":"key-1"}`))
	b64Header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"at+jwt","kid":"key-1","b64":false}`))
	x5uHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"at+jwt","kid":"key-1","x5u":"https://attacker.example/cert"}`))
	jwkHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"at+jwt","kid":"key-1","jwk":{"kty":"RSA"}}`))
	tests := map[string]string{
		"malformed":             "not-a-token",
		"two segments":          parts[0] + "." + parts[1],
		"four segments":         valid + ".extra",
		"JWE shape":             "a.b.c.d.e",
		"detached payload":      parts[0] + ".." + parts[2],
		"empty signature":       parts[0] + "." + parts[1] + ".",
		"unsigned":              unsignedHeader + "." + parts[1] + ".",
		"disallowed algorithm":  disallowedHeader + "." + parts[1] + "." + parts[2],
		"RS384":                 rs384Header + "." + parts[1] + "." + parts[2],
		"duplicate header":      duplicateHeader + "." + parts[1] + "." + parts[2],
		"padded segment":        parts[0] + "=." + parts[1] + "." + parts[2],
		"b64 header":            b64Header + "." + parts[1] + "." + parts[2],
		"x5u header":            x5uHeader + "." + parts[1] + "." + parts[2],
		"embedded jwk header":   jwkHeader + "." + parts[1] + "." + parts[2],
		"missing typ":           signTokenWithHeader(t, key, map[jose.HeaderKey]any{"kid": "key-1"}, validClaims(now)),
		"missing kid":           signTokenWithHeader(t, key, map[jose.HeaderKey]any{"typ": "at+jwt"}, validClaims(now)),
		"empty kid":             signToken(t, key, "", "at+jwt", validClaims(now)),
		"parameterized typ":     signToken(t, key, "key-1", "at+jwt; charset=utf-8", validClaims(now)),
		"critical header":       signTokenWithHeader(t, key, map[jose.HeaderKey]any{"typ": "at+jwt", "kid": "key-1", "crit": []string{"exp"}}, validClaims(now)),
		"remote jwk header":     signTokenWithHeader(t, key, map[jose.HeaderKey]any{"typ": "at+jwt", "kid": "key-1", "jku": "https://attacker.example/jwks"}, validClaims(now)),
		"duplicate payload":     signPayload(t, key, []byte(`{"iss":"`+testIssuer+`","iss":"`+testIssuer+`"}`)),
		"trailing payload data": signPayload(t, key, append(validPayload, []byte(`{}`)...)),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := verifier.Verify(t.Context(), token, TransportHTTP)
			requireKind(t, err, KindInvalid)
			if client.callCount() != 2 {
				t.Fatalf("locally invalid token caused %d provider calls, want initial discovery and JWKS only", client.callCount())
			}
		})
	}
}

func TestKeySetAdmission(t *testing.T) {
	key := loadTestRSAKey(t, "test-key-1.pem")
	valid, err := publicJWK(&key.PublicKey, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(map[string]any) map[string]any
		valid  bool
	}{
		{name: "compatible public RSA key", valid: true, mutate: func(jwk map[string]any) map[string]any { return jwk }},
		{name: "missing kid", mutate: func(jwk map[string]any) map[string]any { delete(jwk, "kid"); return jwk }},
		{name: "wrong algorithm", mutate: func(jwk map[string]any) map[string]any { jwk["alg"] = "HS256"; return jwk }},
		{name: "wrong use", mutate: func(jwk map[string]any) map[string]any { jwk["use"] = "enc"; return jwk }},
		{name: "sign-only key operation", mutate: func(jwk map[string]any) map[string]any {
			jwk["key_ops"] = []string{"sign"}
			return jwk
		}},
		{name: "undersized RSA key", mutate: func(jwk map[string]any) map[string]any {
			modulus := make([]byte, 128)
			modulus[0] = 0x80
			jwk["n"] = base64.RawURLEncoding.EncodeToString(modulus)
			return jwk
		}},
		{name: "symmetric key", mutate: func(map[string]any) map[string]any {
			return map[string]any{
				"kty": "oct",
				"kid": "key-1",
				"alg": "RS256",
				"use": "sig",
				"k":   base64.RawURLEncoding.EncodeToString([]byte("not-an-rsa-key")),
			}
		}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var jwk map[string]any
			if err := json.Unmarshal(valid, &jwk); err != nil {
				t.Fatal(err)
			}
			candidate, err := json.Marshal(map[string]any{"keys": []any{testCase.mutate(jwk)}})
			if err != nil {
				t.Fatal(err)
			}
			keySet, err := parseKeySet(candidate, time.Unix(1_900_000_000, 0))
			if !testCase.valid {
				if err == nil {
					t.Fatalf("parseKeySet() accepted incompatible key: %#v", keySet)
				}
				return
			}
			if err != nil || len(keySet.keys) != 1 || keySet.keys["key-1"] == nil {
				t.Fatalf("parseKeySet(valid) = (%#v, %v), want one public RSA key", keySet, err)
			}
		})
	}
}

func TestVerifyRejectsExplicitNullNotBefore(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	claims := validClaims(now)
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	raw["nbf"] = nil
	payload, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}

	verifier := newTestVerifier(t, now, key)
	_, err = verifier.Verify(t.Context(), signPayload(t, key, payload), TransportHTTP)
	requireKind(t, err, KindInvalid)
}

func TestVerify_TimePolicy(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	base := map[string]any{
		"iss":       testIssuer,
		"sub":       "opaque-subject",
		"aud":       []string{"another", testAudience},
		"client_id": "client-1",
		"jti":       "token-1",
		"exp":       now.Add(5 * time.Minute).Unix(),
		"iat":       now.Unix(),
	}
	tests := []struct {
		name     string
		mutate   func(map[string]any)
		wantKind Kind
	}{
		{
			name: "expiration just inside skew is accepted",
			mutate: func(claims map[string]any) {
				claims["exp"] = now.Add(-ClockSkew + time.Second).Unix()
			},
		},
		{
			name: "expiration at skew boundary is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = now.Add(-ClockSkew).Unix()
			},
			wantKind: KindInvalid,
		},
		{
			name: "not-before at skew boundary is accepted",
			mutate: func(claims map[string]any) {
				claims["nbf"] = now.Add(ClockSkew).Unix()
			},
		},
		{
			name: "not-before beyond skew is rejected",
			mutate: func(claims map[string]any) {
				claims["nbf"] = now.Add(ClockSkew + time.Second).Unix()
			},
			wantKind: KindInvalid,
		},
		{
			name: "issued-at at skew boundary is accepted",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(ClockSkew).Unix()
			},
		},
		{
			name: "issued-at beyond skew is rejected",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(ClockSkew + time.Second).Unix()
			},
			wantKind: KindInvalid,
		},
		{
			name: "fractional expiration is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1900000000.5")
			},
			wantKind: KindInvalid,
		},
		{
			name: "overflowing expiration is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("9223372036854775808")
			},
			wantKind: KindInvalid,
		},
		{
			name: "null not-before is rejected",
			mutate: func(claims map[string]any) {
				claims["nbf"] = nil
			},
			wantKind: KindInvalid,
		},
		{
			name: "missing expiration is rejected",
			mutate: func(claims map[string]any) {
				delete(claims, "exp")
			},
			wantKind: KindInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			claims := make(map[string]any, len(base))
			maps.Copy(claims, base)
			testCase.mutate(claims)
			payload, err := json.Marshal(claims)
			if err != nil {
				t.Fatalf("marshal claims: %v", err)
			}
			verifier := newTestVerifier(t, now, key)
			principal, err := verifier.Verify(
				t.Context(),
				signPayload(t, key, payload),
				TransportHTTP,
			)
			requireKind(t, err, testCase.wantKind)
			if testCase.wantKind == 0 && principal.Subject != "opaque-subject" {
				t.Fatalf("principal = %+v, want opaque subject", principal)
			}
		})
	}
}

func TestKeyMissRefresh(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	rotatedJWKS := jwksDocument(t, second, "key-2")
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status: http.StatusOK,
		body:   rotatedJWKS,
	})}
	verifier := newTestVerifierWithClient(t, now, client)
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))
	principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
	if err != nil || principal.Subject == "" {
		t.Fatalf("Verify(rotated token) = (%+v, %v), want success", principal, err)
	}
	if client.callCount() != 3 {
		t.Fatalf("provider calls = %d, want discovery + initial JWKS + one refresh", client.callCount())
	}

	outage := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		err: errors.New("poison issuer outage"),
	})}
	outageVerifier := newTestVerifierWithClient(t, now, outage)
	unknown := signToken(t, second, "unknown", "at+jwt", validClaims(now))
	_, err = outageVerifier.Verify(t.Context(), unknown, TransportHTTP)
	requireKind(t, err, KindInvalid)
	valid := signToken(t, first, "key-1", "at+jwt", validClaims(now))
	principal, err = outageVerifier.Verify(t.Context(), valid, TransportHTTP)
	if err != nil || principal.Subject == "" {
		t.Fatalf("Verify(cached valid token after outage) = (%+v, %v), want success", principal, err)
	}
}

func TestSameKIDRotation(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	client := &scriptedClient{responses: append(initialResponses(t, first),
		scriptedResponse{status: http.StatusOK, body: jwksDocument(t, second, "key-1")},
		scriptedResponse{err: errors.New("sequential poison outage")},
		scriptedResponse{err: errors.New("post-cooldown poison outage")},
	)}
	verifier, err := newVerifier(
		t.Context(),
		testPolicy(t),
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		func() time.Time { return now },
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)

	rotated := signToken(t, second, "key-1", "at+jwt", validClaims(now))
	if principal, verifyErr := verifier.Verify(t.Context(), rotated, TransportHTTP); verifyErr != nil ||
		principal.Subject == "" {
		t.Fatalf("Verify(same-kid rotation) = (%+v, %v), want success", principal, verifyErr)
	}
	if client.callCount() != 3 {
		t.Fatalf("provider calls after same-kid rotation = %d, want 3", client.callCount())
	}

	unknown := signToken(t, first, "unknown", "at+jwt", validClaims(now))
	_, err = verifier.Verify(t.Context(), unknown, TransportHTTP)
	requireKind(t, err, KindInvalid)
	if client.callCount() != 3 {
		t.Fatalf("provider calls during rotation cooldown = %d, want 3", client.callCount())
	}

	now = now.Add(RefreshCooldown)
	_, err = verifier.Verify(t.Context(), unknown, TransportHTTP)
	requireKind(t, err, KindInvalid)
	if client.callCount() != 4 {
		t.Fatalf("provider calls at cooldown boundary = %d, want 4", client.callCount())
	}
	_, err = verifier.Verify(t.Context(), unknown, TransportHTTP)
	requireKind(t, err, KindInvalid)
	if client.callCount() != 4 {
		t.Fatalf("provider calls inside sequential cooldown = %d, want 4", client.callCount())
	}
}

func TestRejectedJWKSIsAtomic(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	duplicate := append([]byte(`{"keys":[`), jwksDocument(t, second, "key-2")[9:len(jwksDocument(t, second, "key-2"))-2]...)
	duplicate = append(duplicate, ',')
	duplicate = append(duplicate, jwksDocument(t, first, "key-2")[9:len(jwksDocument(t, first, "key-2"))-2]...)
	duplicate = append(duplicate, ']', '}')

	for _, candidate := range [][]byte{
		[]byte(`{"keys":[]}`),
		duplicate,
		[]byte(`{"keys":[{"kid":"bad","kty":"RSA","alg":"HS256","use":"sig","n":"AQ","e":"Aw"}]}`),
	} {
		client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
			status: http.StatusOK,
			body:   candidate,
		})}
		verifier := newTestVerifierWithClient(t, now, client)
		unknown := signToken(t, second, "key-2", "at+jwt", validClaims(now))
		_, err := verifier.Verify(t.Context(), unknown, TransportHTTP)
		requireKind(t, err, KindInvalid)

		known := signToken(t, first, "key-1", "at+jwt", validClaims(now))
		principal, err := verifier.Verify(t.Context(), known, TransportHTTP)
		if err != nil || principal.Subject == "" {
			t.Fatalf("Verify(known token after invalid refresh) = (%+v, %v), want success", principal, err)
		}
	}
}

func TestStaleUnknownKIDPerformsRequestDrivenRecovery(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status: http.StatusOK,
		body:   jwksDocument(t, second, "key-2"),
	})}
	verifier, err := newVerifier(
		t.Context(),
		testPolicy(t),
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		func() time.Time { return now },
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)
	now = now.Add(MaxKeySetAge)

	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))
	principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
	if err != nil || principal.Subject == "" {
		t.Fatalf("Verify(stale unknown-kid recovery) = (%+v, %v), want success", principal, err)
	}
	if client.callCount() != 3 {
		t.Fatalf("provider calls = %d, want one request-driven recovery", client.callCount())
	}
}

func TestRefreshCoalescing(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	release := make(chan struct{})
	started := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status:  http.StatusOK,
		body:    jwksDocument(t, second, "key-2"),
		wait:    release,
		started: started,
	})}
	verifier := newTestVerifierWithClient(t, now, client)
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))

	var successful atomic.Int64
	var wait sync.WaitGroup
	for range 20 {
		wait.Go(func() {
			if _, err := verifier.Verify(context.Background(), token, TransportHTTP); err == nil {
				successful.Add(1)
			}
		})
	}
	<-started
	close(release)
	wait.Wait()
	if successful.Load() != 20 {
		t.Fatalf("successful verifications = %d, want 20", successful.Load())
	}
	if client.callCount() != 3 {
		t.Fatalf("provider calls = %d, want one coalesced refresh", client.callCount())
	}
}

func TestInitialTrustOutageFailsClosed(t *testing.T) {
	policy := testPolicy(t)
	client := &scriptedClient{responses: []scriptedResponse{{err: errors.New("poison outage")}}}
	_, err := newVerifier(
		t.Context(),
		policy,
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		time.Now,
		nil,
		nil,
		nil,
	)
	if err == nil || strings.Contains(err.Error(), "poison") {
		t.Fatalf("newVerifier() error = %v, want sanitized startup failure", err)
	}
}

func TestInitialTrustHonorsCancellationAndDeadline(t *testing.T) {
	policy := testPolicy(t)
	for _, testCase := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
	}{
		{
			name: "canceled",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
		},
		{
			name: "deadline",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Unix(1, 0))
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := testCase.ctx()
			defer cancel()
			client := &scriptedClient{responses: []scriptedResponse{{status: http.StatusOK, body: []byte(`{}`)}}}
			_, err := newVerifier(
				ctx,
				policy,
				func(string) (providerClient, error) {
					return providerClient{request: client, close: func() {}}, nil
				},
				time.Now,
				nil,
				nil,
				nil,
			)
			if err == nil || strings.Contains(err.Error(), testIssuer) {
				t.Fatalf("newVerifier() error = %v, want sanitized fail-closed cancellation", err)
			}
		})
	}
}

func TestRefreshCancellation(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	release := make(chan struct{})
	started := make(chan struct{})
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		status:  http.StatusOK,
		body:    jwksDocument(t, second, "key-2"),
		wait:    release,
		started: started,
	})}
	verifier := newTestVerifierWithClient(t, now, client)
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := verifier.Verify(ctx, token, TransportHTTP)
		result <- err
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify() error = %v, want context cancellation", err)
	}
	close(release)
}

func TestStaleKeySetFailsReadinessAndVerificationClosed(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	client := &scriptedClient{responses: initialResponses(t, key)}
	policy := testPolicy(t)
	verifier, err := newVerifier(
		t.Context(),
		policy,
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		func() time.Time { return now },
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)
	now = now.Add(MaxKeySetAge)

	requireKind(t, verifier.CheckReady(), KindUnavailable)
	token := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	_, err = verifier.Verify(t.Context(), token, TransportHTTP)
	requireKind(t, err, KindUnavailable)
}

func TestErrorsAndLogsRedactCredentialAndProviderContent(t *testing.T) {
	const sensitive = "sensitive-token-and-claim"
	var output strings.Builder
	log := slog.New(slog.NewTextHandler(&output, nil))
	policy := testPolicy(t)
	client := &scriptedClient{responses: []scriptedResponse{{
		status: http.StatusBadGateway,
		body:   []byte(sensitive),
		err:    errors.New(sensitive),
	}}}
	_, err := newVerifier(
		t.Context(),
		policy,
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		time.Now,
		nil,
		nil,
		log,
	)
	if err == nil {
		t.Fatal("newVerifier() error = nil, want provider failure")
	}
	if strings.Contains(err.Error(), sensitive) || strings.Contains(output.String(), sensitive) {
		t.Fatalf("sensitive provider content escaped: error=%q log=%q", err, output.String())
	}

	verifier := newTestVerifier(t, time.Unix(1_900_000_000, 0), loadTestRSAKey(t, "test-key-1.pem"))
	_, err = verifier.Verify(t.Context(), sensitive, TransportHTTP)
	if err == nil || strings.Contains(err.Error(), sensitive) {
		t.Fatalf("credential escaped through verification error: %v", err)
	}
}

func TestAuthnRedactionCoversRefreshPanicAndTelemetry(t *testing.T) {
	const poison = "poison-authn-marker-7c5e"
	now := time.Unix(1_900_000_000, 0)
	first := loadTestRSAKey(t, "test-key-1.pem")
	second := loadTestRSAKey(t, "test-key-2.pem")
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(t.Context()); err != nil {
			t.Errorf("shutdown metric provider: %v", err)
		}
	})
	var output strings.Builder
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{panic: poison})}
	verifier, err := newVerifier(
		t.Context(),
		testPolicy(t),
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		func() time.Time { return now },
		nil,
		provider,
		slog.New(slog.NewJSONHandler(&output, nil)),
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)

	claims := validClaims(now)
	claims.Subject = poison
	claims.JWTID = poison
	token := signToken(t, second, poison, "at+jwt", claims)
	_, err = verifier.Verify(t.Context(), token, TransportHTTP)
	requireKind(t, err, KindInvalid)
	if strings.Contains(err.Error(), poison) || strings.Contains(output.String(), poison) {
		t.Fatalf("refresh panic escaped through error or logs: error=%q logs=%q", err, output.String())
	}

	verifier.now = func() time.Time { panic(poison) }
	_, err = verifier.Verify(t.Context(), token, TransportHTTP)
	requireKind(t, err, KindUnavailable)
	if strings.Contains(err.Error(), poison) || strings.Contains(output.String(), poison) {
		t.Fatalf("verification panic escaped through error or logs: error=%q logs=%q", err, output.String())
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &metrics); err != nil {
		t.Fatalf("collect authn metrics: %v", err)
	}
	if encoded := fmt.Sprintf("%#v", metrics); strings.Contains(encoded, poison) {
		t.Fatalf("authn metrics contain poison marker: %s", encoded)
	}
}

func TestBearerCredential(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		want     string
		wantKind Kind
	}{
		{name: "valid", values: []string{"bEaReR abc"}, want: "abc"},
		{name: "missing", wantKind: KindMissing},
		{name: "duplicate", values: []string{"Bearer a", "Bearer b"}, wantKind: KindMalformed},
		{name: "alternate", values: []string{"Basic abc"}, wantKind: KindMalformed},
		{name: "extra whitespace", values: []string{"Bearer  abc"}, wantKind: KindMalformed},
		{name: "comma joined", values: []string{"Bearer a,Bearer b"}, wantKind: KindMalformed},
		{name: "oversized", values: []string{"Bearer " + strings.Repeat("a", MaxTokenBytes+1)}, wantKind: KindOversize},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := bearerToken(testCase.values)
			requireKind(t, err, testCase.wantKind)
			if got != testCase.want {
				t.Fatalf("bearerToken() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestHTTPAuthnBoundaryTransportAndIdentity(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	verifier := newTestVerifier(t, now, key)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://service.example/protected", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Authorization", "Bearer "+signToken(t, key, "key-1", "at+jwt", validClaims(now)))
	input := &openapi3filter.AuthenticationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request},
	}
	principal, err := verifier.ResolveHTTP(t.Context(), input)
	if err != nil || principal.Subject != "opaque-subject" {
		t.Fatalf("ResolveHTTP() = (%+v, %v), want subject", principal, err)
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("handler-visible Authorization header was retained")
	}

	request.RemoteAddr = "198.51.100.10:1234"
	_, err = verifier.ResolveHTTP(t.Context(), input)
	requireKind(t, err, KindUntrustedTransport)
}

func TestHTTPAuthnBoundaryCarrierClasses(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	trustedKey := loadTestRSAKey(t, "test-key-1.pem")
	otherKey := loadTestRSAKey(t, "test-key-2.pem")
	valid := signToken(t, trustedKey, "key-1", "at+jwt", validClaims(now))
	invalid := signToken(t, otherKey, "key-1", "at+jwt", validClaims(now))
	staleClaims := validClaims(now)
	staleClaims.Expires = now.Add(30 * time.Minute).Unix()
	validAfterStaleCutoff := signToken(t, trustedKey, "key-1", "at+jwt", staleClaims)

	tests := []struct {
		name        string
		headers     []string
		remoteAddr  string
		stale       bool
		wantKind    Kind
		wantSubject string
	}{
		{
			name:        "valid",
			headers:     []string{"Bearer " + valid},
			wantSubject: "opaque-subject",
		},
		{
			name:     "missing",
			wantKind: KindMissing,
		},
		{
			name:     "malformed",
			headers:  []string{"Basic opaque-token"},
			wantKind: KindMalformed,
		},
		{
			name:     "duplicate",
			headers:  []string{"Bearer one", "Bearer two"},
			wantKind: KindMalformed,
		},
		{
			name:     "oversize",
			headers:  []string{"Bearer " + strings.Repeat("x", MaxTokenBytes+1)},
			wantKind: KindOversize,
		},
		{
			name:     "invalid",
			headers:  []string{"Bearer " + invalid},
			wantKind: KindInvalid,
		},
		{
			name:       "untrusted transport",
			headers:    []string{"Bearer " + valid},
			remoteAddr: "198.51.100.10:1234",
			wantKind:   KindUntrustedTransport,
		},
		{
			name:     "stale trust",
			headers:  []string{"Bearer " + validAfterStaleCutoff},
			stale:    true,
			wantKind: KindUnavailable,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			client := &scriptedClient{responses: initialResponses(t, trustedKey)}
			verifier := newTestVerifierWithClient(t, now, client)
			if testCase.stale {
				verifier.now = func() time.Time { return now.Add(MaxKeySetAge) }
			}

			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"https://service.example/protected",
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			request.RemoteAddr = testCase.remoteAddr
			if request.RemoteAddr == "" {
				request.RemoteAddr = "127.0.0.1:1234"
			}
			request.Header.Set("X-Forwarded-Proto", "https")
			for _, value := range testCase.headers {
				request.Header.Add("Authorization", value)
			}
			input := &openapi3filter.AuthenticationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request},
			}

			principal, err := verifier.ResolveHTTP(t.Context(), input)
			requireKind(t, err, testCase.wantKind)
			if principal.Subject != testCase.wantSubject {
				t.Fatalf("principal subject = %q, want %q", principal.Subject, testCase.wantSubject)
			}
			if testCase.wantKind != KindUntrustedTransport &&
				request.Header.Get("Authorization") != "" {
				t.Fatal("parsed Authorization header was retained")
			}
			if (testCase.wantKind == KindMalformed || testCase.wantKind == KindOversize) &&
				client.callCount() != 2 {
				t.Fatalf("provider calls = %d, want only initial Discovery and JWKS", client.callCount())
			}
		})
	}
}

func TestDocumentedTokenExample(t *testing.T) {
	document, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "authentication.md"))
	if err != nil {
		t.Fatalf("read authentication guide: %v", err)
	}
	guide := string(document)
	for _, required := range []string{
		"AUTHN=oidc-jwt",
		"APP__AUTHN__ISSUER=https://",
		"APP__AUTHN__AUDIENCE=",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS=",
		"RS256 access token",
		"native gRPC is enabled with this profile, its server transport must be TLS",
		"There is no bypass, fake principal mode, or accept-all switch",
	} {
		if !strings.Contains(guide, required) {
			t.Fatalf("authentication guide is missing %q", required)
		}
	}

	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	verifier := newTestVerifier(t, now, key)
	token := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
	if err != nil || principal.Subject != "opaque-subject" {
		t.Fatalf("documented signed-token flow = (%+v, %v), want valid opaque principal", principal, err)
	}
}

func FuzzStrictToken(f *testing.F) {
	f.Add("not-a-token")
	f.Add("e30.e30.signature")
	f.Fuzz(func(t *testing.T, token string) {
		policy, err := NewPolicy(testIssuer, testAudience, "127.0.0.0/8")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = parseToken(token, policy, time.Unix(1_900_000_000, 0))
	})
}

func newTestVerifier(t *testing.T, now time.Time, key *rsa.PrivateKey) *Verifier {
	t.Helper()
	client := &scriptedClient{responses: initialResponses(t, key)}
	return newTestVerifierWithClient(t, now, client)
}

func newTestVerifierWithClient(t *testing.T, now time.Time, client *scriptedClient) *Verifier {
	t.Helper()
	policy := testPolicy(t)
	verifier, err := newVerifier(
		t.Context(),
		policy,
		func(string) (providerClient, error) {
			return providerClient{request: client, close: func() {}}, nil
		},
		func() time.Time { return now },
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("newVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)
	return verifier
}

func testPolicy(t *testing.T) Policy {
	t.Helper()
	policy, err := NewPolicy(testIssuer, testAudience, "127.0.0.0/8,::1/128")
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	return policy
}

func initialResponses(t *testing.T, key *rsa.PrivateKey) []scriptedResponse {
	t.Helper()
	discovery, err := json.Marshal(discoveryDocument{Issuer: testIssuer, JWKSURI: testJWKSURI})
	if err != nil {
		t.Fatal(err)
	}
	return []scriptedResponse{
		{status: http.StatusOK, body: discovery},
		{status: http.StatusOK, body: jwksDocument(t, key, "key-1")},
	}
}

func jwksDocument(t *testing.T, key *rsa.PrivateKey, keyID string) []byte {
	t.Helper()
	encoded, err := publicJWK(&key.PublicKey, keyID)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(`{"keys":[` + string(encoded) + `]}`)
}

func publicJWK(key *rsa.PublicKey, keyID string) ([]byte, error) {
	if key == nil {
		return nil, errors.New("nil RSA key")
	}
	exponent := big.NewInt(int64(key.E)).Bytes()
	encoded, err := json.Marshal(map[string]string{
		"kty": "RSA",
		"kid": keyID,
		"alg": AllowedAlgorithm,
		"use": "sig",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exponent),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal public JWK: %w", err)
	}
	return encoded, nil
}

func loadTestRSAKey(t *testing.T, name string) *rsa.PrivateKey {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixed test key: %v", err)
	}
	block, rest := pem.Decode(encoded)
	if block == nil || block.Type != "PRIVATE KEY" || len(rest) != 0 {
		t.Fatalf("decode fixed test key %q", name)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("parse fixed test key: %v", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() != 2048 {
		t.Fatalf("fixed test key %q is not RSA-2048", name)
	}
	return key
}

func validClaims(now time.Time) tokenClaims {
	return tokenClaims{
		Issuer:   testIssuer,
		Subject:  "opaque-subject",
		Audience: []string{"another", testAudience},
		ClientID: "client-1",
		JWTID:    "token-1",
		Expires:  now.Add(5 * time.Minute).Unix(),
		IssuedAt: now.Unix(),
		Scope:    []string{"admin"},
		Roles:    []string{"owner"},
	}
}

func mutateClaims(input tokenClaims, change func(*tokenClaims)) tokenClaims {
	change(&input)
	return input
}

func signToken(t *testing.T, key *rsa.PrivateKey, keyID, typ string, claims tokenClaims) string {
	t.Helper()
	return signTokenWithHeader(t, key, map[jose.HeaderKey]any{"typ": typ, "kid": keyID}, claims)
}

func signTokenWithHeader(
	t *testing.T,
	key *rsa.PrivateKey,
	headers map[jose.HeaderKey]any,
	claims tokenClaims,
) string {
	t.Helper()
	options := &jose.SignerOptions{}
	for name, value := range headers {
		options.WithHeader(name, value)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return signPayloadWithOptions(t, key, options, payload)
}

func signPayload(t *testing.T, key *rsa.PrivateKey, payload []byte) string {
	t.Helper()
	options := (&jose.SignerOptions{}).WithHeader("typ", "at+jwt").WithHeader("kid", "key-1")
	return signPayloadWithOptions(t, key, options, payload)
}

func signPayloadWithOptions(
	t *testing.T,
	key *rsa.PrivateKey,
	options *jose.SignerOptions,
	payload []byte,
) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, options)
	if err != nil {
		t.Fatalf("jose.NewSigner() error = %v", err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	compact, err := signed.CompactSerialize()
	if err != nil {
		t.Fatalf("CompactSerialize() error = %v", err)
	}
	return compact
}

func requireKind(t *testing.T, err error, want Kind) {
	t.Helper()
	if want == 0 {
		if err != nil {
			t.Fatalf("error = %v, want nil", err)
		}
		return
	}
	got, ok := KindOf(err)
	if !ok || got != want {
		t.Fatalf("error = %v, kind = (%v, %v), want %v", err, got, ok, want)
	}
}
