package oidcjwt

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

type keySet struct {
	fetchedAt time.Time
	keys      map[string]*rsa.PublicKey
}

// verifies reports whether this set holds the token's key and that key signs
// exactly the bytes the claims were parsed from. Re-comparing the payload is
// what stops a signature made over a different body from being accepted for
// this one. Age is not consulted here; the caller decides what a match from an
// old set is worth.
func (s *keySet) verifies(parsed parsedToken) bool {
	if s == nil {
		return false
	}
	key := s.keys[parsed.keyID]
	if key == nil {
		return false
	}
	payload, err := parsed.signed.Verify(key)
	return err == nil && bytes.Equal(payload, parsed.payload)
}

// trustStore holds the key set a [Verifier] currently trusts and announces each
// replacement.
//
// A set is replaced wholesale and never mutated, so a verification reads one
// consistent set without blocking a refresh. Making install the only writer is
// what lets every reader treat current as non-nil, and what keeps a replacement
// from being installed without being announced — arming one and forgetting the
// other is silent, and the symptom is a refresh cadence still measured against
// the set it just replaced. [refreshSchedule] exists against the same hazard.
type trustStore struct {
	keys    atomic.Pointer[keySet]
	changed chan struct{}
}

func newTrustStore(initial *keySet) *trustStore {
	store := &trustStore{changed: make(chan struct{}, 1)}
	store.keys.Store(initial)
	return store
}

func (s *trustStore) current() *keySet {
	return s.keys.Load()
}

func (s *trustStore) install(keys *keySet) {
	s.keys.Store(keys)
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// replaced carries one nudge per install, and drops a nudge nobody has collected
// yet rather than blocking the installer. A reader that wakes late is not owed
// one wake per replacement: it reads the newest set from current.
func (s *trustStore) replaced() <-chan struct{} {
	return s.changed
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
		// One entry, decoded twice on purpose. jose parses the key material but
		// keeps neither the raw n and e strings canonicalRSAParameters must see nor
		// the optional alg, use, and key_ops compatibleRSAKey narrows on.
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

// compatibleRSAKey reports whether one JWK entry may verify this service's
// tokens. An entry that fails a term is skipped rather than refused, because a
// provider legitimately publishes keys for algorithms and uses this service does
// not accept; parseKeySet fails closed only when no entry survives.
//
// alg, use, and key_ops are optional RFC 7517 metadata, so each is read only
// when the provider set it — three vocabularies for the same narrowing, and
// silence contradicts nothing. kid is required even though the RFC leaves it
// optional, because the set is indexed by it and a token names one: an entry
// without a kid could never be selected. The size and exponent floor below is
// this service's own minimum rather than anything the provider declared.
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

// verificationKeyOps accepts key_ops only when it is absent or says exactly
// "verify". A key the provider also signs or wraps with is refused rather than
// narrowed to its verify half: RFC 7517 discourages combining unrelated
// operations, so a combined key is a provider statement this boundary should not
// reinterpret on its own.
func verificationKeyOps(operations []string) bool {
	if len(operations) == 0 {
		return true
	}
	if len(operations) != 1 {
		return false
	}
	return operations[0] == "verify"
}

// canonicalRSAParameters reports whether n and e are the only encoding of the
// integers they name. It matters for the reason decodeCanonicalSegment exists:
// several segments decode to one value, so without this a provider could publish
// what reads as two distinct entries for the same key, and parseKeySet's
// ambiguity check would have nothing to catch.
func canonicalRSAParameters(modulus, exponent string) bool {
	n, err := decodeCanonicalSegment(modulus)
	if err != nil || len(n) == 0 {
		return false
	}
	e, err := decodeCanonicalSegment(exponent)
	if err != nil || len(e) == 0 {
		return false
	}
	// RFC 7518 requires the JWK integer encoding to be unsigned and minimal, so a
	// leading zero byte is a second spelling of the same value, not a larger one.
	if n[0] == 0 || e[0] == 0 {
		return false
	}
	exponentValue := new(big.Int).SetBytes(e)
	return exponentValue.IsInt64() && exponentValue.Int64() >= 3
}
