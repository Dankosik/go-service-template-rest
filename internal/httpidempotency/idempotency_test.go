package httpidempotency

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
)

func TestNewRequestUsesScopeAndSemanticInput(t *testing.T) {
	t.Parallel()

	base, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", 1, struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(): %v", err)
	}
	same, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", 1, struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(same): %v", err)
	}
	changed, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", 1, struct{ Amount int }{Amount: 11})
	if err != nil {
		t.Fatalf("NewRequest(changed): %v", err)
	}
	otherCaller, err := NewRequest(Scope{Caller: "other", Operation: "create"}, "key-1", 1, struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(other caller): %v", err)
	}
	otherVersion, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", 2, struct{ Amount int }{Amount: 10})
	if err != nil {
		t.Fatalf("NewRequest(other version): %v", err)
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
	for name, input := range map[string]struct {
		scope Scope
		key   string
	}{
		"operation": {scope: Scope{Caller: "caller", Operation: "update"}, key: "key-1"},
		"resource":  {scope: Scope{Caller: "caller", Operation: "create", Resource: "widget-1"}, key: "key-1"},
		"key":       {scope: Scope{Caller: "caller", Operation: "create"}, key: "key-2"},
	} {
		changedIdentity, err := NewRequest(input.scope, input.key, 1, struct{ Amount int }{Amount: 10})
		if err != nil {
			t.Fatalf("NewRequest(%s): %v", name, err)
		}
		if bytes.Equal(base.Identity(), changedIdentity.Identity()) {
			t.Fatalf("different %s shared an identity", name)
		}
	}
	if got := hex.EncodeToString(base.Identity()); got != "04304754d2e1051a6656c9a4a47fe49e97b7df96428764e8166b2fac8f826ac4" {
		t.Fatalf("identity vector = %s", got)
	}
	version, fingerprint = base.Fingerprint()
	if got := hex.EncodeToString(fingerprint); version != 1 || got != "2ada3a04075d65481e605c8239af61ef529b95af367e6d700cddf098e7acec65" {
		t.Fatalf("fingerprint vector = v%d %s", version, got)
	}
	version, fingerprint = otherVersion.Fingerprint()
	if base.MatchesFingerprint(version, fingerprint) {
		t.Fatal("different semantic contract versions matched")
	}
	if _, err := NewRequest(Scope{Caller: "caller", Operation: "create"}, "key-1", 0, struct{}{}); !errors.Is(err, ErrInvalidFingerprint) {
		t.Fatalf("NewRequest(invalid version) error = %v, want ErrInvalidFingerprint", err)
	}
}

func TestParseKey(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"!#$%&'*+-.0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ^_`abcdefghijklmnopqrstuvwxyz|~",
		strings.Repeat("a", MaxKeyBytes),
	} {
		if got, err := parseKey([]string{key}); err != nil || got != key {
			t.Fatalf("parseKey() = %q, %v", got, err)
		}
	}
	for _, values := range [][]string{
		nil,
		{""},
		{"a", "a"},
		{"has space"},
		{"has,comma"},
		{`"quoted"`},
		{"has\ttab"},
		{"has\x7fcontrol"},
		{"non-ASCII-щ"},
		{strings.Repeat("a", MaxKeyBytes+1)},
	} {
		if _, err := parseKey(values); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("parseKey(%q) error = %v, want ErrInvalidKey", values, err)
		}
	}
}

func TestNewRequestFromContextRejectsMultipleWireValues(t *testing.T) {
	t.Parallel()

	scope := Scope{Caller: "caller", Operation: "create"}
	ctx := contextWithKeyValues(t.Context(), []string{"key-1", "key-2"})
	if _, err := NewRequestFromContext(ctx, scope, 1, struct{}{}); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("NewRequestFromContext(multiple) error = %v, want ErrInvalidKey", err)
	}
	ctx = contextWithKeyValues(t.Context(), []string{"key-1"})
	if _, err := NewRequestFromContext(ctx, scope, 1, struct{}{}); err != nil {
		t.Fatalf("NewRequestFromContext(single): %v", err)
	}
}

func TestResultRoundTripAndBounds(t *testing.T) {
	t.Parallel()

	type response struct {
		Location string `json:"location"`
		Body     string `json:"body"`
	}
	codec := JSONCodec[response](http.StatusCreated)
	want := response{Location: "/widgets/1", Body: "created"}
	encoded, err := codec.Encode(want)
	if err != nil {
		t.Fatalf("Encode(): %v", err)
	}
	got, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode(): %v", err)
	}
	if got != want {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}

	if _, err := JSONCodec[response](http.StatusInternalServerError).Encode(want); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("Encode(500) error = %v, want ErrInvalidResult", err)
	}
	if _, err := codec.Encode(response{Body: strings.Repeat("x", maxResultBytes)}); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("Encode(oversized) error = %v, want ErrInvalidResult", err)
	}

	body := []byte(`{"body":"created","location":"/widgets/1"}`)
	legacy, err := json.Marshal(struct {
		Schema int                 `json:"schema"`
		Status int                 `json:"status"`
		Header map[string][]string `json:"header"`
		Body   []byte              `json:"body"`
	}{Schema: resultSchema, Status: http.StatusCreated, Header: map[string][]string{"Location": {want.Location}}, Body: body})
	if err != nil {
		t.Fatalf("marshal legacy result: %v", err)
	}
	if got, err := codec.Decode(legacy); err != nil || got != want {
		t.Fatalf("Decode(legacy) = %#v, %v; want %#v", got, err, want)
	}
	type renamedResponse struct {
		URI  string `json:"uri"`
		Body string `json:"body"`
	}
	if _, err := JSONCodec[renamedResponse](http.StatusCreated).Decode(legacy); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("Decode(changed response shape) error = %v, want ErrInvalidResult", err)
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
	for _, internal := range []error{ErrIntegrity, ErrInvalidFingerprint, ErrInvalidResult} {
		if got, ok := ClassifyError(fmt.Errorf("execute: %w", internal)); ok || got != (failure.Classification{}) {
			t.Errorf("ClassifyError(%v) = (%+v, %t), want unclassified internal fault", internal, got, ok)
		}
	}
}
