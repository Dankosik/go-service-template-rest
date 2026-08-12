package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineClaimRegistersKnownCommitBeforeHandlerStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claimStarted := make(chan struct{})
		releaseClaim := make(chan struct{})
		started := make(chan struct{})
		allowFinish := make(chan struct{})
		registry := engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			close(started)
			<-allowFinish
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		})
		claim := engineClaim()
		store := &engineStoreStub{claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
			close(claimStarted)
			<-releaseClaim
			return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
		}}
		engine, err := newEngine(store, registry, engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		run := make(chan error, 1)
		go func() { run <- engine.Run(context.Background()) }()
		<-claimStarted
		acknowledged := make(chan EngineFacts, 1)
		go func() { acknowledged <- engine.Facts() }()
		synctest.Wait()
		select {
		case facts := <-acknowledged:
			t.Fatalf("drain acknowledgement saw %+v before claim registration", facts)
		default:
		}
		close(releaseClaim)
		if err := <-run; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		<-started
		if got := (<-acknowledged).InFlight; got != 1 {
			t.Fatalf("in-flight handlers = %d, want registered handler", got)
		}
		close(allowFinish)
		synctest.Wait()
	})
}

func TestEngineClaimUnknownCommitStartsOnlyWriterResolvedAttempt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{}, 1)
		registry := engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			started <- struct{}{}
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		})
		known, unknown := engineClaim(), engineClaim()
		unknown.Attempt.AttemptGeneration = 2
		store := &engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				return ClaimResult{Attempts: []ClaimedAttempt{known, unknown}}, postgres.ErrCommitUnknown
			},
			resolve: func(_ context.Context, attempts []AttemptIdentity) ([]ClaimResolution, error) {
				if len(attempts) != 2 {
					t.Fatalf("ResolveClaims attempts = %d, want 2", len(attempts))
				}
				return []ClaimResolution{{Attempt: known.Attempt, Committed: true}, {Attempt: unknown.Attempt}}, nil
			},
		}
		engine, err := newEngine(store, registry, engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		<-started
		synctest.Wait()
		select {
		case <-started:
			t.Fatal("unknown claim started a handler")
		default:
		}
	})
}

func TestEngineClaimCoverageFaultClosesAdmission(t *testing.T) {
	registry := engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
		return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
	})
	store := &engineStoreStub{claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
		return ClaimResult{}, jobs.ErrUnsupportedRevision
	}}
	engine, err := newEngine(store, registry, engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(context.Background()); !errors.Is(err, jobs.ErrUnsupportedRevision) {
		t.Fatalf("Run() error = %v, want coverage fault", err)
	}
	if facts := engine.Facts(); facts.ClaimAdmissionOpen || facts.Compatible {
		t.Fatalf("Facts() after coverage fault = %+v, want closed incompatible", facts)
	}
}

func TestEngineClaimCommittedUnsupportedRevisionClosesAdmission(t *testing.T) {
	claim := engineClaim()
	claim.Revision.PolicyVersion = "p2"
	started := make(chan struct{}, 1)
	store := &engineStoreStub{claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
		return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
	}}
	engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
		started <- struct{}{}
		return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
	}), engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(context.Background()); !errors.Is(err, jobs.ErrUnsupportedRevision) {
		t.Fatalf("Run() error = %v, want coverage fault", err)
	}
	if facts := engine.Facts(); facts.ClaimAdmissionOpen || facts.Compatible || facts.InFlight != 0 {
		t.Fatalf("Facts() after committed unsupported revision = %+v, want closed incompatible empty", facts)
	}
	select {
	case <-started:
		t.Fatal("unsupported revision started a handler")
	default:
	}
}
