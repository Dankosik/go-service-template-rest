package oidcjwt

import "testing"

func TestPolicyRejectsUnknownProfile(t *testing.T) {
	t.Parallel()
	if _, err := NewPolicy(PolicyInput{Issuer: testIssuer, Audience: testAudience, TokenProfile: "strict"}); err == nil {
		t.Fatal("NewPolicy() accepted an unknown token profile")
	}
}
