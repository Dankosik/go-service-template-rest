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
