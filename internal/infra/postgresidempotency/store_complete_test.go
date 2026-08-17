package postgresidempotency

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestCompletionFingerprintFence(t *testing.T) {
	fingerprint, err := httpidempotency.NewFingerprint("v2", []byte("canonical"))
	if err != nil {
		t.Fatalf("new fingerprint: %v", err)
	}
	if !sameFingerprint("v2", fingerprint.Digest[:], fingerprint) {
		t.Fatal("matching terminal fingerprint did not pass its fence")
	}
	fingerprint.Digest[0]++
	if sameFingerprint("v2", fingerprint.Digest[:], httpidempotency.Fingerprint{Version: "v2"}) {
		t.Fatal("different terminal fingerprint passed its fence")
	}
}
