package httpidempotency

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"strings"
)

var ErrInvalidKey = errors.New("idempotency key is invalid")

// Scope contains the endpoint-owned values that partition an idempotency key.
type Scope struct {
	Authority   string
	OperationID string
	APIVersion  string
	Resource    string
	Environment string
	Region      string
}

// Validate checks the endpoint-owned identity components the shared framing
// cannot safely infer.
func (s Scope) Validate() error {
	if strings.TrimSpace(s.Authority) == "" || strings.TrimSpace(s.OperationID) == "" || strings.TrimSpace(s.APIVersion) == "" {
		return errors.New("idempotency identity: authority, operation ID, and API version are required")
	}
	return nil
}

// Fingerprint is the versioned digest of an endpoint's typed semantic input.
type Fingerprint struct {
	Version string
	Digest  [sha256.Size]byte
}

// Attempt is request-local input to the later Store protocol. It retains raw
// values only for the duration of the request; persistence uses its digests.
type Attempt struct {
	Scope       Scope
	Key         string
	Identity    [sha256.Size]byte
	Fingerprint Fingerprint
}

// ParseKey accepts exactly one RFC 9110 tchar field value without normalizing
// its bytes or case.
func ParseKey(values []string, maxBytes int) (string, error) {
	if maxBytes <= 0 || len(values) != 1 {
		return "", ErrInvalidKey
	}
	key := values[0]
	if len(key) > maxBytes || !validToken(key) {
		return "", ErrInvalidKey
	}
	return key, nil
}

// NewFingerprint hashes one operation-owned canonical semantic document.
func NewFingerprint(version string, canonical []byte) (Fingerprint, error) {
	if strings.TrimSpace(version) == "" {
		return Fingerprint{}, errors.New("idempotency fingerprint: version is required")
	}
	return Fingerprint{Version: version, Digest: sha256.Sum256(canonical)}, nil
}

// EncodeIdentity returns the versioned bytes whose SHA-256 is the durable
// identity token.
func EncodeIdentity(scope Scope, key string) ([]byte, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if key == "" {
		return nil, ErrInvalidKey
	}
	fields := [...]string{scope.Authority, scope.OperationID, scope.APIVersion, scope.Resource, scope.Environment, scope.Region, key}
	encoded := make([]byte, 0, len("http-idempotency.identity.v1")+1+len(key)+64)
	encoded = append(encoded, "http-idempotency.identity.v1"...)
	encoded = append(encoded, 0)
	for _, field := range fields {
		if len(field) > math.MaxUint32 {
			return nil, errors.New("idempotency identity: field exceeds uint32 length")
		}
		var length [4]byte
		binary.BigEndian.PutUint32(length[:], uint32(len(field))) // #nosec G115 -- field length is rejected above math.MaxUint32.
		encoded = append(encoded, length[:]...)
		encoded = append(encoded, field...)
	}
	return encoded, nil
}

// NewAttempt joins validated request values into the value the Store later uses.
func NewAttempt(scope Scope, key string, fingerprint Fingerprint) (Attempt, error) {
	if strings.TrimSpace(fingerprint.Version) == "" {
		return Attempt{}, errors.New("idempotency attempt: fingerprint version is required")
	}
	encoded, err := EncodeIdentity(scope, key)
	if err != nil {
		return Attempt{}, err
	}
	return Attempt{
		Scope:       scope,
		Key:         key,
		Identity:    sha256.Sum256(encoded),
		Fingerprint: fingerprint,
	}, nil
}
