package postgresjobs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
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
				reader := telemetrytest.InstallManualReader(t)
				registry := engineRegistryWithAttemptDuration(t, test.duration, test.handler)
				claim := engineClaim()
				finalized := make(chan FinalizeInput, 1)
				store := &engineStoreStub{
					claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
						return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
					},
					finalize: func(_ context.Context, input FinalizeInput) (PersistedTransition, error) {
						finalized <- input
						return PersistedTransition{Status: TransitionApplied}, nil
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
				assertJobsEvent(t, reader, "attempt", test.want)
			})
		})
	}
}

func TestEngineAttemptClassifiesPoisonPayload(t *testing.T) {
	claim := engineClaim()
	claim.Payload = []byte(`{"unknown":true}`)
	registered, err := engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
		t.Fatal("poison payload reached handler")
		return jobs.HandlerResult{}
	}).Lookup(claim.Revision)
	if err != nil {
		t.Fatal(err)
	}
	_, outcome, effect := dispatchAttempt(context.Background(), claim, registered)
	if outcome != jobs.OutcomePoison || effect != jobs.EffectNone {
		t.Fatalf("dispatchAttempt(poison) = %q/%q, want poison/none", outcome, effect)
	}
}

func TestEngineAttemptDoesNotRecordUnappliedFinalization(t *testing.T) {
	for _, test := range []struct {
		name   string
		result PersistedTransition
		err    error
	}{
		{name: string(TransitionStale), result: PersistedTransition{Status: TransitionStale}},
		{name: string(TransitionNotFound), result: PersistedTransition{Status: TransitionNotFound}},
		{name: "error", err: errors.New("finalize failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				reader := telemetrytest.InstallManualReader(t)
				claim := engineClaim()
				store := &engineStoreStub{
					claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
						return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
					},
					finalize: func(context.Context, FinalizeInput) (PersistedTransition, error) {
						return test.result, test.err
					},
				}
				engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
					return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
				}), engineConfig())
				if err != nil {
					t.Fatal(err)
				}
				if err := engine.Run(context.Background()); err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				assertNoJobsEvent(t, reader, "attempt", jobs.OutcomeSuccess)
			})
		})
	}
}

func TestEngineAttemptBudgetIncludesClaimHandoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claim := engineClaim()
		claim.BudgetElapsed = time.Hour - 1500*time.Millisecond
		finalized := make(chan FinalizeInput, 1)
		store := &engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
			},
			finalize: func(_ context.Context, input FinalizeInput) (PersistedTransition, error) {
				finalized <- input
				return PersistedTransition{Status: TransitionApplied}, nil
			},
		}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			time.Sleep(2 * time.Second)
			return jobs.HandlerResult{Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		if got := (<-finalized).Transition.State; got != jobs.StateExhausted {
			t.Fatalf("finalized state = %q, want %q", got, jobs.StateExhausted)
		}
	})
}

func TestEngineAttemptFinalizationFailureIsTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		want := fmt.Errorf("%w: finalize failed", ErrOperationTimeout)
		claim := engineClaim()
		claims := 0
		store := &engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				claims++
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
			},
			finalize: func(context.Context, FinalizeInput) (PersistedTransition, error) {
				return PersistedTransition{}, want
			},
		}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := <-engine.Terminal(); !errors.Is(err, want) {
			t.Fatalf("terminal error = %v, want %v", err, want)
		}
		if err := engine.Run(context.Background()); !errors.Is(err, want) {
			t.Fatalf("Run() after terminal failure = %v, want %v", err, want)
		}
		if claims != 1 {
			t.Fatalf("claim calls after terminal failure = %d, want 1", claims)
		}
		if facts := engine.Facts(); facts.ClaimAdmissionOpen || facts.ObservationFresh {
			t.Fatalf("Facts() = %+v, want terminal admission closure", facts)
		}
	})
}

func TestEngineAttemptUnknownFinalizationUsesFencedReadback(t *testing.T) {
	for _, test := range []struct {
		name         string
		second       PersistedTransition
		firstErr     error
		wantCalls    int
		wantEvent    bool
		wantTerminal bool
	}{
		{name: "committed", firstErr: fmt.Errorf("%w: response lost", postgres.ErrCommitUnknown), second: PersistedTransition{Status: TransitionRepeated, Transition: jobs.Transition{Outcome: jobs.OutcomeSuccess}}, wantCalls: 2, wantEvent: true},
		{name: "not committed", firstErr: fmt.Errorf("%w: response lost", postgres.ErrCommitUnknown), second: PersistedTransition{Status: TransitionApplied}, wantCalls: 2, wantEvent: true},
		{name: "stale", firstErr: fmt.Errorf("%w: response lost", postgres.ErrCommitUnknown), second: PersistedTransition{Status: TransitionStale}, wantCalls: 2},
		{name: "session lost", firstErr: errors.Join(postgres.ErrCommitUnknown, ErrSessionTerminal), wantCalls: 1, wantTerminal: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := telemetrytest.InstallManualReader(t)
			calls := 0
			store := &engineStoreStub{finalize: func(context.Context, FinalizeInput) (PersistedTransition, error) {
				calls++
				if calls == 1 {
					return PersistedTransition{}, test.firstErr
				}
				return test.second, nil
			}}
			engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
			if err != nil {
				t.Fatal(err)
			}
			input := FinalizeInput{Attempt: engineClaim().Attempt, Transition: jobs.Transition{
				State: jobs.StateSucceeded, AttemptsUsed: 1, ElapsedUsed: time.Second,
				Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted,
			}}
			engine.finalizeAttempt(context.Background(), input, time.Second)
			if calls != test.wantCalls {
				t.Fatalf("Finalize calls = %d, want %d", calls, test.wantCalls)
			}
			if test.wantEvent {
				assertJobsEvent(t, reader, "attempt", jobs.OutcomeSuccess)
			} else {
				assertNoJobsEvent(t, reader, "attempt", jobs.OutcomeSuccess)
			}
			select {
			case terminalErr := <-engine.Terminal():
				if !test.wantTerminal || !errors.Is(terminalErr, ErrSessionTerminal) {
					t.Fatalf("terminal error = %v, wantTerminal=%t", terminalErr, test.wantTerminal)
				}
			default:
				if test.wantTerminal {
					t.Fatal("terminal error was not reported")
				}
			}
		})
	}
}

func TestEngineAttemptRecordsAppliedRetry(t *testing.T) {
	reader := telemetrytest.InstallManualReader(t)
	store := &engineStoreStub{finalize: func(context.Context, FinalizeInput) (PersistedTransition, error) {
		return PersistedTransition{Status: TransitionApplied}, nil
	}}
	engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult { return jobs.HandlerResult{} }), engineConfig())
	if err != nil {
		t.Fatal(err)
	}
	input := FinalizeInput{Attempt: engineClaim().Attempt, Transition: jobs.Transition{
		State: jobs.StateRetryWait, Delay: time.Second, AttemptsUsed: 1,
		ElapsedUsed: time.Second, Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone,
	}}
	engine.finalizeAttempt(context.Background(), input, time.Second)
	assertJobsEvent(t, reader, "attempt", jobs.OutcomeRetryable)
	assertJobsEvent(t, reader, "retry", jobs.OutcomeRetryable)
}

func TestEngineAttemptEvaluationFailureIsTerminal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claim := engineClaim()
		claim.BudgetElapsed = -time.Second
		store := &engineStoreStub{claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
			return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
		}}
		engine, err := newEngine(store, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := <-engine.Terminal(); !errors.Is(err, jobs.ErrInvalidTransition) {
			t.Fatalf("terminal error = %v, want ErrInvalidTransition", err)
		}
	})
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
		select {
		case <-finalized:
			t.Fatal("attempt finalized while coordinator observation was running")
		default:
		}
		engine.mu.Lock()
		state := engine.inflight[claim.Attempt]
		state.renewAt = time.Now()
		engine.inflight[claim.Attempt] = state
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
