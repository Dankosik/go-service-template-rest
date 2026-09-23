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

func TestBindingsRejectCrossBoundSecret(t *testing.T) {
	t.Parallel()

	var bindings Bindings[string]
	secret := bytes.Repeat([]byte{'k'}, 32)
	if !bindings.Bind(secret, "a") || !bindings.Bind(secret, "a") {
		t.Fatal("Bind() rejected the same principal")
	}
	if !bindings.Bind(bytes.Repeat([]byte{'m'}, 32), "b") {
		t.Fatal("Bind() rejected a distinct secret")
	}
	if bindings.Bind(secret, "b") {
		t.Fatal("Bind() accepted a secret bound to another principal")
	}
}
