package postgresjobs

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineAttemptEvaluatesAndFinalizesCapturedFacts(t *testing.T) {
	for _, test := range []struct {
		name     string
		handler  func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult
		want     jobs.OutcomeClass
		state    jobs.State
		duration time.Duration
		cancel   bool
	}{
		{name: "panic", handler: func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { panic("boom") }, want: jobs.OutcomePanic, state: jobs.StateOutcomeUnknown, duration: time.Minute},
		{name: "timeout", handler: func(ctx context.Context, _ jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			<-ctx.Done()
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}, want: jobs.OutcomeTimeout, state: jobs.StateOutcomeUnknown, duration: time.Second},
		{name: "cancellation", handler: func(ctx context.Context, _ jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			<-ctx.Done()
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}, want: jobs.OutcomeCancelled, state: jobs.StateOutcomeUnknown, duration: time.Minute, cancel: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				registry := engineRegistryWithAttemptDuration(t, test.duration, test.handler)
				claim := engineClaim()
				finalized := make(chan FinalizeInput, 1)
				store := &engineStoreStub{
					claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
						return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
					},
					finalize: func(_ context.Context, input FinalizeInput) (PersistedTransition, error) {
						finalized <- input
						return PersistedTransition{}, nil
					},
				}
				engine, err := newEngine(store, registry, engineConfig())
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if err := engine.Run(ctx); err != nil {
					t.Fatal(err)
				}
				if test.name == "timeout" {
					time.Sleep(test.duration)
				} else if test.cancel {
					cancel()
				}
				got := <-finalized
				if got.Transition.Outcome != test.want {
					t.Fatalf("finalized outcome = %q, want %q", got.Transition.Outcome, test.want)
				}
				if got.Transition.State != test.state || got.Transition.Effect != jobs.EffectUnknown {
					t.Fatalf("finalized transition = %+v, want state %q with unknown effect", got.Transition, test.state)
				}
				synctest.Wait()
				select {
				case duplicate := <-finalized:
					t.Fatalf("second finalization = %+v", duplicate)
				default:
				}
			})
		})
	}
}

func TestEngineAttemptFinalizationWaitsForCoordinator(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		observing := make(chan struct{})
		releaseObservation := make(chan struct{})
		finalized := make(chan struct{}, 1)
		order := make([]string, 0, 2)
		claim := engineClaim()
		store := &engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
			},
			observe: func(context.Context, []jobs.Revision) (Observation, error) {
				close(observing)
				<-releaseObservation
				return Observation{ObservedAt: time.Now(), Compatible: true}, nil
			},
			renew: func(context.Context, []AttemptIdentity, time.Duration) ([]Renewal, error) {
				order = append(order, "renew")
				return []Renewal{{Attempt: claim.Attempt, ObservedAt: time.Now()}}, nil
			},
			finalize: func(context.Context, FinalizeInput) (PersistedTransition, error) {
				order = append(order, "finalize")
				finalized <- struct{}{}
				return PersistedTransition{}, nil
			},
		}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		run := make(chan error, 1)
		go func() { run <- engine.Run(context.Background()) }()
		<-observing
		synctest.Wait()
		select {
		case <-finalized:
			t.Fatal("attempt finalized while coordinator observation was running")
		default:
		}
		engine.mu.Lock()
		engine.lastLease = time.Now().Add(-engine.config.LeaseDuration / 3)
		engine.mu.Unlock()
		close(releaseObservation)
		if err := <-run; err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		<-finalized
		if len(order) != 2 || order[0] != "renew" || order[1] != "finalize" {
			t.Fatalf("completion order = %v, want renewal before finalization", order)
		}
		synctest.Wait()
	})
}
