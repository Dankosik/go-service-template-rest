// Package configtest owns the environment isolation a parity test needs.
//
// The rules internal/config enforces are restated by the runtime owners that
// consume them, because the depguard rule config_no_runtime_owners stops its
// parent from importing them. Each parity test therefore lives with its runtime
// owner — the direction the import works — so more than one package needs to run
// the real loader against one corpus entry at a time. This is that step, owned
// once: a helper that exists three times is the shape those tests prevent.
package configtest

import (
	"os"
	"strings"
	"testing"
)

// IsolateEnv clears every namespaced variable the ambient shell may hold, so one
// corpus entry decides the load on its own rather than inheriting a developer's
// exported overrides. testing.TB.Setenv captures each prior value for restore and
// marks the test non-parallel, which is what keeps these mutations from colliding
// with another case.
//
// What it leaves behind is the minimum a load accepts, not an empty environment:
// a section the active build profile requires would otherwise fail every case
// for a reason unrelated to the values under test. A caller varying one of these
// values sets it after this returns.
func IsolateEnv(tb testing.TB) {
	tb.Helper()

	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found || !strings.HasPrefix(key, "APP__") {
			continue
		}
		tb.Setenv(key, value)
		if err := os.Unsetenv(key); err != nil {
			tb.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	tb.Setenv("APP__APP__ENV", "local")
	// profile:authn-bearer:start
	tb.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	tb.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	// profile:authn-bearer:end
	// profile:outbound-auth-oauth2-client-credentials:start
	// #nosec G101 -- Deterministic test-only placeholders are installed only through testing.TB.Setenv.
	for key, value := range map[string]string{
		"APP__OUTBOUND_AUTH__TOKEN_URL":     "https://auth.example.com/oauth/token",
		"APP__OUTBOUND_AUTH__CLIENT_ID":     "test-client",
		"APP__OUTBOUND_AUTH__CLIENT_SECRET": "test-client-secret",
		"APP__OUTBOUND_AUTH__SCOPES":        "payments.read payments.write",
		"APP__OUTBOUND_AUTH__RESOURCE":      "https://payments.example.com",
	} {
		tb.Setenv(key, value)
	}
	// profile:outbound-auth-oauth2-client-credentials:end
	// profile:object-storage:start
	for key, value := range map[string]string{
		"APP__OBJECT_STORAGE__PROVIDER":              "amazon_s3",
		"APP__OBJECT_STORAGE__REGION":                "us-east-1",
		"APP__OBJECT_STORAGE__BUCKET":                "examplebucket",
		"APP__OBJECT_STORAGE__EXPECTED_BUCKET_OWNER": "123456789012",
		"APP__OBJECT_STORAGE__CREDENTIAL_SOURCE":     "aws_default",
		"APP__OBJECT_STORAGE__MAX_OBJECT_BYTES":      "10485760",
	} {
		tb.Setenv(key, value)
	}
	// profile:object-storage:end
}
