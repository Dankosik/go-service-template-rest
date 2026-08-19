package httpidempotency

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
)

func TestNewRequestUsesScopeAndSemanticInput(t *testing.T) {
	t.Parallel()

	base, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	same, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(same): %v", err)
	}
	changed, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", struct{ Amount int }{Amount: 11})
	if err != nil {
		t.Fatalf("NewRequest(changed): %v", err)
	}
	otherCaller, err := NewRequest(Scope{Caller: "other", Operation: "create"}, "key-1", struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(other caller): %v", err)
	}

	version, fingerprint := same.Fingerprint()
	if !base.MatchesFingerprint(version, fingerprint) {
		t.Fatal("same semantic input did not match")
	}
	version, fingerprint = changed.Fingerprint()
	if base.MatchesFingerprint(version, fingerprint) {
		t.Fatal("changed semantic input matched")
	}
	if bytes.Equal(base.Identity(), otherCaller.Identity()) {
		t.Fatal("different callers shared an identity")
	}
}

func TestParseKey(t *testing.T) {
	t.Parallel()

	if got, err := ParseKey([]string{"key-1"}); err != nil || got != "key-1" {
		t.Fatalf("ParseKey() = %q, %v", got, err)
	}
	for _, values := range [][]string{nil, {""}, {"a", "b"}, {"has space"}, {strings.Repeat("a", MaxKeyBytes+1)}} {
		if _, err := ParseKey(values); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("ParseKey(%q) error = %v, want ErrInvalidKey", values, err)
		}
	}
}

func TestNewRequestFromContextRejectsMultipleWireValues(t *testing.T) {
	t.Parallel()

	scope := Scope{Caller: "caller", Operation: "create"}
	ctx := ContextWithKeyValues(t.Context(), []string{"key-1", "key-2"})
	if _, err := NewRequestFromContext(ctx, scope, struct{}{}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("NewRequestFromContext(multiple) error = %v, want ErrInvalidKey", err)
	}
	ctx = ContextWithKeyValues(t.Context(), []string{"key-1"})
	if _, err := NewRequestFromContext(ctx, scope, struct{}{}); err != nil {
		t.Fatalf("NewRequestFromContext(single): %v", err)
	}
}

func TestResultRoundTripAndBounds(t *testing.T) {
	t.Parallel()

	want := Result{
		Status: http.StatusCreated,
		Header: http.Header{"Location": {"/widgets/1"}, "Content-Type": {"application/json"}},
		Body:   []byte(`{"id":"1"}`),
	}
	encoded, err := EncodeResult(want)
	if err != nil {
		t.Fatalf("EncodeResult(): %v", err)
	}
	got, err := DecodeResult(encoded)
	if err != nil {
		t.Fatalf("DecodeResult(): %v", err)
	}
	if got.Status != want.Status || !bytes.Equal(got.Body, want.Body) || got.Header.Get("Location") != "/widgets/1" {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}

	if _, err := EncodeResult(Result{Status: http.StatusInternalServerError}); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("EncodeResult(500) error = %v, want ErrInvalidResult", err)
	}
	if _, err := EncodeResult(Result{Status: http.StatusOK, Header: http.Header{"Set-Cookie": {"secret"}}}); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("EncodeResult(Set-Cookie) error = %v, want ErrInvalidResult", err)
	}
}

func TestJSONCodecReconstructsGeneratedSuccess(t *testing.T) {
	t.Parallel()

	type response struct {
		Location string `json:"location"`
		Body     string `json:"body"`
	}
	codec := JSONCodec[response](http.StatusCreated)
	want := response{Location: "/widgets/1", Body: "created"}
	stored, err := codec.Encode(want)
	if err != nil {
		t.Fatalf("Encode(): %v", err)
	}
	got, err := codec.Decode(stored)
	if err != nil {
		t.Fatalf("Decode(): %v", err)
	}
	if got != want {
		t.Fatalf("Decode() = %#v, want %#v", got, want)
	}
}

func TestClassifyError(t *testing.T) {
	t.Parallel()

	classification, ok := ClassifyError(fmt.Errorf("execute: %w", ErrMismatch))
	if !ok || classification.Code != failure.CodeIdempotencyKeyMismatch {
		t.Fatalf("ClassifyError(ErrMismatch) = %+v, %v", classification, ok)
	}
	if _, ok := ClassifyError(errors.New("business error")); ok {
		t.Fatal("ClassifyError classified an unrelated business error")
	}
}
