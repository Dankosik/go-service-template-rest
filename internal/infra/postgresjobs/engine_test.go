package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

type engineStoreStub struct {
	claim      func(context.Context, ClaimOptions) (ClaimResult, error)
	resolve    func(context.Context, []AttemptIdentity) ([]ClaimResolution, error)
	finalize   func(context.Context, FinalizeInput) (PersistedTransition, error)
	renew      func(context.Context, []AttemptIdentity, time.Duration) ([]Renewal, error)
	candidates func(context.Context, RescueCandidateOptions) ([]RescueCandidate, error)
	rescue     func(context.Context, RescueInput) (PersistedTransition, error)
	observe    func(context.Context, []jobs.Revision) (Observation, error)
}

func (s *engineStoreStub) Renew(ctx context.Context, attempts []AttemptIdentity, lease time.Duration) ([]Renewal, error) {
	if s.renew != nil {
		return s.renew(ctx, attempts, lease)
	}
	return nil, nil
}

func (s *engineStoreStub) RescueCandidates(ctx context.Context, options RescueCandidateOptions) ([]RescueCandidate, error) {
	if s.candidates != nil {
		return s.candidates(ctx, options)
	}
	return nil, nil
}

func (s *engineStoreStub) Rescue(ctx context.Context, input RescueInput) (PersistedTransition, error) {
	if s.rescue != nil {
		return s.rescue(ctx, input)
	}
	return PersistedTransition{}, nil
}

func (s *engineStoreStub) Observe(ctx context.Context, keys []jobs.Revision) (Observation, error) {
	if s.observe != nil {
		return s.observe(ctx, keys)
	}
	return Observation{ObservedAt: time.Now(), Compatible: true}, nil
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
	if _, err := newEngine(&engineStoreStub{}, new(jobs.Registry), engineConfig()); !errors.Is(err, ErrConfig) {
		t.Fatalf("newEngine(empty registry) error = %v, want ErrConfig", err)
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
	t.Helper()
	return engineRegistryWithAttemptDuration(t, time.Minute, handler)
}

func engineRegistryWithAttemptDuration(t *testing.T, attemptDuration time.Duration, handler jobs.Handler[engineArgs]) *jobs.Registry {
	t.Helper()
	definition := engineDefinition(t, jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}, attemptDuration, 8)
	registry := new(jobs.Registry)
	if err := jobs.Register(registry, definition, handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return registry
}

func engineDefinition(t *testing.T, revision jobs.Revision, attemptDuration time.Duration, recoveryWave uint32) jobs.Definition[engineArgs] {
	t.Helper()
	return engineDefinitionWithAmbiguousAction(t, revision, attemptDuration, recoveryWave, jobs.AmbiguousEffectOutcomeUnknown)
}

func engineDefinitionWithAmbiguousAction(t *testing.T, revision jobs.Revision, attemptDuration time.Duration, recoveryWave uint32, action jobs.AmbiguousEffectAction) jobs.Definition[engineArgs] {
	t.Helper()
	definition, err := jobs.NewDefinition(jobs.DefinitionInput[engineArgs]{
		Revision:        revision,
		MaxPayloadBytes: 1024,
		Validate: func(args engineArgs) error {
			if args.Task == "" {
				return errors.New("task is required")
			}
			return nil
		},
		Policy: jobs.Policy{
			Effect:             jobs.EffectPolicy{AmbiguousAction: action},
			Retry:              jobs.RetryPolicy{MaxAttempts: 4, MaxElapsed: time.Hour, InitialBackoff: time.Second, MaxBackoff: time.Minute, HintPolicy: jobs.RetryHintPrefer, Jitter: jobs.JitterSHA256, JitterPermille: 100, MaxRecoveryWave: recoveryWave},
			Recovery:           jobs.RecoveryPolicy{Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved},
			MaxAttemptDuration: attemptDuration, TerminationEnvelope: time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("NewDefinition() error = %v", err)
	}
	return definition
}

func engineConfig() EngineConfig {
	return EngineConfig{WorkerID: "worker-1", MaxConcurrency: 1, LeaseDuration: time.Minute, ObservationInterval: time.Minute, ObservationMaxAge: 2 * time.Minute, DrainTimeout: time.Second}
}

func engineClaim() ClaimedAttempt {
	return ClaimedAttempt{
		Identity: jobs.AcceptanceIdentity{LogicalJobID: "job-1", ProducerScope: "orders", ProducerKey: "producer-1", OccurrenceScope: "orders", OccurrenceID: "occurrence-1", EffectScope: "orders", EffectKey: "effect-1"},
		Revision: jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}, Payload: []byte(`{"task":"send"}`),
		Attempt: AttemptIdentity{LogicalJobID: "job-1", AttemptGeneration: 1, WorkerID: "worker-1"}, AttemptNumber: 1,
		BudgetStartedAt: time.Now(), StartedAt: time.Now(), LeaseExpiresAt: time.Now().Add(time.Minute),
	}
}
