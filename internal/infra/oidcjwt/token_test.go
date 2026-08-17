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
	t.Parallel()
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
		{
			name:     "blank client ID",
			mutate:   func(c *tokenClaims) { c.ClientID = "" },
			wantKind: KindInvalid,
		},
		{
			name:     "blank JWT ID",
			mutate:   func(c *tokenClaims) { c.JWTID = "" },
			wantKind: KindInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
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
			principal, err := verifier.verify(t.Context(), token, transportHTTP)
			requireKind(t, err, testCase.wantKind)
			if testCase.wantKind == 0 {
				if principal.Issuer != claims.Issuer ||
					principal.Subject != claims.Subject ||
					principal.ClientID != claims.ClientID ||
					len(principal.Scopes) != 0 {
					t.Fatalf("principal = %+v, want issuer, subject, client ID, and no scopes", principal)
				}
			}
		})
	}
}

func TestVerify_Serialization(t *testing.T) {
	t.Parallel()
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
	missingClientID := claimsMap(t, validClaims(now))
	delete(missingClientID, "client_id")
	missingClientIDPayload, err := json.Marshal(missingClientID)
	if err != nil {
		t.Fatal(err)
	}
	missingJWTID := claimsMap(t, validClaims(now))
	delete(missingJWTID, "jti")
	missingJWTIDPayload, err := json.Marshal(missingJWTID)
	if err != nil {
		t.Fatal(err)
	}
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
		"missing client ID":     signPayload(t, key, missingClientIDPayload),
		"missing JWT ID":        signPayload(t, key, missingJWTIDPayload),
		"trailing payload data": signPayload(t, key, append(validPayload, []byte(`{}`)...)),
	}
	for name, token := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := verifier.verify(t.Context(), token, transportHTTP)
			requireKind(t, err, KindInvalid)
			requireProviderCalls(t, client, 0, "after a locally invalid token")
		})
	}
}

func TestVerifyRejectsExplicitNullNotBefore(t *testing.T) {
	t.Parallel()
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	claims := claimsMap(t, validClaims(now))
	claims["nbf"] = nil
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}

	verifier := newTestVerifier(t, key)
	_, err = verifier.verify(t.Context(), signPayload(t, key, payload), transportHTTP)
	requireKind(t, err, KindInvalid)
}

func TestVerify_TimePolicy(t *testing.T) {
	t.Parallel()
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
				claims["iat"] = now.Add(-time.Minute).Unix()
				claims["exp"] = now.Add(-ClockSkew + time.Second).Unix()
			},
		},
		{
			name: "expiration at skew boundary is rejected",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(-time.Minute).Unix()
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
			name: "maximum token lifetime is accepted",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Unix()
				claims["exp"] = now.Add(MaxTokenLifetime).Unix()
			},
		},
		{
			// Its own category, because it names an issuer configured for longer
			// tokens rather than a credential a caller got wrong. KindLifetime owns
			// why, and TestHTTPAuthnBoundary holds it to the same answer a caller
			// receives for KindInvalid.
			name: "token lifetime beyond maximum is its own category",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Unix()
				claims["exp"] = now.Add(MaxTokenLifetime + time.Second).Unix()
			},
			wantKind: KindLifetime,
		},
		{
			name: "expiration must follow issued-at",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(ClockSkew).Unix()
				claims["exp"] = now.Add(ClockSkew).Unix()
			},
			wantKind: KindInvalid,
		},
		{
			name: "not-before must precede expiration",
			mutate: func(claims map[string]any) {
				claims["iat"] = now.Add(-time.Minute).Unix()
				claims["exp"] = now.Add(20 * time.Second).Unix()
				claims["nbf"] = now.Add(20 * time.Second).Unix()
			},
			wantKind: KindInvalid,
		},
		{
			name: "fractional expiration is accepted",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1900000000.5")
			},
		},
		{
			name: "exponent expiration is accepted",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1.9000003e9")
			},
		},
		{
			name: "expanding exponent is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1e1000000")
			},
			wantKind: KindInvalid,
		},
		{
			name: "collapsing exponent is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1e-1000000")
			},
			wantKind: KindInvalid,
		},
		{
			name: "unreadable exponent is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1e99999999999999999999")
			},
			wantKind: KindInvalid,
		},
		{
			name: "over-long numeric date literal is rejected",
			mutate: func(claims map[string]any) {
				claims["exp"] = json.Number("1900000000." + strings.Repeat("5", maxNumericDateLiteral))
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
			name: "extreme numeric dates cannot overflow the lifetime check",
			mutate: func(claims map[string]any) {
				claims["iat"] = json.Number("-9223372036854775808")
				claims["exp"] = json.Number("9223372036854775807")
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
		{
			name: "missing issued-at is rejected",
			mutate: func(claims map[string]any) {
				delete(claims, "iat")
			},
			wantKind: KindInvalid,
		},
		{
			name: "blank issued-at is rejected",
			mutate: func(claims map[string]any) {
				claims["iat"] = ""
			},
			wantKind: KindInvalid,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			claims := make(map[string]any, len(base))
			maps.Copy(claims, base)
			testCase.mutate(claims)
			payload, err := json.Marshal(claims)
			if err != nil {
				t.Fatalf("marshal claims: %v", err)
			}
			verifier := newTestVerifier(t, key)
			principal, err := verifier.verify(
				t.Context(),
				signPayload(t, key, payload),
				transportHTTP,
			)
			requireKind(t, err, testCase.wantKind)
			if testCase.wantKind == 0 && principal.Subject != "opaque-subject" {
				t.Fatalf("principal = %+v, want opaque subject", principal)
			}
		})
	}
}

// TestNumericDateBudgetSurvivesAnExpandingExponent is the regression for a
// refusal that used to arrive after the cost.
//
// Every claim is decided before jose checks a signature, so what one costs to
// parse is what an unauthenticated caller can make this service spend. math/big
// accepts a base-10 exponent up to a million, so "1e1000000" — nine bytes, in a
// token anyone can mint precisely because nothing has verified it yet — used to
// expand into a million-digit number and only then be refused for naming no
// representable instant, at several MiB and tens of milliseconds for each of
// exp, iat, and nbf.
//
// The refusal is not the property under test, because the old code refused it
// too and the table above already covers the answer. The budget is, so this
// measures what parsing allocated.
func TestNumericDateBudgetSurvivesAnExpandingExponent(t *testing.T) {
	t.Parallel()
	// Far above the honest cost of parsing a token this size and far below one
	// expansion, so neither has to be tracked precisely for this to stay true.
	const budgetBytes = 64 << 10

	key := loadTestRSAKey(t, testSigningKey)
	claims := claimsMap(t, validClaims(testNow))
	for _, claim := range []string{"exp", "iat", "nbf"} {
		claims[claim] = json.Number("1e1000000")
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	poisoned := signPayload(t, key, payload)
	policy := testPolicy(t)

	measured := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		b.ReportAllocs()
		for b.Loop() {
			if _, parseErr := parseToken(poisoned, policy, testNow); parseErr == nil {
				b.Fatal("a token whose NumericDate claims name no instant parsed successfully")
			}
		}
	})
	if allocated := measured.AllocedBytesPerOp(); allocated > budgetBytes {
		t.Fatalf(
			"parseToken allocated %d bytes for an unsigned token carrying three expanding exponents, want at most %d",
			allocated,
			budgetBytes,
		)
	}
}

func TestBearerCredential(t *testing.T) {
	t.Parallel()
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
		{name: "multiple scheme spaces", values: []string{"Bearer  abc"}, want: "abc"},
		{name: "tab separator", values: []string{"Bearer\tabc"}, wantKind: KindMalformed},
		{name: "leading whitespace", values: []string{" Bearer abc"}, wantKind: KindMalformed},
		{name: "trailing whitespace", values: []string{"Bearer abc "}, wantKind: KindMalformed},
		{name: "comma joined", values: []string{"Bearer a,Bearer b"}, wantKind: KindMalformed},
		{name: "oversized", values: []string{"Bearer " + strings.Repeat("a", MaxTokenBytes+1)}, wantKind: KindOversize},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
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
