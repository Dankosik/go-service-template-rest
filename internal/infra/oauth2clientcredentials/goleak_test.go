package oauth2clientcredentials

import (
	"crypto/x509"
	"fmt"
	"os"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	previousDebug := os.Getenv("GODEBUG")
	if err := os.Setenv("GODEBUG", appendGoDebug(previousDebug, "x509usefallbackroots=1")); err != nil {
		fmt.Fprintf(os.Stderr, "set GODEBUG for fallback roots: %v\n", err)
		os.Exit(1)
	}
	pki, err := newTestPKI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build outbound-auth test PKI: %v\n", err)
		os.Exit(1)
	}
	suiteTestPKI = pki
	x509.SetFallbackRoots(pki.pool)
	goleak.VerifyTestMain(m, goleak.Cleanup(func(exitCode int) {
		if err := os.Setenv("GODEBUG", previousDebug); err != nil && exitCode == 0 {
			fmt.Fprintf(os.Stderr, "restore GODEBUG: %v\n", err)
			exitCode = 1
		}
		os.Exit(exitCode)
	}))
}

func appendGoDebug(current, setting string) string {
	if current == "" {
		return setting
	}
	return current + "," + setting
}
