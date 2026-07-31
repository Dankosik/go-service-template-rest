package oidcjwt

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

type keySet struct {
	fetchedAt time.Time
	keys      map[string]*rsa.PublicKey
}

type rawJWKSet struct {
	Keys []json.RawMessage `json:"keys"`
}

type rawJWKMetadata struct {
	KeyID  string   `json:"kid"`
	Alg    string   `json:"alg"`
	Use    string   `json:"use"`
	KeyOps []string `json:"key_ops"`
	N      string   `json:"n"`
	E      string   `json:"e"`
}

func parseKeySet(data []byte, fetchedAt time.Time) (*keySet, error) {
	var raw rawJWKSet
	if err := strictUnmarshal(data, &raw); err != nil {
		return nil, errors.New("invalid JWKS")
	}
	if len(raw.Keys) == 0 || len(raw.Keys) > MaxJWKEntries {
		return nil, errors.New("invalid JWKS")
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, encoded := range raw.Keys {
		var metadata rawJWKMetadata
		if err := strictUnmarshal(encoded, &metadata); err != nil {
			return nil, errors.New("invalid JWKS")
		}
		var jwk jose.JSONWebKey
		if err := strictUnmarshal(encoded, &jwk); err != nil {
			return nil, errors.New("invalid JWKS")
		}

		publicKey, compatible := compatibleRSAKey(metadata, jwk)
		if !compatible {
			continue
		}
		if _, exists := keys[metadata.KeyID]; exists {
			return nil, errors.New("ambiguous JWKS")
		}
		keys[metadata.KeyID] = publicKey
	}
	if len(keys) == 0 {
		return nil, errors.New("JWKS contains no usable key")
	}
	return &keySet{fetchedAt: fetchedAt, keys: keys}, nil
}

func compatibleRSAKey(metadata rawJWKMetadata, jwk jose.JSONWebKey) (*rsa.PublicKey, bool) {
	if strings.TrimSpace(metadata.KeyID) == "" ||
		(metadata.Alg != "" && metadata.Alg != AllowedAlgorithm) ||
		(metadata.Use != "" && metadata.Use != "sig") ||
		!verificationKeyOps(metadata.KeyOps) ||
		!canonicalRSAParameters(metadata.N, metadata.E) ||
		!jwk.IsPublic() {
		return nil, false
	}
	publicKey, ok := jwk.Key.(*rsa.PublicKey)
	if !ok || publicKey.N == nil || publicKey.N.BitLen() < 2048 || publicKey.E < 3 {
		return nil, false
	}
	return publicKey, true
}

func verificationKeyOps(operations []string) bool {
	if len(operations) == 0 {
		return true
	}
	if len(operations) != 1 {
		return false
	}
	return operations[0] == "verify"
}

func canonicalRSAParameters(modulus, exponent string) bool {
	n, err := decodeCanonicalSegment(modulus)
	if err != nil || len(n) == 0 {
		return false
	}
	e, err := decodeCanonicalSegment(exponent)
	if err != nil || len(e) == 0 {
		return false
	}
	// The JWK integer encoding is unsigned and minimal.
	if n[0] == 0 || e[0] == 0 {
		return false
	}
	exponentValue := new(big.Int).SetBytes(e)
	return exponentValue.IsInt64() && exponentValue.Int64() >= 3
}
