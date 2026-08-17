package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/jobs"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestEngineClaimRegistersKnownCommitBeforeHandlerStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader := telemetrytest.InstallManualReader(t)
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
		if facts := engine.Facts(); facts.InFlight != 0 || !facts.ClaimAdmissionOpen {
			t.Fatalf("Facts() during claim = %+v, want responsive open admission without registered work", facts)
		}
		close(releaseClaim)
		if err := <-run; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		<-started
		if got := engine.Facts().InFlight; got != 1 {
			t.Fatalf("in-flight handlers = %d, want registered handler", got)
		}
		assertJobsEvent(t, reader, "claim", jobs.OutcomeSuccess)
		close(allowFinish)
		synctest.Wait()
	})
}

func assertJobsEvent(t *testing.T, reader *sdkmetric.ManualReader, event string, outcome jobs.OutcomeClass) {
	t.Helper()
	if value := jobsEventCount(t, reader, event, outcome); value != 1 {
		t.Fatalf("%s/%s events = %d, want 1", event, outcome, value)
	}
}

func assertNoJobsEvent(t *testing.T, reader *sdkmetric.ManualReader, event string, outcome jobs.OutcomeClass) {
	t.Helper()
	if value := jobsEventCount(t, reader, event, outcome); value != 0 {
		t.Fatalf("%s/%s events = %d, want 0", event, outcome, value)
	}
}

func jobsEventCount(t *testing.T, reader *sdkmetric.ManualReader, event string, outcome jobs.OutcomeClass) int64 {
	t.Helper()
	var value int64
	telemetrytest.ForEachMetric(t, reader, func(measured metricdata.Metrics) {
		if measured.Name != "postgres.jobs.events" {
			return
		}
		for _, point := range telemetrytest.Int64Sum(t, measured).DataPoints {
			if telemetrytest.Attribute(t, point.Attributes, "event") == event &&
				telemetrytest.Attribute(t, point.Attributes, "outcome") == string(outcome) {
				value += point.Value
			}
		}
	})
	return value
}

func TestEngineClaimUnknownCommitStartsOnlyWriterResolvedAttempt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader := telemetrytest.InstallManualReader(t)
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
		assertJobsEvent(t, reader, "claim", jobs.OutcomeSuccess)
	})
}

func TestEngineClaimUnknownCommitWithNoResolvedAttemptDoesNotRecordEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader := telemetrytest.InstallManualReader(t)
		claim := engineClaim()
		started := make(chan struct{}, 1)
		store := &engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, postgres.ErrCommitUnknown
			},
			resolve: func(context.Context, []AttemptIdentity) ([]ClaimResolution, error) {
				return nil, nil
			},
		}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			started <- struct{}{}
			return jobs.HandlerResult{}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		select {
		case <-started:
			t.Fatal("unresolved claim started a handler")
		default:
		}
		assertNoJobsEvent(t, reader, "claim", jobs.OutcomeSuccess)
	})
}

func TestEngineClaimUnknownCommitWithLostSessionDoesNotReadBackOrDispatch(t *testing.T) {
	claim := engineClaim()
	resolved := false
	started := false
	store := &engineStoreStub{
		claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
			return ClaimResult{Attempts: []ClaimedAttempt{claim}}, errors.Join(postgres.ErrCommitUnknown, ErrSessionTerminal)
		},
		resolve: func(context.Context, []AttemptIdentity) ([]ClaimResolution, error) {
			resolved = true
			return nil, nil
		},
	}
	engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
		started = true
		return jobs.HandlerResult{}
	}), engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(context.Background()); !errors.Is(err, ErrSessionTerminal) {
		t.Fatalf("Run() error = %v, want ErrSessionTerminal", err)
	}
	if resolved || started {
		t.Fatalf("lost Session resolved/dispatched = %t/%t, want neither", resolved, started)
	}
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
