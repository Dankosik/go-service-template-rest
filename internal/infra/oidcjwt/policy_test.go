package oidcjwt

// Proof for policy.go: the trust values a deployment may hold. The rules this
// package and internal/config both enforce are proven in config_parity_test.go.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicyInputIsHeldToExhaustruct proves the lint entry [PolicyInput]'s own
// documentation calls load-bearing is actually configured.
//
// That entry is the only thing that makes a configured trust value added here
// fail the build until the composition root sets it; doc.go counts it as the one
// item on its list a tool demands. Nothing else notices if it is dropped from
// .golangci.yml — the build goes green and the next added field reaches
// production as a zero value at an authentication boundary.
//
// The assertion is one live, anchored line rather than a substring of the file: a
// substring survives every way this entry actually breaks — commented out during
// a lint triage, or left in place with the pattern widened past the type it pins.
// The module path is not anchored, because a generated service rewrites it.
func TestPolicyInputIsHeldToExhaustruct(t *testing.T) {
	t.Parallel()
	const required = `oidcjwt\.PolicyInput$`
	config, err := os.ReadFile(filepath.Join("..", "..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatalf("read lint configuration: %v", err)
	}
	for line := range strings.SplitSeq(string(config), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, required) {
			return
		}
	}
	t.Fatalf(
		"no active exhaustruct include entry ends in %s; "+
			"without it a field added to PolicyInput ships unset from the composition root",
		required,
	)
}

func TestPolicyRejectsDuplicateTrustedProxyCIDRs(t *testing.T) {
	t.Parallel()
	_, err := NewPolicy(PolicyInput{
		Issuer:            testIssuer,
		Audience:          testAudience,
		TrustedProxyCIDRs: "127.0.0.0/8,127.0.0.1/8",
	})
	if err == nil {
		t.Fatal("NewPolicy() error = nil, want duplicate CIDR rejection")
	}
}
