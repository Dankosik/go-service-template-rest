package oidcjwt

// Proof for token.go: the compact-token shape this service accepts, the claim
// terms it requires, its time policy, and the credential header both transports
// share.

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"maps"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

func TestVerify_Claims(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	otherKey := loadTestRSAKey(t, testRotatedKey)

	// Each case starts from validClaims and states only what it changes: mutate
	// is the one claim term under test, key defaults to the trusted signing key,
	// and typ defaults to "at+jwt". A claim requirement added to
	// parseAccessTokenClaims is one accepting row and one rejecting row here.
	tests := []struct {
		name     string
		key      *rsa.PrivateKey
		typ      string
		mutate   func(*tokenClaims)
		wantKind Kind
	}{
		{name: "valid token"},
		{name: "media type typ", typ: "APPLICATION/AT+JWT"},
		{name: "wrong typ", typ: "JWT", wantKind: KindInvalid},
		{name: "invalid signature", key: otherKey, wantKind: KindInvalid},
		{
			name:   "valid scalar audience",
			mutate: func(c *tokenClaims) { c.Audience = testAudience },
		},
		{
			name:     "wrong issuer",
			mutate:   func(c *tokenClaims) { c.Issuer = "https://wrong.example" },
			wantKind: KindInvalid,
		},
		{
			name:     "wrong audience",
			mutate:   func(c *tokenClaims) { c.Audience = "other" },
			wantKind: KindInvalid,
		},
		{
			name:     "empty audience",
			mutate:   func(c *tokenClaims) { c.Audience = "" },
			wantKind: KindInvalid,
		},
		{
			name:     "missing audience",
			mutate:   func(c *tokenClaims) { c.Audience = nil },
			wantKind: KindInvalid,
		},
		{
			name:     "duplicate audience",
			mutate:   func(c *tokenClaims) { c.Audience = []string{testAudience, testAudience} },
			wantKind: KindInvalid,
		},
		{
			name:     "expired",
			mutate:   func(c *tokenClaims) { c.Expires = now.Add(-ClockSkew).Unix() },
			wantKind: KindInvalid,
		},
		{
			name: "premature",
			mutate: func(c *tokenClaims) {
				notBefore := now.Add(ClockSkew + time.Second).Unix()
				c.NotBefore = &notBefore
			},
			wantKind: KindInvalid,
		},
		{
			name:     "future issued",
			mutate:   func(c *tokenClaims) { c.IssuedAt = now.Add(ClockSkew + time.Second).Unix() },
			wantKind: KindInvalid,
		},
		{
			name:     "missing subject",
			mutate:   func(c *tokenClaims) { c.Subject = "" },
			wantKind: KindInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			signingKey := key
			if testCase.key != nil {
				signingKey = testCase.key
			}
			typ := testCase.typ
			if typ == "" {
				typ = "at+jwt"
			}
			claims := validClaims(now)
			if testCase.mutate != nil {
				testCase.mutate(&claims)
			}

			verifier := newTestVerifier(t, key)
			token := signToken(t, signingKey, "key-1", typ, claims)
			principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
			requireKind(t, err, testCase.wantKind)
			if testCase.wantKind == 0 {
				if principal.Subject != claims.Subject || len(principal.Scopes) != 0 {
					t.Fatalf("principal = %+v, want opaque subject and no scopes", principal)
				}
			}
		})
	}
}

func TestVerify_Serialization(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	client := &scriptedClient{responses: initialResponses(t, key)}
	verifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: client})
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
	criticalHeaderToken := signTokenWithHeader(
		t, key,
		map[jose.HeaderKey]any{"typ": "at+jwt", "kid": "key-1", "crit": []string{"exp"}},
		validClaims(now),
	)
	remoteJWKToken := signTokenWithHeader(
		t, key,
		map[jose.HeaderKey]any{"typ": "at+jwt", "kid": "key-1", "jku": "https://attacker.example/jwks"},
		validClaims(now),
	)
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
		"critical header":       criticalHeaderToken,
		"remote jwk header":     remoteJWKToken,
		"duplicate payload":     signPayload(t, key, []byte(`{"iss":"`+testIssuer+`","iss":"`+testIssuer+`"}`)),
		"trailing payload data": signPayload(t, key, append(validPayload, []byte(`{}`)...)),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := verifier.Verify(t.Context(), token, TransportHTTP)
			requireKind(t, err, KindInvalid)
			requireProviderCalls(t, client, 0, "after a locally invalid token")
		})
	}
}

func TestVerifyRejectsExplicitNullNotBefore(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	claims := claimsMap(t, validClaims(now))
	claims["nbf"] = nil
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}

	verifier := newTestVerifier(t, key)
	_, err = verifier.Verify(t.Context(), signPayload(t, key, payload), TransportHTTP)
	requireKind(t, err, KindInvalid)
}

func TestVerify_TimePolicy(t *testing.T) {
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	base := claimsMap(t, validClaims(now))
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
			verifier := newTestVerifier(t, key)
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

func FuzzStrictToken(f *testing.F) {
	f.Add("not-a-token")
	f.Add("e30.e30.signature")
	f.Fuzz(func(t *testing.T, token string) {
		policy, err := NewPolicy(PolicyInput{
			Issuer:            testIssuer,
			Audience:          testAudience,
			TrustedProxyCIDRs: "127.0.0.0/8",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = parseToken(token, policy, testNow)
	})
}
