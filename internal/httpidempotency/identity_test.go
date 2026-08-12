package httpidempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestHTTPIdempotencyKeyParser(t *testing.T) {
	for value := 0; value <= 0xff; value++ {
		key := string([]byte{byte(value)})
		_, err := ParseKey([]string{key}, 1)
		wantValid := testTchar(byte(value))
		if (err == nil) != wantValid {
			t.Fatalf("byte 0x%02x valid = %t, want %t", value, err == nil, wantValid)
		}
	}

	for _, punctuation := range []string{"!", "#", "$", "%", "&", "'", "*", "+", "-", ".", "^", "_", "`", "|", "~"} {
		if got, err := ParseKey([]string{"A" + punctuation + "9"}, 3); err != nil || got != "A"+punctuation+"9" {
			t.Fatalf("ParseKey(%q) = (%q, %v), want accepted unchanged key", punctuation, got, err)
		}
	}

	for _, values := range [][]string{
		nil,
		{""},
		{"a,b"},
		{"\"a\""},
		{"a b"},
		{"a\tb"},
		{"a\x00b"},
		{"é"},
		{"same", "same"},
		{"one", "two"},
	} {
		if _, err := ParseKey(values, 64); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("ParseKey(%q) error = %v, want ErrInvalidKey", values, err)
		}
	}

	if _, err := ParseKey([]string{strings.Repeat("a", 64)}, 64); err != nil {
		t.Fatalf("ParseKey(boundary) error = %v", err)
	}
	if _, err := ParseKey([]string{strings.Repeat("a", 65)}, 64); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("ParseKey(over boundary) error = %v, want ErrInvalidKey", err)
	}
	upper, err := ParseKey([]string{"AbC"}, 64)
	if err != nil {
		t.Fatalf("ParseKey(upper) error = %v", err)
	}
	lower, err := ParseKey([]string{"abc"}, 64)
	if err != nil {
		t.Fatalf("ParseKey(lower) error = %v", err)
	}
	if upper == lower {
		t.Fatal("case variants were aliased")
	}
}

func TestHTTPIdempotencyCanonicalVectors(t *testing.T) {
	scope := Scope{
		Authority:   "authority-A",
		OperationID: "CreateWidget",
		APIVersion:  "v1",
		Resource:    "widgets",
		Environment: "test",
		Region:      "region-1",
	}
	encodedIdentity, err := EncodeIdentity(scope, "AbC_123-xyz")
	if err != nil {
		t.Fatalf("EncodeIdentity() error = %v", err)
	}
	const wantIdentityHex = "687474702d6964656d706f74656e63792e6964656e746974792e7631000000000b617574686f726974792d410000000c4372656174655769646765740000000276310000000777696467657473000000047465737400000008726567696f6e2d310000000b4162435f3132332d78797a"
	if got := hex.EncodeToString(encodedIdentity); got != wantIdentityHex {
		t.Fatalf("identity bytes = %s, want %s", got, wantIdentityHex)
	}
	if got := sha256.Sum256(encodedIdentity); got != [sha256.Size]byte{0xb4, 0x7f, 0xe8, 0x51, 0x09, 0xe0, 0xdb, 0xef, 0xf7, 0x60, 0x97, 0xf0, 0x3b, 0x81, 0x8b, 0xa7, 0xdd, 0x3d, 0x55, 0x33, 0xd5, 0xbb, 0x90, 0x38, 0x1d, 0xa7, 0xcf, 0x54, 0x00, 0x82, 0x5c, 0x40} {
		t.Fatalf("identity digest = %x", got)
	}

	type widgetInput struct {
		Name    string
		Retries *int
	}
	canonicalize := func(input widgetInput) []byte {
		retries := 3
		if input.Retries != nil {
			retries = *input.Retries
		}
		return []byte(input.Name + "\x00" + string(rune(retries)))
	}
	three := 3
	first, err := NewFingerprint("v1", canonicalize(widgetInput{Name: "widget"}))
	if err != nil {
		t.Fatalf("NewFingerprint(default) error = %v", err)
	}
	equivalent, err := NewFingerprint("v1", canonicalize(widgetInput{Name: "widget", Retries: &three}))
	if err != nil {
		t.Fatalf("NewFingerprint(explicit default) error = %v", err)
	}
	if first != equivalent {
		t.Fatal("defaulted and explicit typed inputs produced different fingerprints")
	}
	for _, changed := range []widgetInput{{Name: "other"}, {Name: "widget", Retries: intPointer(4)}} {
		fingerprint, err := NewFingerprint("v1", canonicalize(changed))
		if err != nil {
			t.Fatalf("NewFingerprint(changed) error = %v", err)
		}
		if fingerprint == first {
			t.Fatalf("semantic change %+v produced the original fingerprint", changed)
		}
	}

	contract := testContract()
	result := Result{
		Status:    201,
		MediaType: "application/json",
		Codec:     "create-widget/v1",
		Headers:   map[string][]string{"Location": {"/widgets/w_1"}},
		Payload:   []byte(`{"id":"w_1"}`),
	}
	encodedResult, err := EncodeResult(contract, result)
	if err != nil {
		t.Fatalf("EncodeResult() error = %v", err)
	}
	const wantResultHex = "687474702d6964656d706f74656e63792e726573756c742e76310000c9000000106170706c69636174696f6e2f6a736f6e000000106372656174652d7769646765742f76310001000000086c6f636174696f6e00010000000c2f776964676574732f775f310000000c7b226964223a22775f31227d"
	if got := hex.EncodeToString(encodedResult); got != wantResultHex {
		t.Fatalf("result bytes = %s, want %s", got, wantResultHex)
	}
	decoded, err := DecodeResult(contract, encodedResult)
	if err != nil {
		t.Fatalf("DecodeResult() error = %v", err)
	}
	if decoded.Status != 201 || decoded.MediaType != "application/json" || decoded.Codec != "create-widget/v1" || decoded.Headers.Get("Location") != "/widgets/w_1" || string(decoded.Payload) != `{"id":"w_1"}` {
		t.Fatalf("decoded result = %+v", decoded)
	}
	result.Headers.Set("Set-Cookie", "secret")
	if _, err := EncodeResult(contract, result); err == nil {
		t.Fatal("EncodeResult() accepted an excluded header")
	}
}

func testTchar(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(value))
}

func intPointer(value int) *int {
	return &value
}
