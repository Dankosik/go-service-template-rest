// Package httpidempotency owns the fixed request identity and replay value used
// by the PostgreSQL HTTP idempotency adapter.
package httpidempotency

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/failure"
)

const (
	Header      = "Idempotency-Key"
	MaxKeyBytes = 255

	maxResultBytes = 1 << 20
	resultSchema   = 1
)

var (
	ErrInvalidKey         = errors.New("idempotency key is invalid")
	ErrInvalidScope       = errors.New("idempotency scope is invalid")
	ErrInvalidFingerprint = errors.New("idempotency fingerprint is invalid")
	ErrInvalidResult      = errors.New("idempotency result is invalid")
	ErrMismatch           = errors.New("idempotency key reused for different input")
	ErrUnavailable        = errors.New("idempotency is unavailable")
	ErrOutcomeUnknown     = errors.New("idempotency commit outcome is unknown")
	ErrIntegrity          = errors.New("idempotency evidence is inconsistent")
)

// ClassifyError maps the closed component failures once for every transport.
func ClassifyError(err error) (failure.Classification, bool) {
	switch {
	case errors.Is(err, ErrInvalidKey):
		return failure.Classification{Code: failure.CodeBadRequest, Detail: "idempotency key is missing or invalid"}, true
	case errors.Is(err, ErrMismatch):
		return failure.Classification{Code: failure.CodeIdempotencyKeyMismatch, Detail: "idempotency key was used for different input"}, true
	case errors.Is(err, ErrUnavailable):
		return failure.Classification{Code: failure.CodeIdempotencyUnavailable, Detail: "idempotency is unavailable"}, true
	case errors.Is(err, ErrOutcomeUnknown):
		return failure.Classification{Code: failure.CodeIdempotencyOutcomeUnknown, Detail: "idempotency outcome is unknown"}, true
	default:
		return failure.Classification{}, false
	}
}

// Scope is the business identity a key is unique within. Caller is derived
// from the verified principal; Resource is optional when Operation alone is
// the complete business scope.
type Scope struct {
	Caller    string
	Operation string
	Resource  string
}

func (s Scope) valid() bool {
	return strings.TrimSpace(s.Caller) != "" && strings.TrimSpace(s.Operation) != ""
}

// Request is the opaque durable identity produced from one authorized typed
// request. Callers choose scope and semantic input, never hashing mechanics.
type Request struct {
	identity           [sha256.Size]byte
	fingerprintVersion int16
	fingerprint        [sha256.Size]byte
}

type keyValuesContextKey struct{}

// CaptureKey preserves duplicate header lines without parsing them before
// authentication. NewRequestFromContext applies the grammar in the handler.
func CaptureKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(contextWithKeyValues(r.Context(), r.Header.Values(Header))))
	})
}

func contextWithKeyValues(ctx context.Context, values []string) context.Context {
	return context.WithValue(ctx, keyValuesContextKey{}, slices.Clone(values))
}

// NewRequestFromContext applies the fixed key grammar after authentication.
func NewRequestFromContext(
	ctx context.Context,
	scope Scope,
	fingerprintVersion int16,
	semanticInput any,
) (Request, error) {
	values, _ := ctx.Value(keyValuesContextKey{}).([]string)
	key, err := parseKey(values)
	if err != nil {
		return Request{}, err
	}
	return NewRequest(scope, key, fingerprintVersion, semanticInput)
}

// NewRequest validates one key and hashes an operation-owned stable semantic
// input. Keep fingerprintVersion unchanged while equivalent requests must replay.
func NewRequest(scope Scope, key string, fingerprintVersion int16, semanticInput any) (Request, error) {
	if !scope.valid() {
		return Request{}, ErrInvalidScope
	}
	if !validKey(key) {
		return Request{}, ErrInvalidKey
	}
	if fingerprintVersion <= 0 {
		return Request{}, ErrInvalidFingerprint
	}
	canonical, err := json.Marshal(semanticInput)
	if err != nil {
		return Request{}, fmt.Errorf("%w: encode semantic input: %w", ErrInvalidFingerprint, err)
	}

	return Request{
		identity:           digest("http-idempotency.identity.v2", scope.Caller, scope.Operation, scope.Resource, key),
		fingerprintVersion: fingerprintVersion,
		fingerprint:        digestBytes("http-idempotency.fingerprint.v2", canonical),
	}, nil
}

func parseKey(values []string) (string, error) {
	if len(values) != 1 || !validKey(values[0]) {
		return "", ErrInvalidKey
	}
	return values[0], nil
}

func validKey(value string) bool {
	if value == "" || len(value) > MaxKeyBytes {
		return false
	}
	for i := range len(value) {
		if !validKeyByte(value[i]) {
			return false
		}
	}
	return true
}

func validKeyByte(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' ||
		strings.ContainsRune("!#$%&'*+-.^_`|~", rune(value))
}

func digest(domain string, values ...string) [sha256.Size]byte {
	hash := sha256.New()
	writePart(hash.Write, []byte(domain))
	for _, value := range values {
		writePart(hash.Write, []byte(value))
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func digestBytes(domain string, value []byte) [sha256.Size]byte {
	hash := sha256.New()
	writePart(hash.Write, []byte(domain))
	writePart(hash.Write, value)
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func writePart(write func([]byte) (int, error), value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = write(length[:])
	_, _ = write(value)
}

func (r Request) Valid() bool {
	return r.identity != [sha256.Size]byte{} && r.fingerprintVersion > 0 && r.fingerprint != [sha256.Size]byte{}
}

func (r Request) Identity() []byte { return r.identity[:] }

func (r Request) Fingerprint() (int16, []byte) {
	return r.fingerprintVersion, r.fingerprint[:]
}

func (r Request) MatchesFingerprint(version int16, fingerprint []byte) bool {
	return version == r.fingerprintVersion && len(fingerprint) == len(r.fingerprint) &&
		subtle.ConstantTimeCompare(fingerprint, r.fingerprint[:]) == 1
}

// Work is the business effect. Repository is transaction-bound by the concrete
// adapter before feature code receives it.
type Work[Repository, Response any] func(context.Context, Repository) (Response, error)

// Codec is the closed generated-response persistence format for one operation.
type Codec[Response any] struct{ status int }

// JSONCodec persists one concrete generated success response. The generated
// response itself later renders the public wire representation.
func JSONCodec[Response any](status int) Codec[Response] {
	return Codec[Response]{status: status}
}

func (c Codec[Response]) Valid() bool {
	return c.status >= http.StatusOK && c.status < http.StatusMultipleChoices
}

func (c Codec[Response]) Encode(response Response) ([]byte, error) {
	if !c.Valid() {
		return nil, fmt.Errorf("%w: status %d is not a success", ErrInvalidResult, c.status)
	}
	body, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("%w: encode generated response: %w", ErrInvalidResult, err)
	}
	encoded, err := json.Marshal(storedResult{Schema: resultSchema, Status: c.status, Body: body})
	if err != nil {
		return nil, fmt.Errorf("%w: encode: %w", ErrInvalidResult, err)
	}
	if len(encoded) > maxResultBytes {
		return nil, fmt.Errorf("%w: exceeds %d bytes", ErrInvalidResult, maxResultBytes)
	}
	return encoded, nil
}

func (c Codec[Response]) Decode(encoded []byte) (Response, error) {
	var response Response
	if !c.Valid() || len(encoded) == 0 || len(encoded) > maxResultBytes {
		return response, ErrInvalidResult
	}
	var stored storedResult
	if err := json.Unmarshal(encoded, &stored); err != nil {
		return response, fmt.Errorf("%w: decode: %w", ErrInvalidResult, err)
	}
	if stored.Schema != resultSchema {
		return response, fmt.Errorf("%w: result schema %d", ErrInvalidResult, stored.Schema)
	}
	if stored.Status != c.status {
		return response, fmt.Errorf("%w: stored status %d, want %d", ErrInvalidResult, stored.Status, c.status)
	}
	if err := json.Unmarshal(stored.Body, &response); err != nil {
		return response, fmt.Errorf("%w: decode generated response: %w", ErrInvalidResult, err)
	}
	return response, nil
}

// Executor is the feature-facing seam. Bootstrap supplies the concrete
// adapter's Execute method; the handler sees no PostgreSQL type.
type Executor[Repository, Response any] func(
	context.Context,
	Request,
	Work[Repository, Response],
) (Response, bool, error)

type storedResult struct {
	Schema int    `json:"schema"`
	Status int    `json:"status"`
	Body   []byte `json:"body,omitempty"`
}
