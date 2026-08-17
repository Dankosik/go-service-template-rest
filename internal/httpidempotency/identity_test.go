package httpidempotency

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestHTTPIdempotencyKeyParser(t *testing.T) {
	t.Parallel()
	for value := range 0x100 {
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
	t.Parallel()
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
	for _, changed := range []widgetInput{{Name: "other"}, {Name: "widget", Retries: new(4)}} {
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
	if _, err := NewFingerprint(" ", nil); err == nil {
		t.Fatal("NewFingerprint() error = nil")
	}
	if _, err := EncodeIdentity(Scope{Authority: "authority-A", OperationID: "CreateWidget", APIVersion: "v1"}, ""); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("EncodeIdentity() error = %v, want ErrInvalidKey", err)
	}
}

func TestEncodeResultRejectsResponsesThatCannotBeSafelyReplayed(t *testing.T) {
	t.Parallel()
	valid := Result{
		Status:    http.StatusCreated,
		MediaType: "application/json",
		Codec:     "create-widget/v1",
		Headers:   http.Header{"Location": {"/widgets/w_1"}},
		Payload:   []byte(`{"id":"w_1"}`),
	}
	for _, test := range []struct {
		name   string
		update func(*Result)
	}{
		{name: "non replay status", update: func(result *Result) { result.Status = http.StatusOK }},
		{name: "missing media type", update: func(result *Result) { result.MediaType = " " }},
		{name: "missing codec", update: func(result *Result) { result.Codec = " " }},
		{name: "undeclared codec", update: func(result *Result) { result.Codec = "other/v1" }},
		{name: "forbidden header", update: func(result *Result) { result.Headers = http.Header{"Set-Cookie": {"secret"}} }},
		{name: "duplicate header", update: func(result *Result) { result.Headers = http.Header{"Location": {"/one"}, "location": {"/two"}} }},
		{name: "empty header values", update: func(result *Result) { result.Headers = http.Header{"Location": nil} }},
		{name: "invalid header value", update: func(result *Result) { result.Headers = http.Header{"Location": {"/one\r\ntwo"}} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := valid
			test.update(&result)
			if _, err := EncodeResult(testContract(), result); err == nil {
				t.Fatal("EncodeResult() error = nil")
			}
		})
	}
	invalid := testContract()
	invalid.ResultMaxBytes = 0
	if _, err := EncodeResult(invalid, valid); err == nil {
		t.Fatal("EncodeResult() error = nil")
	}
	tooSmall := testContract()
	tooSmall.ResultMaxBytes = 1
	if _, err := EncodeResult(tooSmall, valid); err == nil {
		t.Fatal("EncodeResult() error = nil")
	}
}

func TestDecodeResultRejectsCorruptOrOverlongRetainedData(t *testing.T) {
	t.Parallel()
	contract := testContract()
	encoded, err := EncodeResult(contract, Result{
		Status:    http.StatusCreated,
		MediaType: "application/json",
		Codec:     "create-widget/v1",
		Payload:   []byte(`{"id":"w_1"}`),
	})
	if err != nil {
		t.Fatalf("EncodeResult() error = %v", err)
	}
	tooSmall := contract
	tooSmall.ResultMaxBytes = len(encoded) - 1
	for _, test := range []struct {
		name     string
		contract Contract
		encoded  []byte
	}{
		{name: "truncated", contract: contract, encoded: encoded[:len(encoded)-1]},
		{name: "trailing bytes", contract: contract, encoded: append(append([]byte(nil), encoded...), 0)},
		{name: "overlong", contract: tooSmall, encoded: encoded},
		{name: "wrong envelope", contract: contract, encoded: []byte("other")},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeResult(test.contract, test.encoded); err == nil {
				t.Fatal("DecodeResult() error = nil")
			}
		})
	}
	invalid := testContract()
	invalid.ResultMaxBytes = 0
	if _, err := DecodeResult(invalid, []byte{}); err == nil {
		t.Fatal("DecodeResult() error = nil")
	}

	contract.ResultMaxBytes = 4096
	prefix := append([]byte("http-idempotency.result.v1\x00"), 0, http.StatusCreated)
	mediaType := appendRetainedField(prefix, "application/json")
	codec := appendRetainedField(mediaType, "create-widget/v1")
	for _, test := range []struct {
		name    string
		encoded []byte
	}{
		{name: "status", encoded: []byte("http-idempotency.result.v1\x00")},
		{name: "media type", encoded: prefix},
		{name: "codec", encoded: mediaType},
		{name: "header count", encoded: codec},
		{name: "header name", encoded: append(codec, 0, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeResult(contract, test.encoded); err == nil {
				t.Fatal("DecodeResult() error = nil")
			}
		})
	}
}

func TestDecodeResultRejectsInvalidRetainedFields(t *testing.T) {
	t.Parallel()
	contract := testContract()
	contract.ResultMaxBytes = 4096
	for _, test := range []struct {
		name    string
		encoded []byte
	}{
		{name: "non replay status", encoded: retainedResult(http.StatusOK, "application/json", "create-widget/v1", nil)},
		{name: "blank media type", encoded: retainedResult(http.StatusCreated, "", "create-widget/v1", nil)},
		{name: "undeclared codec", encoded: retainedResult(http.StatusCreated, "application/json", "other/v1", nil)},
		{name: "uppercase retained header", encoded: retainedResult(http.StatusCreated, "application/json", "create-widget/v1", []retainedHeader{{name: "Location", values: []string{"/widgets/w_1"}}})},
		{name: "duplicate retained header", encoded: retainedResult(http.StatusCreated, "application/json", "create-widget/v1", []retainedHeader{{name: "location", values: []string{"/one"}}, {name: "location", values: []string{"/two"}}})},
		{name: "empty retained header values", encoded: retainedResult(http.StatusCreated, "application/json", "create-widget/v1", []retainedHeader{{name: "location"}})},
		{name: "invalid retained header value", encoded: retainedResult(http.StatusCreated, "application/json", "create-widget/v1", []retainedHeader{{name: "location", values: []string{"/one\r\ntwo"}}})},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeResult(contract, test.encoded); err == nil {
				t.Fatal("DecodeResult() error = nil")
			}
		})
	}
}

func TestResultReaderRejectsTruncatedRetainedFields(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		read func(*resultReader) error
	}{
		{name: "status", read: func(reader *resultReader) error { _, err := reader.uint16(); return err }},
		{name: "length", read: func(reader *resultReader) error { _, err := reader.bytes(); return err }},
		{name: "payload", read: func(reader *resultReader) error { _, err := reader.bytes(); return err }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			data := []byte(nil)
			if test.name == "length" {
				data = []byte{0, 0, 0}
			}
			if test.name == "payload" {
				data = []byte{0, 0, 0, 3, 'x'}
			}
			if err := test.read(&resultReader{data: data}); err == nil {
				t.Fatal("retained-field reader error = nil")
			}
		})
	}
}

type retainedHeader struct {
	name   string
	values []string
}

func retainedResult(status int, mediaType, codec string, headers []retainedHeader) []byte {
	encoded := append([]byte("http-idempotency.result.v1\x00"), byte(status>>8), byte(status))
	encoded = appendRetainedField(encoded, mediaType)
	encoded = appendRetainedField(encoded, codec)
	var count [2]byte
	binary.BigEndian.PutUint16(count[:], uint16(len(headers)))
	encoded = append(encoded, count[:]...)
	for _, header := range headers {
		encoded = appendRetainedField(encoded, header.name)
		binary.BigEndian.PutUint16(count[:], uint16(len(header.values)))
		encoded = append(encoded, count[:]...)
		for _, value := range header.values {
			encoded = appendRetainedField(encoded, value)
		}
	}
	return appendRetainedField(encoded, `{"id":"w_1"}`)
}

func appendRetainedField(dst []byte, value string) []byte {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(value)))
	dst = append(dst, length[:]...)
	return append(dst, value...)
}

func testTchar(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(value))
}
