package postgresjobs

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineRenewPrecedesRescueAndSignalsMatchingCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		order := make([]string, 0, 2)
		store := &engineStoreStub{
			renew: func(_ context.Context, attempts []AttemptIdentity, _ time.Duration) ([]Renewal, error) {
				order = append(order, "renew")
				return []Renewal{{Attempt: attempts[0], ObservedAt: time.Now(), CancelRequested: true}}, nil
			},
			candidates: func(context.Context, int) ([]RescueCandidate, error) {
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
		engine.inflight[attempt] = cancel
		engine.lastLease = time.Now().Add(-engine.config.LeaseDuration / 3)
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
	})
}

func TestEngineRenewFaultClosesAdmissionAndSignalsAttempts(t *testing.T) {
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
	engine.inflight[attempt] = cancel
	engine.lastLease = time.Now().Add(-engine.config.LeaseDuration / 3)
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
