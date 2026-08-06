// Package configtest owns the environment isolation a parity test needs.
//
// The rules internal/config enforces are restated by the runtime owners that
// consume them, because the depguard rule config_no_runtime_owners stops this
// package's parent from importing them and each restatement has to be held to
// the same answer. Those parity tests live with the runtime owner, which is the
// direction the import works, so more than one package needs to run the real
// configuration loader against one corpus entry at a time. This is that step,
// owned once: without it the next parity test copies it, and a helper that
// exists three times is the shape those tests were written to prevent.
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
	// profile:authn-oidc-jwt:start
	tb.Setenv("APP__AUTHN__ISSUER", "https://issuer.example.com")
	tb.Setenv("APP__AUTHN__AUDIENCE", "https://api.example.com")
	tb.Setenv("APP__AUTHN__TRUSTED_PROXY_CIDRS", "127.0.0.0/8,::1/128")
	// profile:authn-oidc-jwt:end
}
