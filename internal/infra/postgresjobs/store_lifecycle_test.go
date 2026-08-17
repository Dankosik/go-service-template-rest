package postgresjobs

import (
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestUnboundStoreAndSessionFailBeforeIssuingDatabaseWork(t *testing.T) {
	ctx := t.Context()
	var store Store
	if _, err := store.Stage(ctx, nil, jobs.Prepared{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("Stage() error = %v, want ErrConfig", err)
	}
	if _, err := store.ResolveAcceptance(ctx, jobs.ReadbackExpectation{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("ResolveAcceptance() error = %v, want ErrConfig", err)
	}
	if err := store.CheckProducerPath(ctx); !errors.Is(err, ErrConfig) {
		t.Fatalf("CheckProducerPath() error = %v, want ErrConfig", err)
	}

	revision := jobs.Revision{Kind: "email", ArgsVersion: "v1", PolicyVersion: "p1"}
	attempt := testAttempt()
	transition := jobs.Transition{
		State: jobs.StateSucceeded, AttemptsUsed: 1, Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectNone,
	}
	var session Session
	for _, test := range []struct {
		name string
		call func() error
	}{
		{name: "claim", call: func() error {
			_, err := session.Claim(ctx, ClaimOptions{RegistryKeys: []jobs.Revision{revision}, WorkerID: attempt.WorkerID, Limit: 1, LeaseDuration: time.Second})
			return err
		}},
		{name: "resolve claims", call: func() error {
			_, err := session.ResolveClaims(ctx, []AttemptIdentity{attempt})
			return err
		}},
		{name: "finalize", call: func() error {
			_, err := session.Finalize(ctx, FinalizeInput{Attempt: attempt, Transition: transition})
			return err
		}},
		{name: "renew", call: func() error {
			_, err := session.Renew(ctx, []AttemptIdentity{attempt}, time.Second)
			return err
		}},
		{name: "observe", call: func() error {
			_, err := session.Observe(ctx, []jobs.Revision{revision})
			return err
		}},
		{name: "rescue candidates", call: func() error {
			_, err := session.RescueCandidates(ctx, RescueCandidateOptions{
				Limits: []RescueLimit{{Revision: revision, MaxRecoveryWave: 1}}, Limit: 1,
			})
			return err
		}},
		{name: "rescue", call: func() error {
			_, err := session.Rescue(ctx, RescueInput{Attempt: attempt, Transition: transition})
			return err
		}},
		{name: "schema", call: func() error { return session.CheckSchema(ctx) }},
		{name: "producer authority", call: func() error { return session.checkProducerAuthority(ctx) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, ErrConfig) {
				t.Fatalf("unbound Session error = %v, want ErrConfig", err)
			}
		})
	}
	if got := session.BackendPID(); got != 0 {
		t.Fatalf("BackendPID() = %d, want 0 without a session", got)
	}
	session.Release(ctx)
}
