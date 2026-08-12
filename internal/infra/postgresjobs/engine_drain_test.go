package postgresjobs

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestEngineDrainClosesAdmissionBeforeAnotherClaim(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		claims := 0
		engine, err := newEngine(&engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				claims++
				return ClaimResult{}, nil
			},
		}, engineRegistry(t, func(context.Context, jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if result := engine.StartDrain(context.Background()); !result.CleanupSafe || result.Err != nil {
			t.Fatalf("StartDrain() = %+v, want safe successful drain", result)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatalf("Run() after drain error = %v", err)
		}
		if claims != 0 {
			t.Fatalf("claims after drain = %d, want 0", claims)
		}
	})
}

func TestEngineDrainCancelsAndJoinsRegisteredAttempt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		stopped := make(chan struct{})
		claim := engineClaim()
		engine, err := newEngine(&engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
			},
		}, engineRegistry(t, func(ctx context.Context, _ jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			close(started)
			<-ctx.Done()
			close(stopped)
			return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectUnknown}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		<-started
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		got := engine.StartDrain(ctx)
		if !got.CleanupSafe || got.Err != nil {
			t.Fatalf("StartDrain() = %+v, want joined cancellation result", got)
		}
		<-stopped
	})
}

func TestEngineDrainReportsUnsafeUntilAttemptJoins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		stopped := make(chan struct{})
		claim := engineClaim()
		engine, err := newEngine(&engineStoreStub{
			claim: func(context.Context, ClaimOptions) (ClaimResult, error) {
				return ClaimResult{Attempts: []ClaimedAttempt{claim}}, nil
			},
		}, engineRegistry(t, func(ctx context.Context, _ jobs.HandlerInput[engineArgs]) jobs.HandlerResult {
			close(started)
			<-release
			<-ctx.Done()
			close(stopped)
			return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectUnknown}
		}), engineConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err := engine.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		<-started
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		result := make(chan DrainResult, 1)
		go func() { result <- engine.StartDrain(ctx) }()
		time.Sleep(2 * time.Second)
		got := <-result
		if got.CleanupSafe || got.Err == nil {
			t.Fatalf("StartDrain() = %+v, want unsafe bounded result", got)
		}
		close(release)
		<-stopped
		synctest.Wait()
	})
}
