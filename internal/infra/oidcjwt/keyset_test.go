package oidcjwt

// Proof for keyset.go: which JWK entries this service will verify with, and that
// a set it refuses replaces nothing.

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
)

func TestKeySetAdmission(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
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
			t.Parallel()
			var jwk map[string]any
			if err := json.Unmarshal(valid, &jwk); err != nil {
				t.Fatal(err)
			}
			candidate, err := json.Marshal(map[string]any{"keys": []any{testCase.mutate(jwk)}})
			if err != nil {
				t.Fatal(err)
			}
			keySet, err := parseKeySet(candidate, testNow)
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

func TestRejectedJWKSIsAtomic(t *testing.T) {
	t.Parallel()
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	// Two different keys claiming one key id: the set is ambiguous, so no part
	// of it may be installed.
	duplicate := jwksDocumentOf(t, jwkEntry{key: second, keyID: "key-2"}, jwkEntry{key: first, keyID: "key-2"})

	for _, candidate := range [][]byte{
		[]byte(`{"keys":[]}`),
		duplicate,
		[]byte(`{"keys":[{"kid":"bad","kty":"RSA","alg":"HS256","use":"sig","n":"AQ","e":"Aw"}]}`),
	} {
		client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
			status: http.StatusOK,
			body:   candidate,
		})}
		verifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: client})
		unknown := signToken(t, second, "key-2", "at+jwt", validClaims(now))
		_, err := verifier.verify(t.Context(), unknown, transportHTTP)
		requireKind(t, err, KindInvalid)

		known := signToken(t, first, "key-1", "at+jwt", validClaims(now))
		principal, err := verifier.verify(t.Context(), known, transportHTTP)
		if err != nil || principal.Subject == "" {
			t.Fatalf("Verify(known token after invalid refresh) = (%+v, %v), want success", principal, err)
		}
	}
}
