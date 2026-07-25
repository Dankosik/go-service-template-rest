package telemetry

import (
	"os"
	"strings"
	"testing"
)

func TestResolveInstanceIDPrefersConfiguredValue(t *testing.T) {
	t.Parallel()

	if got := ResolveInstanceID("  configured-instance  "); got != "configured-instance" {
		t.Fatalf("ResolveInstanceID(configured) = %q, want %q", got, "configured-instance")
	}
}

func TestResolveInstanceIDFallsBackToHostname(t *testing.T) {
	t.Parallel()

	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		t.Skip("host has no usable hostname; the random fallback is covered separately")
	}

	if got := ResolveInstanceID(""); got != strings.TrimSpace(hostname) {
		t.Fatalf("ResolveInstanceID(\"\") = %q, want the hostname %q", got, hostname)
	}
}

// TestResolveInstanceIDNeverReturnsEmpty is the property the resource depends on.
// An empty service.instance.id is the failure this resolution exists to prevent:
// it makes every replica's exported telemetry identical, so cumulative counters
// from different replicas collide into one series.
func TestResolveInstanceIDNeverReturnsEmpty(t *testing.T) {
	t.Parallel()

	for _, configured := range []string{"", "   ", "\t\n"} {
		if got := ResolveInstanceID(configured); strings.TrimSpace(got) == "" {
			t.Fatalf("ResolveInstanceID(%q) = %q, want a non-empty identifier", configured, got)
		}
	}
}
