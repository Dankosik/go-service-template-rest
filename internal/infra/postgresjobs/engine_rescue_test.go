package postgresjobs

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineRescueBoundsCandidatesAndUsesStoredRevision(t *testing.T) {
	reader := telemetrytest.InstallManualReader(t)
	candidate := RescueCandidate{Attempt: engineClaim().Attempt, Revision: jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}, State: jobs.StateRunning, AttemptNumber: 1, Elapsed: time.Second}
	var limit, rescues int
	store := &engineStoreStub{
		candidates: func(_ context.Context, got int) ([]RescueCandidate, error) {
			limit = got
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
	engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
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
	if err := engine.rescue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if value := jobsEventCount(t, reader, "rescue", jobs.OutcomeLost); value != 1 {
		t.Fatalf("repeated rescue events = %d, want 1", value)
	}
}
