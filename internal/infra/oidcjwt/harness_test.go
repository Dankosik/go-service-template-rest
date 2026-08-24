package oidcjwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer     = "https://issuer.example.com"
	testAudience   = "https://api.example.com"
	testSigningKey = "test-key-1.pem"
)

var testNow = time.Unix(1_900_000_000, 0)

type tokenClaims struct {
	Issuer          string
	Subject         string
	Audience        string
	ClientID        string
	AuthorizedParty string
	ApplicationID   string
	OktaClientID    string
	JWTID           string
	IssuedAt        *time.Time
	ExpiresAt       *time.Time
	NotBefore       *time.Time
}

func validClaims(now time.Time) tokenClaims {
	expires := now.Add(time.Hour)
	return tokenClaims{
		Issuer:    testIssuer,
		Subject:   "subject-1",
		Audience:  testAudience,
		ClientID:  "client-1",
		JWTID:     "token-1",
		IssuedAt:  &now,
		ExpiresAt: &expires,
	}
}

func testPolicy(t *testing.T) Policy {
	t.Helper()
	return testPolicyWithProfile(t, "resource-server")
}

func testPolicyWithProfile(t *testing.T, profile string) Policy {
	t.Helper()
	policy, err := NewPolicy(PolicyInput{Issuer: testIssuer, Audience: testAudience, TokenProfile: profile})
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	return policy
}

func newTestVerifier(t *testing.T, key *rsa.PrivateKey) *Verifier {
	t.Helper()
	return newTestVerifierWithUse(t, key, "sig")
}

func newTestVerifierWithUse(t *testing.T, key *rsa.PrivateKey, use string) *Verifier {
	t.Helper()
	keys, err := keyfunc.NewJWKSetJSON(testJWKSet(t, &key.PublicKey, "key-1", use))
	if err != nil {
		t.Fatalf("keyfunc.NewJWKSetJSON() error = %v", err)
	}
	signingKeys, err := keyfunc.New(keyfunc.Options{
		Ctx:          context.Background(),
		Storage:      keys.Storage(),
		UseWhitelist: []jwkset.USE{"", jwkset.UseSig},
	})
	if err != nil {
		t.Fatalf("signingKeyFunc() error = %v", err)
	}
	return newVerifier(
		testPolicy(t),
		signingKeys.KeyfuncCtx,
		func() time.Time { return testNow },
		nil,
		nil,
	)
}

func testJWKSet(t *testing.T, key *rsa.PublicKey, keyID, use string) []byte {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA",
			"use": use,
			"alg": AllowedAlgorithm,
			"kid": keyID,
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}},
	})
	if err != nil {
		t.Fatalf("marshal test JWKS: %v", err)
	}
	return encoded
}

func loadTestRSAKey(t *testing.T, name string) *rsa.PrivateKey {
	t.Helper()
	encoded, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read test key: %v", err)
	}
	block, _ := pem.Decode(encoded)
	if block == nil {
		t.Fatal("decode test key: no PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		pkcs1, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			t.Fatalf("parse test key: %v", err)
		}
		return pkcs1
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("test key type = %T, want *rsa.PrivateKey", key)
	}
	return rsaKey
}

func signToken(t *testing.T, key *rsa.PrivateKey, keyID, typ string, claims tokenClaims) string {
	t.Helper()
	values := jwt.MapClaims{
		"iss": claims.Issuer,
		"sub": claims.Subject,
		"aud": claims.Audience,
	}
	for name, value := range map[string]string{
		"client_id": claims.ClientID,
		"azp":       claims.AuthorizedParty,
		"appid":     claims.ApplicationID,
		"cid":       claims.OktaClientID,
		"jti":       claims.JWTID,
	} {
		if value != "" {
			values[name] = value
		}
	}
	if claims.IssuedAt != nil {
		values["iat"] = claims.IssuedAt.Unix()
	}
	if claims.ExpiresAt != nil {
		values["exp"] = claims.ExpiresAt.Unix()
	}
	if claims.NotBefore != nil {
		values["nbf"] = claims.NotBefore.Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, values)
	if keyID != "" {
		token.Header["kid"] = keyID
	}
	if typ != "" {
		token.Header["typ"] = typ
	}
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func requireKind(t *testing.T, err error, want bearerauthn.Kind) {
	t.Helper()
	got, ok := bearerauthn.KindOf(err)
	if !ok || got != want {
		t.Fatalf("KindOf(%v) = %v, %v; want %v, true", err, got, ok, want)
	}
}
