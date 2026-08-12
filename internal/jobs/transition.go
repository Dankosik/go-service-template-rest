package jobs

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"
)

var ErrInvalidTransition = errors.New("invalid jobs transition")

type State string

const (
	StateReady           State = "ready"
	StateScheduled       State = "scheduled"
	StateRetryWait       State = "retry_wait"
	StateRunning         State = "running"
	StateCancelRequested State = "cancel_requested"
	StateSucceeded       State = "succeeded"
	StateCancelled       State = "cancelled"
	StateExhausted       State = "exhausted"
	StatePermanent       State = "permanent"
	StatePoison          State = "poison"
	StateOutcomeUnknown  State = "outcome_unknown"
)

type OutcomeClass string

const (
	OutcomeSuccess   OutcomeClass = "success"
	OutcomeRetryable OutcomeClass = "retryable"
	OutcomePermanent OutcomeClass = "permanent"
	OutcomePoison    OutcomeClass = "poison"
	OutcomeTimeout   OutcomeClass = "timeout"
	OutcomeCancelled OutcomeClass = "cancelled"
	OutcomePanic     OutcomeClass = "panic"
	OutcomeLost      OutcomeClass = "lost"
	OutcomeUnknown   OutcomeClass = "unknown"
)

type EffectStatus string

const (
	EffectNone      EffectStatus = "none"
	EffectCompleted EffectStatus = "completed"
	EffectPartial   EffectStatus = "partial"
	EffectUnknown   EffectStatus = "unknown"
)

type AttemptFacts struct {
	LogicalJobID       LogicalJobID
	AttemptGeneration  uint64
	RecoveryGeneration uint64
	AttemptNumber      uint32
	Elapsed            time.Duration
	Outcome            OutcomeClass
	Effect             EffectStatus
	RetryHint          time.Duration
	RecoveryFrom       State
}

type Transition struct {
	State        State
	Delay        time.Duration
	AttemptsUsed uint32
	ElapsedUsed  time.Duration
	Outcome      OutcomeClass
	Effect       EffectStatus
}

func (d Definition[A]) Evaluate(facts AttemptFacts) (Transition, error) {
	if err := d.valid(); err != nil {
		return Transition{}, err
	}
	if err := facts.validate(); err != nil {
		return Transition{}, err
	}
	attempts, elapsed, err := d.policy.recoveryBudget(facts)
	if err != nil {
		return Transition{}, err
	}
	decision := Transition{
		AttemptsUsed: attempts,
		ElapsedUsed:  elapsed,
		Outcome:      facts.Outcome,
		Effect:       facts.Effect,
	}

	switch facts.Effect {
	case EffectCompleted:
		decision.State = StateSucceeded
		return decision, nil
	case EffectPartial, EffectUnknown:
		if d.policy.Effect.AmbiguousAction == AmbiguousEffectOutcomeUnknown {
			decision.State = StateOutcomeUnknown
			return decision, nil
		}
		return d.retryTransition(facts, decision)
	case EffectNone:
	}

	switch facts.Outcome {
	case OutcomeSuccess:
		decision.State = StateSucceeded
	case OutcomePermanent:
		decision.State = StatePermanent
	case OutcomePoison:
		decision.State = StatePoison
	case OutcomeCancelled:
		decision.State = StateCancelled
	case OutcomeUnknown:
		decision.State = StateOutcomeUnknown
	case OutcomeRetryable, OutcomeTimeout, OutcomePanic, OutcomeLost:
		return d.retryTransition(facts, decision)
	default:
		return Transition{}, fmt.Errorf("%w: unknown outcome", ErrInvalidTransition)
	}
	return decision, nil
}

func (d Definition[A]) retryTransition(facts AttemptFacts, decision Transition) (Transition, error) {
	if decision.AttemptsUsed >= d.policy.Retry.MaxAttempts || decision.ElapsedUsed >= d.policy.Retry.MaxElapsed {
		decision.State = StateExhausted
		return decision, nil
	}
	delay := exponentialBackoff(d.policy.Retry.InitialBackoff, d.policy.Retry.MaxBackoff, decision.AttemptsUsed)
	switch d.policy.Retry.HintPolicy {
	case RetryHintIgnore:
	case RetryHintPrefer:
		if facts.RetryHint > 0 {
			delay = min(facts.RetryHint, d.policy.Retry.MaxBackoff)
		}
	case RetryHintBackoffFloor:
		if facts.RetryHint > delay {
			delay = min(facts.RetryHint, d.policy.Retry.MaxBackoff)
		}
	}
	if d.policy.Retry.Jitter == JitterSHA256 {
		delay += deterministicJitter(d.revision, facts, delay, d.policy.Retry.JitterPermille)
		if delay < 0 {
			delay = 0
		}
	}
	if delay > d.policy.Retry.MaxElapsed-decision.ElapsedUsed {
		decision.State = StateExhausted
		decision.Delay = 0
		return decision, nil
	}
	decision.State = StateRetryWait
	decision.Delay = delay
	return decision, nil
}

func (p Policy) recoveryBudget(facts AttemptFacts) (uint32, time.Duration, error) {
	attempts, elapsed := facts.AttemptNumber, facts.Elapsed
	if facts.RecoveryFrom == "" {
		return attempts, elapsed, nil
	}
	if p.Recovery.Mode != RecoveryAllowed {
		return 0, 0, fmt.Errorf("%w: recovery is unavailable", ErrInvalidTransition)
	}
	if !slices.Contains(p.Recovery.Eligible, facts.RecoveryFrom) {
		return 0, 0, fmt.Errorf("%w: recovery source is not admitted", ErrInvalidTransition)
	}
	if p.Recovery.Attempts == BudgetReset {
		attempts = 1
	}
	if p.Recovery.Elapsed == BudgetReset {
		elapsed = 0
	}
	return attempts, elapsed, nil
}

func (f AttemptFacts) validate() error {
	if err := validateToken("logical_job_id", string(f.LogicalJobID)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTransition, err)
	}
	if f.AttemptGeneration == 0 || f.AttemptNumber == 0 {
		return fmt.Errorf("%w: attempt identity must be positive", ErrInvalidTransition)
	}
	if f.Elapsed < 0 || f.RetryHint < 0 {
		return fmt.Errorf("%w: elapsed and retry_hint must not be negative", ErrInvalidTransition)
	}
	if !f.Outcome.valid() {
		return fmt.Errorf("%w: outcome is required", ErrInvalidTransition)
	}
	if !f.Effect.valid() {
		return fmt.Errorf("%w: effect is required", ErrInvalidTransition)
	}
	return nil
}

func exponentialBackoff(initial, maximum time.Duration, attempts uint32) time.Duration {
	delay := initial
	for step := uint32(1); step < attempts && delay < maximum; step++ {
		if delay > maximum/2 {
			return maximum
		}
		delay *= 2
	}
	return min(delay, maximum)
}

func deterministicJitter(revision Revision, facts AttemptFacts, base time.Duration, permille uint16) time.Duration {
	hash := sha256.New()
	for _, value := range []string{revision.Kind, revision.ArgsVersion, revision.PolicyVersion, string(facts.LogicalJobID)} {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	var numbers [24]byte
	binary.BigEndian.PutUint64(numbers[0:8], facts.AttemptGeneration)
	binary.BigEndian.PutUint64(numbers[8:16], facts.RecoveryGeneration)
	binary.BigEndian.PutUint64(numbers[16:24], uint64(facts.AttemptNumber))
	_, _ = hash.Write(numbers[:])
	digest := hash.Sum(nil)
	units := int64(binary.BigEndian.Uint64(digest[:8])%uint64(2*permille+1)) - int64(permille)
	baseValue := int64(base)
	return time.Duration((baseValue/1000)*units + ((baseValue%1000)*units)/1000)
}

func (s State) manualRecoveryEligible() bool {
	return s == StateCancelled || s == StateExhausted || s == StatePermanent || s == StatePoison || s == StateOutcomeUnknown
}

func (v OutcomeClass) valid() bool {
	switch v {
	case OutcomeSuccess, OutcomeRetryable, OutcomePermanent, OutcomePoison, OutcomeTimeout, OutcomeCancelled, OutcomePanic, OutcomeLost, OutcomeUnknown:
		return true
	default:
		return false
	}
}

func (v EffectStatus) valid() bool {
	return v == EffectNone || v == EffectCompleted || v == EffectPartial || v == EffectUnknown
}
