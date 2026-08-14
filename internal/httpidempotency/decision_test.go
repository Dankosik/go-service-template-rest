package httpidempotency

import "testing"

func TestNewAttemptBindsValidatedIdentityAndFingerprint(t *testing.T) {
	t.Parallel()

	fingerprint, err := NewFingerprint("v1", []byte(`{"name":"widget"}`))
	if err != nil {
		t.Fatalf("NewFingerprint() error = %v", err)
	}
	attempt, err := NewAttempt(Scope{Authority: "tenant-1", OperationID: "CreateWidget", APIVersion: "v1"}, "key-1", fingerprint)
	if err != nil {
		t.Fatalf("NewAttempt() error = %v", err)
	}
	if attempt.Key != "key-1" || attempt.Fingerprint != fingerprint || attempt.Identity == ([32]byte{}) {
		t.Fatalf("NewAttempt() = %#v, want bound idempotency attempt", attempt)
	}
	if _, err := NewAttempt(Scope{}, "key-1", fingerprint); err == nil {
		t.Fatal("NewAttempt() error = nil, want invalid identity rejected")
	}
	if _, err := NewAttempt(Scope{Authority: "tenant-1", OperationID: "CreateWidget", APIVersion: "v1"}, "key-1", Fingerprint{}); err == nil {
		t.Fatal("NewAttempt() error = nil, want missing fingerprint version rejected")
	}
}

func TestDecisionValidateEnforcesReplayResultOwnership(t *testing.T) {
	t.Parallel()

	result := &Result{}
	for _, tc := range []struct {
		name  string
		value Decision
		valid bool
	}{
		{name: "execute", value: Decision{Outcome: OutcomeExecute}, valid: true},
		{name: "replay", value: Decision{Outcome: OutcomeReplay, Result: result}, valid: true},
		{name: "mismatch", value: Decision{Outcome: OutcomeMismatch}, valid: true},
		{name: "in progress", value: Decision{Outcome: OutcomeInProgress}, valid: true},
		{name: "expired", value: Decision{Outcome: OutcomeExpired}, valid: true},
		{name: "rate limited", value: Decision{Outcome: OutcomeRateLimited}, valid: true},
		{name: "unavailable", value: Decision{Outcome: OutcomeUnavailable}, valid: true},
		{name: "unknown", value: Decision{Outcome: OutcomeUnknown}, valid: true},
		{name: "result too large", value: Decision{Outcome: OutcomeResultTooLarge}, valid: true},
		{name: "integrity conflict", value: Decision{Outcome: OutcomeIntegrityConflict}, valid: true},
		{name: "execute with result", value: Decision{Outcome: OutcomeExecute, Result: result}},
		{name: "replay without result", value: Decision{Outcome: OutcomeReplay}},
		{name: "non replay with result", value: Decision{Outcome: OutcomeMismatch, Result: result}},
		{name: "unknown outcome", value: Decision{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.value.Validate() == nil; got != tc.valid {
				t.Fatalf("Decision.Validate() valid = %t, want %t", got, tc.valid)
			}
		})
	}
}
