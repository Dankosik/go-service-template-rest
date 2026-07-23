package bootstrap

import (
	"errors"
	"strings"
	"testing"
)

func TestParseHostMatchersRejectsEmptySuffixSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "leading dot",
			raw:  ".",
			want: "suffix host matcher cannot be empty",
		},
		{
			name: "wildcard dot",
			raw:  "*.",
			want: "wildcard host matcher cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseHostMatchers(tt.raw)
			if err == nil {
				t.Fatal("parseHostMatchers() error = nil, want invalid matcher syntax")
			}
			var cfgErr *networkPolicyConfigError
			if !errors.As(err, &cfgErr) {
				t.Fatalf("parseHostMatchers() error = %T, want *networkPolicyConfigError", err)
			}
			if cfgErr.reasonClass != "invalid_configuration" {
				t.Fatalf("reasonClass = %q, want invalid_configuration", cfgErr.reasonClass)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseHostMatchers() error = %v, want %q detail", err, tt.want)
			}
		})
	}
}

func TestParseHostMatchersAllowsTrailingDotSuffixSyntax(t *testing.T) {
	t.Parallel()

	matchers, err := parseHostMatchers("*.Example.COM., .Example.ORG.")
	if err != nil {
		t.Fatalf("parseHostMatchers() error = %v, want nil", err)
	}
	if len(matchers) != 2 {
		t.Fatalf("matchers len = %d, want 2", len(matchers))
	}
	if got := matchers[0]; got.suffix != ".example.com" || got.includeApex || got.exact != "" {
		t.Fatalf("wildcard matcher = %+v, want subdomain-only suffix .example.com", got)
	}
	if got := matchers[1]; got.suffix != ".example.org" || !got.includeApex || got.exact != "" {
		t.Fatalf("leading-dot matcher = %+v, want apex-inclusive suffix .example.org", got)
	}
	if !matchesHost("api.example.com", matchers) {
		t.Fatal("matchesHost(api.example.com) = false, want true")
	}
	if matchesHost("example.com", matchers) {
		t.Fatal("matchesHost(example.com) = true, want false for wildcard apex")
	}
	if !matchesHost("example.org", matchers) {
		t.Fatal("matchesHost(example.org) = false, want true for leading-dot apex")
	}
}
