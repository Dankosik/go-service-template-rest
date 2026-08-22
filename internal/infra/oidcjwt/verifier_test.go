package oidcjwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/golang-jwt/jwt/v5"
)

func TestResourceServerProfileAcceptsMainstreamClientIdentityClaims(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)

	for _, testCase := range []struct {
		name     string
		typ      string
		claims   tokenClaims
		clientID string
	}{
		{name: "subject only", typ: "JWT", claims: withoutClientID(validClaims(testNow))},
		{name: "RFC client_id", typ: "at+jwt", claims: validClaims(testNow), clientID: "client-1"},
		{name: "Auth0 azp", typ: "JWT", claims: withClientAlias(validClaims(testNow), "azp"), clientID: "client-1"},
		{name: "Entra appid", claims: withClientAlias(validClaims(testNow), "appid"), clientID: "client-1"},
		{name: "Okta cid", claims: withClientAlias(validClaims(testNow), "cid"), clientID: "client-1"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			token := signToken(t, key, "key-1", testCase.typ, testCase.claims)
			verified, err := verifier.Verify(t.Context(), token)
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if verified.Principal.Subject != "subject-1" || verified.Principal.ClientID != testCase.clientID {
				t.Fatalf("principal = %+v", verified.Principal)
			}
		})
	}

	clientOnly := validClaims(testNow)
	clientOnly.Subject = ""
	verified, err := verifier.Verify(
		t.Context(),
		signToken(t, key, "key-1", "JWT", clientOnly),
	)
	if err != nil || verified.Principal.Subject != "" || verified.Principal.ClientID != "client-1" {
		t.Fatalf("client-only principal = %+v, error = %v", verified.Principal, err)
	}
}

func TestRFC9068ProfileIsExplicit(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
	strict := newVerifier(
		testPolicyWithProfile(t, "rfc9068"),
		staticKeyFunc(key),
		func() time.Time { return testNow },
		newJWKSMetrics(nil),
		nil,
		nil,
	)

	valid := validClaims(testNow)
	if _, err := strict.Verify(t.Context(), signToken(t, key, "key-1", "at+jwt", valid)); err != nil {
		t.Fatalf("strict valid token error = %v", err)
	}
	for _, testCase := range []struct {
		name   string
		typ    string
		claims tokenClaims
	}{
		{name: "JWT type", typ: "JWT", claims: valid},
		{name: "missing client_id", typ: "at+jwt", claims: withoutClientID(valid)},
		{name: "missing jti", typ: "at+jwt", claims: func() tokenClaims { c := valid; c.JWTID = ""; return c }()},
		{name: "missing iat", typ: "at+jwt", claims: func() tokenClaims { c := valid; c.IssuedAt = nil; return c }()},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := strict.Verify(t.Context(), signToken(t, key, "key-1", testCase.typ, testCase.claims))
			requireKind(t, err, bearerauthn.KindInvalid)
		})
	}
}

func TestVerifierRejectsInvalidTrustAndIdentity(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)

	for _, testCase := range []struct {
		name   string
		claims tokenClaims
	}{
		{name: "wrong issuer", claims: func() tokenClaims { c := validClaims(testNow); c.Issuer = "https://other.example"; return c }()},
		{name: "wrong audience", claims: func() tokenClaims { c := validClaims(testNow); c.Audience = "other"; return c }()},
		{name: "expired", claims: func() tokenClaims {
			c := validClaims(testNow)
			expired := testNow.Add(-time.Minute)
			c.ExpiresAt = &expired
			return c
		}()},
		{name: "anonymous", claims: func() tokenClaims { c := withoutClientID(validClaims(testNow)); c.Subject = ""; return c }()},
		{name: "conflicting clients", claims: func() tokenClaims { c := validClaims(testNow); c.AuthorizedParty = "other"; return c }()},
		{name: "missing expiry", claims: func() tokenClaims { c := validClaims(testNow); c.ExpiresAt = nil; return c }()},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := verifier.Verify(t.Context(), signToken(t, key, "key-1", "JWT", testCase.claims))
			requireKind(t, err, bearerauthn.KindInvalid)
		})
	}

	wrongKey := loadTestRSAKey(t, "test-key-2.pem")
	if _, err := verifier.Verify(
		t.Context(),
		signToken(t, wrongKey, "key-1", "JWT", validClaims(testNow)),
	); err == nil {
		t.Fatal("token signed by an untrusted key was accepted")
	}

	hmac := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": testIssuer,
		"sub": "subject-1",
		"aud": testAudience,
		"exp": testNow.Add(time.Hour).Unix(),
	})
	hmac.Header["kid"] = "key-1"
	signedHMAC, err := hmac.SignedString([]byte("not-an-issuer-key"))
	if err != nil {
		t.Fatalf("sign HMAC control: %v", err)
	}
	if _, err := verifier.Verify(t.Context(), signedHMAC); err == nil {
		t.Fatal("token using an unconfigured algorithm was accepted")
	}

	encryptionKeyVerifier := newTestVerifierWithUse(t, key, "enc")
	if _, err := encryptionKeyVerifier.Verify(
		t.Context(),
		signToken(t, key, "key-1", "JWT", validClaims(testNow)),
	); err == nil {
		t.Fatal("JWK marked for encryption was accepted for signature verification")
	}
}

func TestJWKSRefreshFailureIsUnavailable(t *testing.T) {
	t.Parallel()
	verifier := newVerifier(
		testPolicy(t),
		func(ctx context.Context) jwt.Keyfunc {
			return func(*jwt.Token) (any, error) {
				failureState, ok := ctx.Value(refreshFailureKey).(*refreshFailure)
				if !ok {
					return nil, errors.New("refresh failure state is missing")
				}
				failureState.failed.Store(true)
				return nil, errors.New("provider failed")
			}
		},
		func() time.Time { return testNow },
		newJWKSMetrics(nil),
		nil,
		nil,
	)
	key := loadTestRSAKey(t, testSigningKey)
	_, err := verifier.Verify(t.Context(), signToken(t, key, "unknown", "JWT", validClaims(testNow)))
	requireKind(t, err, bearerauthn.KindUnavailable)
}

func TestVerifierCloseIsIdempotent(t *testing.T) {
	t.Parallel()
	var cancels, closes int
	verifier := newVerifier(
		testPolicy(t),
		staticKeyFunc(loadTestRSAKey(t, testSigningKey)),
		func() time.Time { return testNow },
		newJWKSMetrics(nil),
		func() { cancels++ },
		func() { closes++ },
	)
	verifier.Close()
	verifier.Close()
	if cancels != 1 || closes != 1 {
		t.Fatalf("Close() calls = cancel %d, close %d; want one each", cancels, closes)
	}
}

func staticKeyFunc(key *rsa.PrivateKey) func(context.Context) jwt.Keyfunc {
	return func(context.Context) jwt.Keyfunc {
		return func(*jwt.Token) (any, error) { return &key.PublicKey, nil }
	}
}

func withoutClientID(claims tokenClaims) tokenClaims {
	claims.ClientID = ""
	return claims
}

func withClientAlias(claims tokenClaims, alias string) tokenClaims {
	claims.ClientID = ""
	switch alias {
	case "azp":
		claims.AuthorizedParty = "client-1"
	case "appid":
		claims.ApplicationID = "client-1"
	case "cid":
		claims.OktaClientID = "client-1"
	}
	return claims
}
