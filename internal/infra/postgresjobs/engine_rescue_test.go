package postgresjobs

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineRescueBoundsCandidatesAndUsesStoredRevision(t *testing.T) {
	reader := telemetrytest.InstallManualReader(t)
	candidate := RescueCandidate{Attempt: engineClaim().Attempt, Revision: jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}, State: jobs.StateRunning, AttemptNumber: 1, Elapsed: time.Second}
	var limit, rescues int
	store := &engineStoreStub{
		candidates: func(_ context.Context, got RescueCandidateOptions) ([]RescueCandidate, error) {
			limit = got.Limit
			if len(got.Limits) != 1 || got.Limits[0].MaxRecoveryWave != 8 {
				t.Fatalf("RescueCandidateOptions = %+v, want registered wave", got)
			}
			return []RescueCandidate{candidate}, nil
		},
		rescue: func(_ context.Context, input RescueInput) (PersistedTransition, error) {
			rescues++
			if input.Attempt != candidate.Attempt || input.Transition.Outcome != jobs.OutcomeLost || input.Transition.Effect != jobs.EffectUnknown {
				t.Fatalf("RescueInput = %+v", input)
			}
			status := TransitionApplied
			if rescues > 1 {
				status = TransitionRepeated
			}
			return PersistedTransition{Status: status}, nil
		},
	}
	handler := func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }
	registry := new(jobs.Registry)
	if err := jobs.Register(registry, engineDefinitionWithAmbiguousAction(t, candidate.Revision, time.Minute, 8, jobs.AmbiguousEffectRetry), handler); err != nil {
		t.Fatal(err)
	}
	engine, err := newEngine(store, registry, engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.rescue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if limit != engine.config.MaxConcurrency || rescues != 1 {
		t.Fatalf("limit/rescues = %d/%d, want %d/1", limit, rescues, engine.config.MaxConcurrency)
	}
	assertJobsEvent(t, reader, "rescue", jobs.OutcomeLost)
	assertJobsEvent(t, reader, "retry", jobs.OutcomeLost)
	if err := engine.rescue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if value := jobsEventCount(t, reader, "rescue", jobs.OutcomeLost); value != 1 {
		t.Fatalf("repeated rescue events = %d, want 1", value)
	}
}

func TestEngineRescueHonorsDefinitionRecoveryWave(t *testing.T) {
	emailRevision := engineClaim().Revision
	reportRevision := jobs.Revision{Kind: "report", ArgsVersion: "v1", PolicyVersion: "p1"}
	candidates := make([]RescueCandidate, 0, 11)
	for index := range 9 {
		claim := engineClaim()
		claim.Attempt.LogicalJobID = jobs.LogicalJobID(fmt.Sprintf("email-%d", index))
		candidates = append(candidates, RescueCandidate{
			Attempt: claim.Attempt, Revision: emailRevision, State: jobs.StateRunning,
			AttemptNumber: 1, Elapsed: time.Second,
		})
	}
	for index := range 2 {
		claim := engineClaim()
		claim.Attempt.LogicalJobID = jobs.LogicalJobID(fmt.Sprintf("report-%d", index))
		candidates = append(candidates, RescueCandidate{
			Attempt: claim.Attempt, Revision: reportRevision, State: jobs.StateRunning,
			AttemptNumber: 1, Elapsed: time.Second,
		})
	}
	rescued := make(map[jobs.LogicalJobID]bool)
	store := &engineStoreStub{
		candidates: func(_ context.Context, options RescueCandidateOptions) ([]RescueCandidate, error) {
			if options.Limit != len(candidates) || len(options.Limits) != 2 {
				t.Fatalf("RescueCandidateOptions = %+v", options)
			}
			return append(candidates[:8:8], candidates[9]), nil
		},
		rescue: func(_ context.Context, input RescueInput) (PersistedTransition, error) {
			rescued[input.Attempt.LogicalJobID] = true
			return PersistedTransition{Status: TransitionApplied}, nil
		},
	}
	cfg := engineConfig()
	cfg.MaxConcurrency = len(candidates)
	handler := func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }
	registry := engineRegistry(t, handler)
	if err := jobs.Register(registry, engineDefinition(t, reportRevision, time.Minute, 1), handler); err != nil {
		t.Fatal(err)
	}
	engine, err := newEngine(store, registry, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.rescue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(rescued) != 9 || rescued["email-8"] || !rescued["report-0"] || rescued["report-1"] {
		t.Fatalf("rescued attempts = %v, want email wave 8 and report wave 1", rescued)
	}
}

func TestEngineRescueIncludesLocalTimeAfterDatabaseObservation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		candidate := RescueCandidate{
			Attempt: engineClaim().Attempt, Revision: engineClaim().Revision,
			State: jobs.StateRunning, AttemptNumber: 1, Elapsed: time.Second,
		}
		var transition jobs.Transition
		store := &engineStoreStub{
			candidates: func(context.Context, RescueCandidateOptions) ([]RescueCandidate, error) {
				time.Sleep(time.Second)
				return []RescueCandidate{candidate}, nil
			},
			rescue: func(_ context.Context, input RescueInput) (PersistedTransition, error) {
				transition = input.Transition
				return PersistedTransition{Status: TransitionApplied}, nil
			},
		}
		definitionInput := jobs.DefinitionInput[engineArgs]{
			Revision: candidate.Revision, MaxPayloadBytes: 1024,
			Validate: func(engineArgs) error { return nil },
			Policy: jobs.Policy{
				Effect: jobs.EffectPolicy{AmbiguousAction: jobs.AmbiguousEffectRetry},
				Retry: jobs.RetryPolicy{
					MaxAttempts: 4, MaxElapsed: 5 * time.Second,
					InitialBackoff: 4 * time.Second, MaxBackoff: 4 * time.Second,
					HintPolicy: jobs.RetryHintIgnore, Jitter: jobs.JitterNone, MaxRecoveryWave: 1,
				},
				Recovery: jobs.RecoveryPolicy{
					Mode: jobs.RecoveryUnavailable, Attempts: jobs.BudgetPreserved, Elapsed: jobs.BudgetPreserved,
				},
				MaxAttemptDuration: time.Minute, TerminationEnvelope: time.Minute,
			},
		}
		definition, err := jobs.NewDefinition(definitionInput)
		if err != nil {
			t.Fatal(err)
		}
		registry := new(jobs.Registry)
		if err := jobs.Register(registry, definition, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			return jobs.HandlerResult{}
		}); err != nil {
			t.Fatal(err)
		}
		engine, err := newEngine(store, registry, engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.rescue(context.Background()); err != nil {
			t.Fatal(err)
		}
		if transition.State != jobs.StateExhausted || transition.ElapsedUsed != 2*time.Second {
			t.Fatalf("rescue transition = %+v, want exhausted with 2s elapsed", transition)
		}
	})
}
