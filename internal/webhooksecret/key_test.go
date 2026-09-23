package webhooksecret

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeAndValidKeyBounds(t *testing.T) {
	t.Parallel()

	for _, size := range []int{31, 32, 64, 65} {
		key := bytes.Repeat([]byte{'k'}, size)
		want := size >= 32 && size <= 64
		if got := ValidKey(key); got != want {
			t.Fatalf("ValidKey(%d bytes) = %t, want %t", size, got, want)
		}
		encoded := "whsec_" + base64.StdEncoding.EncodeToString(key)
		decoded, ok := Decode(encoded)
		if ok != want || (ok && !bytes.Equal(decoded, key)) {
			t.Fatalf("Decode(%d bytes) = %d bytes, %t, want %t", size, len(decoded), ok, want)
		}
	}
	for _, value := range []string{
		base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{'k'}, 32)),
		"whsec_***",
		"whsec_",
	} {
		if _, ok := Decode(value); ok {
			t.Fatalf("Decode(%q) accepted invalid secret", value)
		}
	}
}
