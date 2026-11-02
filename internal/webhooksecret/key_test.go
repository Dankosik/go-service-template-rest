package webhooksecret

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeAndValidKeyBounds(t *testing.T) {
	t.Parallel()

	for _, size := range []int{32, 64} {
		key := bytes.Repeat([]byte{'k'}, size)
		if !ValidKey(key) {
			t.Fatalf("ValidKey(%d bytes) = false, want true", size)
		}
		decoded, ok := Decode("whsec_" + base64.StdEncoding.EncodeToString(key))
		if !ok || !bytes.Equal(decoded, key) {
			t.Fatalf("Decode(%d bytes) = %d bytes, %t, want the key back", size, len(decoded), ok)
		}
	}
	for _, size := range []int{31, 65} {
		key := bytes.Repeat([]byte{'k'}, size)
		if ValidKey(key) {
			t.Fatalf("ValidKey(%d bytes) = true, want false", size)
		}
		if _, ok := Decode("whsec_" + base64.StdEncoding.EncodeToString(key)); ok {
			t.Fatalf("Decode(%d bytes) accepted an out-of-bounds key", size)
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
	for range 2 {
		if !bindings.Bind(secret, "a") {
			t.Fatal("Bind() rejected the same principal")
		}
	}
	if !bindings.Bind(bytes.Repeat([]byte{'m'}, 32), "b") {
		t.Fatal("Bind() rejected a distinct secret")
	}
	if bindings.Bind(secret, "b") {
		t.Fatal("Bind() accepted a secret bound to another principal")
	}
}
