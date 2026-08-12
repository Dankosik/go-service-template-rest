package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

type engineStoreStub struct {
	claim    func(context.Context, ClaimOptions) (ClaimResult, error)
	resolve  func(context.Context, []AttemptIdentity) ([]ClaimResolution, error)
	finalize func(context.Context, FinalizeInput) (PersistedTransition, error)
}

func (s *engineStoreStub) Claim(ctx context.Context, options ClaimOptions) (ClaimResult, error) {
	if s.claim == nil {
		return ClaimResult{}, nil
	}
	return s.claim(ctx, options)
}

func (s *engineStoreStub) ResolveClaims(ctx context.Context, attempts []AttemptIdentity) ([]ClaimResolution, error) {
	if s.resolve == nil {
		return nil, nil
	}
	return s.resolve(ctx, attempts)
}

func (s *engineStoreStub) Finalize(ctx context.Context, input FinalizeInput) (PersistedTransition, error) {
	if s.finalize == nil {
		return PersistedTransition{}, nil
	}
	return s.finalize(ctx, input)
}

func TestEngineConstructionAndFacts(t *testing.T) {
	registry := engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
		return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
	})
	if _, err := NewEngine(nil, registry, engineConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewEngine(nil) error = %v, want ErrConfig", err)
	}
	engine, err := newEngine(&engineStoreStub{}, registry, engineConfig())
	if err != nil {
		t.Fatalf("newEngine() error = %v", err)
	}
	if got := engine.Facts(); !got.ClaimAdmissionOpen || !got.Compatible || got.InFlight != 0 {
		t.Fatalf("Facts() = %+v, want open compatible empty", got)
	}
}

type engineArgs struct {
	Task string `json:"task"`
}

func engineRegistry(t *testing.T, handler jobs.Handler[engineArgs]) *jobs.Registry {
	return engineRegistryWithAttemptDuration(t, time.Minute, handler)
}

func engineRegistryWithAttemptDuration(t *testing.T, attemptDuration time.Duration, handler jobs.Handler[engineArgs]) *jobs.Registry {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[engineArgs]{
		Revision:        jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"},
		MaxPayloadBytes: 1024,
		Validate: func(args engineArgs) error {
			if args.Task == "" {
				return errors.New("task is required")
			}
			return nil
		},
		Policy: jobs.Policy{
			Producer: jobs.ProducerPolicy{Scope: "feature-operation", RecognitionPeriod: time.Hour},
			Effect:   jobs.EffectPolicy{Authority: jobs.EffectConditionalWrite, DuplicateTolerance: "same key is harmless", LateResultPrecedence: "effect ledger wins", AmbiguousAction: jobs.AmbiguousEffectOutcomeUnknown, ReadbackAuthority: "effect ledger"},
			Retry:    jobs.RetryPolicy{MaxAttempts: 4, MaxElapsed: time.Hour, InitialBackoff: time.Second, MaxBackoff: time.Minute, HintPolicy: jobs.RetryHintPrefer, Jitter: jobs.JitterSHA256, JitterPermille: 100, MaxRecoveryWave: 8},
			Recovery: jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
			Schedule: jobs.ScheduleOneOff, MaxAttemptDuration: attemptDuration, MaxAttemptCost: 1, MaxUsefulDuration: time.Hour, TerminationEnvelope: time.Minute,
			Data:     jobs.DataPolicy{Classification: "private", Redaction: "omit payload", Retention: "explicit deletion only", Deletion: "disabled", OperatorRoles: "none"},
			Operator: jobs.OperatorUnavailable, WorkClass: jobs.WorkClassNeutral,
		},
	})
	if err != nil {
		t.Fatalf("NewDefinition() error = %v", err)
	}
	registry := jobs.NewRegistry()
	if err := jobs.Register(registry, definition, handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return registry
}

func engineConfig() EngineConfig {
	return EngineConfig{WorkerID: "worker-1", MaxConcurrency: 1, LeaseDuration: time.Minute}
}

func engineClaim() ClaimedAttempt {
	return ClaimedAttempt{
		Identity: jobs.AcceptanceIdentity{LogicalJobID: "job-1", ProducerScope: "orders", ProducerKey: "producer-1", OccurrenceScope: "orders", OccurrenceID: "occurrence-1", EffectScope: "orders", EffectKey: "effect-1"},
		Revision: jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}, Payload: []byte(`{"task":"send"}`),
		Attempt: AttemptIdentity{LogicalJobID: "job-1", AttemptGeneration: 1, WorkerID: "worker-1"}, AttemptNumber: 1,
		BudgetStartedAt: time.Now(), StartedAt: time.Now(), LeaseExpiresAt: time.Now().Add(time.Minute),
	}
}
