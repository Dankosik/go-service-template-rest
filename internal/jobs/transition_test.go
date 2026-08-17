package jobs

import (
	"errors"
	"testing"
	"time"
)

func TestJobsTransition(t *testing.T) {
	t.Parallel()
	definition := testDefinition(t, Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"})
	base := AttemptFacts{
		LogicalJobID: "job-1", AttemptGeneration: 2, RecoveryGeneration: 1,
		AttemptNumber: 2, Elapsed: time.Minute, Outcome: OutcomeRetryable, Effect: EffectNone,
	}

	t.Run("ambiguous effect follows explicit retry policy", func(t *testing.T) {
		t.Parallel()
		input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "ambiguous-retry"})
		input.Policy.Effect.AmbiguousAction = AmbiguousEffectRetry
		retryDefinition, err := NewDefinition(input)
		requireError(t, err)
		facts := base
		facts.Outcome = OutcomeSuccess
		facts.Effect = EffectUnknown
		got, err := retryDefinition.Evaluate(facts)
		requireError(t, err)
		if got.State != StateRetryWait {
			t.Fatalf("ambiguous effect state = %q, want %q", got.State, StateRetryWait)
		}
	})

	first, err := definition.Evaluate(base)
	requireError(t, err)
	second, err := definition.Evaluate(base)
	requireError(t, err)
	if first != second {
		t.Fatalf("replay changed decision: first=%+v second=%+v", first, second)
	}
	if first.State != StateRetryWait || first.Delay != 1_958_000_000 {
		t.Fatalf("deterministic retry = %+v, want retry_wait delay 1.958s", first)
	}

	for _, tc := range []struct {
		name   string
		change func(*AttemptFacts)
		state  State
	}{
		{name: "success", change: func(f *AttemptFacts) { f.Outcome = OutcomeSuccess }, state: StateSucceeded},
		{name: "permanent", change: func(f *AttemptFacts) { f.Outcome = OutcomePermanent }, state: StatePermanent},
		{name: "poison", change: func(f *AttemptFacts) { f.Outcome = OutcomePoison }, state: StatePoison},
		{name: "cancelled no effect", change: func(f *AttemptFacts) { f.Outcome = OutcomeCancelled }, state: StateCancelled},
		{name: "unknown", change: func(f *AttemptFacts) { f.Outcome = OutcomeUnknown }, state: StateOutcomeUnknown},
		{name: "partial effect", change: func(f *AttemptFacts) { f.Effect = EffectPartial }, state: StateOutcomeUnknown},
		{name: "completed effect wins cancellation", change: func(f *AttemptFacts) { f.Outcome = OutcomeCancelled; f.Effect = EffectCompleted }, state: StateSucceeded},
		{name: "timeout", change: func(f *AttemptFacts) { f.Outcome = OutcomeTimeout }, state: StateRetryWait},
		{name: "panic", change: func(f *AttemptFacts) { f.Outcome = OutcomePanic }, state: StateRetryWait},
		{name: "lost attempt", change: func(f *AttemptFacts) { f.Outcome = OutcomeLost }, state: StateRetryWait},
		{name: "attempt exhausted", change: func(f *AttemptFacts) { f.AttemptNumber = 4 }, state: StateExhausted},
		{name: "age exhausted", change: func(f *AttemptFacts) { f.Elapsed = 24 * time.Hour }, state: StateExhausted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			facts := base
			tc.change(&facts)
			got, err := definition.Evaluate(facts)
			requireError(t, err)
			if got.State != tc.state {
				t.Fatalf("Evaluate() state = %q, want %q", got.State, tc.state)
			}
		})
	}

	t.Run("retry hint precedence and cap", func(t *testing.T) {
		t.Parallel()
		input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "no-jitter"})
		input.Policy.Retry.Jitter = JitterNone
		input.Policy.Retry.JitterPermille = 0
		noJitter, err := NewDefinition(input)
		requireError(t, err)
		facts := base
		facts.RetryHint = 2 * time.Hour
		got, err := noJitter.Evaluate(facts)
		requireError(t, err)
		if got.Delay != time.Minute {
			t.Fatalf("retry delay = %s, want cap %s", got.Delay, time.Minute)
		}
	})

	t.Run("retry hint policies", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			policy    RetryHintPolicy
			retryHint time.Duration
			want      time.Duration
		}{
			{name: "ignore", policy: RetryHintIgnore, retryHint: 30 * time.Second, want: 2 * time.Second},
			{name: "prefer", policy: RetryHintPrefer, retryHint: 500 * time.Millisecond, want: 500 * time.Millisecond},
			{name: "backoff floor", policy: RetryHintBackoffFloor, retryHint: 30 * time.Second, want: 30 * time.Second},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: tc.name})
				input.Policy.Retry.HintPolicy = tc.policy
				input.Policy.Retry.Jitter = JitterNone
				input.Policy.Retry.JitterPermille = 0
				definition, err := NewDefinition(input)
				requireError(t, err)
				facts := base
				facts.RetryHint = tc.retryHint
				got, err := definition.Evaluate(facts)
				requireError(t, err)
				if got.Delay != tc.want {
					t.Fatalf("Evaluate().Delay = %s, want %s", got.Delay, tc.want)
				}
			})
		}
	})

	t.Run("recovery reset", func(t *testing.T) {
		t.Parallel()
		input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "recovery"})
		input.Policy.Recovery = RecoveryPolicy{
			Mode: RecoveryAllowed, Eligible: []State{StateExhausted},
			Attempts: BudgetReset, Elapsed: BudgetReset,
		}
		recoveryDefinition, err := NewDefinition(input)
		requireError(t, err)
		facts := base
		facts.RecoveryFrom = StateExhausted
		facts.AttemptNumber = input.Policy.Retry.MaxAttempts
		facts.Elapsed = input.Policy.Retry.MaxElapsed
		got, err := recoveryDefinition.Evaluate(facts)
		requireError(t, err)
		if got.State != StateRetryWait || got.AttemptsUsed != 1 || got.ElapsedUsed != 0 {
			t.Fatalf("recovery decision = %+v", got)
		}
		if _, err := definition.Evaluate(facts); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("unavailable recovery error = %v", err)
		}
		facts.RecoveryFrom = StateSucceeded
		if _, err := recoveryDefinition.Evaluate(facts); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("ineligible recovery source error = %v", err)
		}
	})

	t.Run("recovery preserves budgets", func(t *testing.T) {
		t.Parallel()
		input := testDefinitionInput(Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "recovery-preserved"})
		input.Policy.Recovery = RecoveryPolicy{
			Mode: RecoveryAllowed, Eligible: []State{StateExhausted},
			Attempts: BudgetPreserved, Elapsed: BudgetPreserved,
		}
		definition, err := NewDefinition(input)
		requireError(t, err)
		facts := base
		facts.RecoveryFrom = StateExhausted
		got, err := definition.Evaluate(facts)
		requireError(t, err)
		if got.State != StateRetryWait || got.AttemptsUsed != facts.AttemptNumber || got.ElapsedUsed != facts.Elapsed {
			t.Fatalf("preserved recovery decision = %+v", got)
		}
	})

	for _, facts := range []AttemptFacts{
		{LogicalJobID: "job-1", AttemptGeneration: 1, AttemptNumber: 1, Outcome: "future", Effect: EffectNone},
		{LogicalJobID: "job-1", AttemptGeneration: 1, AttemptNumber: 1, Outcome: OutcomeSuccess, Effect: "future"},
		{LogicalJobID: "job-1", AttemptGeneration: 1, AttemptNumber: 0, Outcome: OutcomeSuccess, Effect: EffectNone},
	} {
		if _, err := definition.Evaluate(facts); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("Evaluate(invalid %+v) error = %v", facts, err)
		}
	}
}

func TestJobsTransitionRejectsImpossiblePersistedFacts(t *testing.T) {
	t.Parallel()
	valid := Transition{State: StateSucceeded, AttemptsUsed: 1, Outcome: OutcomeSuccess, Effect: EffectNone}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid) error = %v", err)
	}
	for _, transition := range []Transition{
		{State: StateSucceeded, AttemptsUsed: 1, Outcome: OutcomePoison, Effect: EffectNone},
		{State: StateCancelled, AttemptsUsed: 1, Outcome: OutcomeCancelled, Effect: EffectCompleted},
		{State: StateRetryWait, AttemptsUsed: 1, Outcome: OutcomeSuccess, Effect: EffectNone},
		{State: StateRunning, AttemptsUsed: 1, Outcome: OutcomeSuccess, Effect: EffectNone},
	} {
		if err := transition.Validate(); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("Validate(%+v) error = %v, want ErrInvalidTransition", transition, err)
		}
	}
}
