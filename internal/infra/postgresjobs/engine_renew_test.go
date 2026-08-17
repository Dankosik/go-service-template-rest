package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineRenewPrecedesRescueAndSignalsMatchingCancellation(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		reader := telemetrytest.InstallManualReader(t)
		order := make([]string, 0, 2)
		store := &engineStoreStub{
			renew: func(_ context.Context, attempts []AttemptIdentity, _ time.Duration) ([]Renewal, error) {
				order = append(order, "renew")
				return []Renewal{{Attempt: attempts[0], ObservedAt: time.Now(), CancelRequested: true}}, nil
			},
			candidates: func(context.Context, RescueCandidateOptions) ([]RescueCandidate, error) {
				order = append(order, "rescue")
				return nil, nil
			},
		}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		attempt := engineClaim().Attempt
		attemptCtx, cancel := context.WithCancel(context.Background())
		engine.mu.Lock()
		engine.inflight[attempt] = inflightAttempt{cancel: cancel, renewAt: time.Now()}
		engine.mu.Unlock()
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(order) < 2 || order[0] != "renew" || order[1] != "rescue" {
			t.Fatalf("stage order = %v, want renew before rescue", order)
		}
		if attemptCtx.Err() == nil {
			t.Fatal("matching cancellation did not signal attempt")
		}
		if got := jobsEventCount(t, reader, "cancellation", jobs.OutcomeCancelled); got != 1 {
			t.Fatalf("cancellation events = %d, want 1", got)
		}
		engine.mu.Lock()
		inflight := engine.inflight[attempt]
		inflight.renewAt = time.Now()
		engine.inflight[attempt] = inflight
		engine.mu.Unlock()
		if err := engine.renew(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got := jobsEventCount(t, reader, "cancellation", jobs.OutcomeCancelled); got != 1 {
			t.Fatalf("repeated cancellation events = %d, want 1", got)
		}
	})
}

func TestEngineRenewFaultClosesAdmissionAndSignalsAttempts(t *testing.T) {
	t.Parallel()
	engine, err := newEngine(&engineStoreStub{renew: func(context.Context, []AttemptIdentity, time.Duration) ([]Renewal, error) {
		return nil, errors.New("lost control session")
	}}, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	attempt := engineClaim().Attempt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.mu.Lock()
	engine.inflight[attempt] = inflightAttempt{cancel: cancel, renewAt: time.Now()}
	engine.mu.Unlock()
	if err := engine.Run(context.Background()); err == nil {
		t.Fatal("Run() error = nil, want renewal failure")
	}
	if facts := engine.Facts(); facts.ClaimAdmissionOpen || facts.ObservationFresh {
		t.Fatalf("Facts() = %+v, want admission closed with stale observation", facts)
	}
	if ctx.Err() == nil {
		t.Fatal("renewal failure did not cancel active attempt")
	}
}

func TestEngineRenewUsesPerAttemptMonotonicDeadline(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		oldAttempt := engineClaim().Attempt
		newAttempt := oldAttempt
		newAttempt.LogicalJobID = "job-2"
		newAttempt.AttemptGeneration = 2
		var renewed []AttemptIdentity
		store := &engineStoreStub{renew: func(_ context.Context, attempts []AttemptIdentity, _ time.Duration) ([]Renewal, error) {
			renewed = append(renewed, attempts...)
			return []Renewal{{Attempt: oldAttempt, ObservedAt: time.Now()}}, nil
		}}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		_, oldCancel := context.WithCancel(context.Background())
		defer oldCancel()
		_, newCancel := context.WithCancel(context.Background())
		defer newCancel()
		engine.mu.Lock()
		engine.inflight[oldAttempt] = inflightAttempt{cancel: oldCancel, renewAt: time.Now()}
		engine.inflight[newAttempt] = inflightAttempt{cancel: newCancel, renewAt: time.Now().Add(engine.config.LeaseDuration / 3)}
		engine.mu.Unlock()

		if err := engine.renew(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(renewed) != 1 || renewed[0] != oldAttempt {
			t.Fatalf("renewed attempts = %+v, want only oldest due attempt", renewed)
		}
	})
}

func TestEngineClaimLatencyCannotPostponeFirstRenewal(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		claim := engineClaim()
		release := make(chan struct{})
		renewed := make(chan []AttemptIdentity, 1)
		store := &engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				time.Sleep(30 * time.Second)
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
			},
			renew: func(_ context.Context, attempts []AttemptIdentity, _ time.Duration) ([]Renewal, error) {
				renewed <- attempts
				return []Renewal{{Attempt: claim.Attempt, ObservedAt: time.Now()}}, nil
			},
		}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			<-release
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		attempts := <-renewed
		if len(attempts) != 1 || attempts[0] != claim.Attempt {
			t.Fatalf("renewed attempts = %+v, want claimed attempt", attempts)
		}
		close(release)
	})
}
