package jobs

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type testArgs struct {
	Task     string            `json:"task"`
	Count    int               `json:"count"`
	Metadata map[string]string `json:"metadata"`
}

func TestJobsDefinition(t *testing.T) {
	input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"})
	if _, err := NewDefinition(input); err != nil {
		t.Fatalf("NewDefinition(complete) error = %v", err)
	}

	t.Run("missing slots", func(t *testing.T) {
		cases := []struct {
			name   string
			change func(*DefinitionInput[testArgs])
			want   string
		}{
			{name: "kind", change: func(i *DefinitionInput[testArgs]) { i.Revision.Kind = "" }, want: "kind"},
			{name: "args version", change: func(i *DefinitionInput[testArgs]) { i.Revision.ArgsVersion = "" }, want: "args_version"},
			{name: "policy version", change: func(i *DefinitionInput[testArgs]) { i.Revision.PolicyVersion = "" }, want: "policy_version"},
			{name: "payload limit", change: func(i *DefinitionInput[testArgs]) { i.MaxPayloadBytes = 0 }, want: "max_payload_bytes"},
			{name: "validator", change: func(i *DefinitionInput[testArgs]) { i.Validate = nil }, want: "validate"},
			{name: "producer scope", change: func(i *DefinitionInput[testArgs]) { i.Policy.Producer.Scope = "" }, want: "producer.scope"},
			{name: "recognition period", change: func(i *DefinitionInput[testArgs]) { i.Policy.Producer.RecognitionPeriod = 0 }, want: "producer.recognition_period"},
			{name: "effect authority", change: func(i *DefinitionInput[testArgs]) { i.Policy.Effect.Authority = "" }, want: "effect.authority"},
			{name: "duplicate tolerance", change: func(i *DefinitionInput[testArgs]) { i.Policy.Effect.DuplicateTolerance = "" }, want: "effect.duplicate_tolerance"},
			{name: "late result", change: func(i *DefinitionInput[testArgs]) { i.Policy.Effect.LateResultPrecedence = "" }, want: "effect.late_result_precedence"},
			{name: "ambiguous effect", change: func(i *DefinitionInput[testArgs]) { i.Policy.Effect.AmbiguousAction = "" }, want: "effect.ambiguous_action"},
			{name: "effect readback", change: func(i *DefinitionInput[testArgs]) { i.Policy.Effect.ReadbackAuthority = "" }, want: "effect.readback_authority"},
			{name: "attempts", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.MaxAttempts = 0 }, want: "retry.max_attempts"},
			{name: "elapsed", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.MaxElapsed = 0 }, want: "retry.max_elapsed"},
			{name: "initial backoff", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.InitialBackoff = 0 }, want: "retry.initial_backoff"},
			{name: "max backoff", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.MaxBackoff = 0 }, want: "retry.max_backoff"},
			{name: "hint policy", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.HintPolicy = "" }, want: "retry.hint_policy"},
			{name: "jitter", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.Jitter = "" }, want: "retry.jitter"},
			{name: "recovery wave", change: func(i *DefinitionInput[testArgs]) { i.Policy.Retry.MaxRecoveryWave = 0 }, want: "retry.max_recovery_wave"},
			{name: "recovery", change: func(i *DefinitionInput[testArgs]) { i.Policy.Recovery.Mode = "" }, want: "recovery.mode"},
			{name: "schedule", change: func(i *DefinitionInput[testArgs]) { i.Policy.Schedule = "" }, want: "schedule"},
			{name: "attempt duration", change: func(i *DefinitionInput[testArgs]) { i.Policy.MaxAttemptDuration = 0 }, want: "max_attempt_duration"},
			{name: "attempt cost", change: func(i *DefinitionInput[testArgs]) { i.Policy.MaxAttemptCost = 0 }, want: "max_attempt_cost"},
			{name: "useful duration", change: func(i *DefinitionInput[testArgs]) { i.Policy.MaxUsefulDuration = 0 }, want: "max_useful_duration"},
			{name: "termination", change: func(i *DefinitionInput[testArgs]) { i.Policy.TerminationEnvelope = 0 }, want: "termination_envelope"},
			{name: "classification", change: func(i *DefinitionInput[testArgs]) { i.Policy.Data.Classification = "" }, want: "data.classification"},
			{name: "redaction", change: func(i *DefinitionInput[testArgs]) { i.Policy.Data.Redaction = "" }, want: "data.redaction"},
			{name: "retention", change: func(i *DefinitionInput[testArgs]) { i.Policy.Data.Retention = "" }, want: "data.retention"},
			{name: "deletion", change: func(i *DefinitionInput[testArgs]) { i.Policy.Data.Deletion = "" }, want: "data.deletion"},
			{name: "operator roles", change: func(i *DefinitionInput[testArgs]) { i.Policy.Data.OperatorRoles = "" }, want: "data.operator_roles"},
			{name: "operator", change: func(i *DefinitionInput[testArgs]) { i.Policy.Operator = "" }, want: "operator"},
			{name: "work class", change: func(i *DefinitionInput[testArgs]) { i.Policy.WorkClass = "" }, want: "work_class"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				candidate := input
				tc.change(&candidate)
				_, err := NewDefinition(candidate)
				if !errors.Is(err, ErrInvalidDefinition) || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("NewDefinition() error = %v, want ErrInvalidDefinition naming %q", err, tc.want)
				}
				if strings.Contains(err.Error(), "email") || strings.Contains(err.Error(), "private") {
					t.Fatalf("diagnostic exposes policy value: %v", err)
				}
			})
		}
	})

	for _, tc := range []struct {
		name   string
		change func(*DefinitionInput[testArgs])
	}{
		{name: "oversized payload", change: func(i *DefinitionInput[testArgs]) { i.MaxPayloadBytes = MaxPayloadBytes + 1 }},
		{name: "periodic", change: func(i *DefinitionInput[testArgs]) { i.Policy.Schedule = ScheduleMode("periodic") }},
		{name: "second class", change: func(i *DefinitionInput[testArgs]) { i.Policy.WorkClass = WorkClass("priority") }},
		{name: "operator capability", change: func(i *DefinitionInput[testArgs]) { i.Policy.Operator = OperatorMode("enabled") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := input
			tc.change(&candidate)
			if _, err := NewDefinition(candidate); !errors.Is(err, ErrInvalidDefinition) {
				t.Fatalf("NewDefinition() error = %v, want ErrInvalidDefinition", err)
			}
		})
	}
}

func testDefinitionInput(revision Revision) DefinitionInput[testArgs] {
	return DefinitionInput[testArgs]{
		Revision:        revision,
		MaxPayloadBytes: 1024,
		Validate: func(args testArgs) error {
			if args.Task == "" {
				return errors.New("task is required")
			}
			if args.Count < 1 {
				return errors.New("count must be positive")
			}
			return nil
		},
		Policy: testPolicy(),
	}
}

func testDefinition(t *testing.T, revision Revision) Definition[testArgs] {
	t.Helper()
	definition, err := NewDefinition(testDefinitionInput(revision))
	if err != nil {
		t.Fatalf("NewDefinition() error = %v", err)
	}
	return definition
}

func testPolicy() Policy {
	return Policy{
		Producer: ProducerPolicy{Scope: "feature-operation", RecognitionPeriod: 30 * 24 * time.Hour},
		Effect: EffectPolicy{
			Authority: EffectConditionalWrite, DuplicateTolerance: "same key is harmless",
			LateResultPrecedence: "effect ledger wins", AmbiguousAction: AmbiguousEffectOutcomeUnknown,
			ReadbackAuthority: "effect ledger",
		},
		Retry: RetryPolicy{
			MaxAttempts: 4, MaxElapsed: 24 * time.Hour, InitialBackoff: time.Second,
			MaxBackoff: time.Minute, HintPolicy: RetryHintPrefer, Jitter: JitterSHA256,
			JitterPermille: 100, MaxRecoveryWave: 8,
		},
		Recovery: RecoveryPolicy{
			Mode: RecoveryUnavailable, Attempts: BudgetPreserved, Elapsed: BudgetPreserved,
		},
		Schedule: ScheduleOneOff, MaxAttemptDuration: time.Minute, MaxAttemptCost: 1,
		MaxUsefulDuration: time.Hour, TerminationEnvelope: 2 * time.Minute,
		Data: DataPolicy{
			Classification: "private", Redaction: "omit payload", Retention: "explicit deletion only",
			Deletion: "disabled", OperatorRoles: "none",
		},
		Operator: OperatorUnavailable, WorkClass: WorkClassNeutral,
	}
}

func testIdentity() AcceptanceIdentity {
	return AcceptanceIdentity{
		LogicalJobID: "job-1", ProducerScope: "orders", ProducerKey: "producer-1",
		OccurrenceScope: "orders", OccurrenceID: "occurrence-1",
		EffectScope: "orders", EffectKey: "effect-1",
	}
}

func testAvailableAt() time.Time {
	return time.Date(2026, time.August, 12, 12, 0, 0, 123456789, time.UTC)
}

func requireError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
