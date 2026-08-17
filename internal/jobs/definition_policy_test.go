package jobs

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDefinitionRejectsUnsafeRetryAndRecoveryPolicies(t *testing.T) {
	for _, test := range []struct {
		name   string
		update func(*DefinitionInput[testArgs])
	}{
		{name: "max backoff below initial", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.Retry.MaxBackoff = input.Policy.Retry.InitialBackoff - time.Nanosecond
		}},
		{name: "unknown hint policy", update: func(input *DefinitionInput[testArgs]) { input.Policy.Retry.HintPolicy = "other" }},
		{name: "no jitter with jitter allowance", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.Retry.Jitter = JitterNone
			input.Policy.Retry.JitterPermille = 1
		}},
		{name: "SHA jitter with zero allowance", update: func(input *DefinitionInput[testArgs]) { input.Policy.Retry.JitterPermille = 0 }},
		{name: "SHA jitter above maximum", update: func(input *DefinitionInput[testArgs]) { input.Policy.Retry.JitterPermille = 1001 }},
		{name: "unavailable recovery with eligible state", update: func(input *DefinitionInput[testArgs]) { input.Policy.Recovery.Eligible = []State{StateExhausted} }},
		{name: "allowed recovery without eligible states", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.Recovery = RecoveryPolicy{Mode: RecoveryAllowed, Attempts: BudgetPreserved, Elapsed: BudgetPreserved}
		}},
		{name: "allowed recovery resets without policy", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.Recovery = RecoveryPolicy{Mode: RecoveryAllowed, Eligible: []State{StateExhausted}}
		}},
		{name: "allowed recovery from non terminal state", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.Recovery = RecoveryPolicy{Mode: RecoveryAllowed, Eligible: []State{StateRunning}, Attempts: BudgetPreserved, Elapsed: BudgetPreserved}
		}},
		{name: "allowed recovery duplicate state", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.Recovery = RecoveryPolicy{Mode: RecoveryAllowed, Eligible: []State{StateExhausted, StateExhausted}, Attempts: BudgetPreserved, Elapsed: BudgetPreserved}
		}},
		{name: "termination before attempt ceiling", update: func(input *DefinitionInput[testArgs]) {
			input.Policy.TerminationEnvelope = input.Policy.MaxAttemptDuration - time.Nanosecond
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "policy-test"})
			test.update(&input)
			if _, err := NewDefinition(input); !errors.Is(err, ErrInvalidDefinition) {
				t.Fatalf("NewDefinition() error = %v, want ErrInvalidDefinition", err)
			}
		})
	}
}

func TestDefinitionRejectsMalformedOrInadmissiblePayloads(t *testing.T) {
	definition := testDefinition(t, Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "payload-test"})
	for _, test := range []struct {
		name    string
		payload []byte
	}{
		{name: "invalid JSON", payload: []byte(`{`)},
		{name: "unknown field", payload: []byte(`{"task":"send","count":1,"private":true}`)},
		{name: "trailing JSON", payload: []byte(`{"task":"send","count":1} {}`)},
		{name: "invalid arguments", payload: []byte(`{"task":"send","count":0}`)},
		{name: "over maximum", payload: []byte(strings.Repeat("x", 1025))},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := definition.Decode(test.payload); !errors.Is(err, ErrInvalidPayload) {
				t.Fatalf("Decode() error = %v, want ErrInvalidPayload", err)
			}
		})
	}
	if _, err := definition.Prepare(testArgs{Task: "send", Count: 1}, testIdentity(), time.Time{}); !errors.Is(err, ErrInvalidAcceptance) {
		t.Fatalf("Prepare(zero available_at) error = %v, want ErrInvalidAcceptance", err)
	}
	if _, err := definition.Prepare(testArgs{Task: "", Count: 1}, testIdentity(), testAvailableAt()); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("Prepare(invalid arguments) error = %v, want ErrInvalidPayload", err)
	}
	if _, err := definition.Prepare(testArgs{Task: "send", Count: 1, Metadata: map[string]string{"large": strings.Repeat("x", 2048)}}, testIdentity(), testAvailableAt()); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("Prepare(over maximum) error = %v, want ErrInvalidPayload", err)
	}
	var invalid Definition[testArgs]
	if _, err := invalid.Decode([]byte(`{"task":"send","count":1}`)); !errors.Is(err, ErrInvalidDefinition) {
		t.Fatalf("Decode(unconstructed definition) error = %v, want ErrInvalidDefinition", err)
	}
}
