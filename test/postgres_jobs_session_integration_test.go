//go:build integration

package integration_test

import (
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
)

func TestPostgresJobsSession(t *testing.T) {
	ctx, pool, store := newPostgresJobsFixture(t)

	healthy, err := store.AcquireSession(ctx)
	if err != nil {
		t.Fatalf("AcquireSession(healthy): %v", err)
	}
	healthyPID := healthy.BackendPID()
	if healthyPID == 0 {
		t.Fatal("healthy Session has zero backend PID")
	}
	if err := healthy.CheckSchema(ctx); err != nil {
		t.Fatalf("healthy CheckSchema(): %v", err)
	}
	healthy.Release(ctx)

	broken, err := store.AcquireSession(ctx)
	if err != nil {
		t.Fatalf("AcquireSession(broken): %v", err)
	}
	brokenPID := broken.BackendPID()
	if brokenPID == 0 {
		t.Fatal("broken Session has zero backend PID before fault")
	}
	var terminated bool
	if err := pool.PGX().QueryRow(ctx, "SELECT pg_terminate_backend($1)", brokenPID).Scan(&terminated); err != nil || !terminated {
		t.Fatalf("terminate backend %d = %t, %v", brokenPID, terminated, err)
	}
	if err := broken.CheckSchema(ctx); !errors.Is(err, postgresjobs.ErrSessionTerminal) {
		t.Fatalf("CheckSchema(broken) error = %v, want ErrSessionTerminal", err)
	}
	if got := broken.BackendPID(); got != brokenPID && got != 0 {
		t.Fatalf("broken Session reacquired backend %d, started with %d", got, brokenPID)
	}
	broken.Release(ctx)

	replacement, err := store.AcquireSession(ctx)
	if err != nil {
		t.Fatalf("AcquireSession(replacement): %v", err)
	}
	defer replacement.Release(ctx)
	if replacementPID := replacement.BackendPID(); replacementPID == 0 || replacementPID == brokenPID {
		t.Fatalf("replacement backend PID = %d, want nonzero and different from %d", replacementPID, brokenPID)
	}
	if err := replacement.CheckSchema(ctx); err != nil {
		t.Fatalf("replacement CheckSchema(): %v", err)
	}
}
