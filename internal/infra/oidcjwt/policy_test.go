package oidcjwt

import "testing"

func TestPolicyProfiles(t *testing.T) {
	t.Parallel()

	for _, profile := range []string{"resource-server", "rfc9068"} {
		policy, err := NewPolicy(PolicyInput{Issuer: testIssuer, Audience: testAudience, TokenProfile: profile})
		if err != nil {
			t.Fatalf("NewPolicy(%q) error = %v", profile, err)
		}
		if got := policy.strictRFC9068(); got != (profile == "rfc9068") {
			t.Fatalf("strictRFC9068(%q) = %v", profile, got)
		}
	}

	if _, err := NewPolicy(PolicyInput{Issuer: testIssuer, Audience: testAudience, TokenProfile: "strict"}); err == nil {
		t.Fatal("NewPolicy() accepted an unknown token profile")
	}
}
