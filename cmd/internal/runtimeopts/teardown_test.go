package runtimeopts_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
)

// TestTeardownBudgetDrawsFromTheProcessDeadline holds the two properties every
// composition root's teardown depends on: a stage never spends budget a later
// stage needs, and a spent process deadline cannot be extended by another stage.
func TestTeardownBudgetDrawsFromTheProcessDeadline(t *testing.T) {
	t.Parallel()

	if got := runtimeopts.TeardownBudget(time.Second, time.Time{}); got != time.Second {
		t.Fatalf("TeardownBudget(1s, unarmed) = %s, want the full stage budget", got)
	}
	if got := runtimeopts.TeardownBudget(time.Second, time.Now().Add(time.Hour)); got != time.Second {
		t.Fatalf("TeardownBudget(1s, an hour left) = %s, want the full stage budget", got)
	}
	if got := runtimeopts.TeardownBudget(time.Hour, time.Now().Add(2*time.Second)); got > 2*time.Second {
		t.Fatalf("TeardownBudget(1h, 2s left) = %s, want at most what is left", got)
	}
	if got := runtimeopts.TeardownBudget(time.Hour, time.Now().Add(-time.Second)); got != 0 {
		t.Fatalf("TeardownBudget(1h, spent) = %s, want zero", got)
	}
}

// TestTeardownStageOutlivesACanceledSignalContext covers the detach. Every
// caller passes the signal context, which is already canceled by the time any
// stage runs, so a stage that inherited cancellation would get an instant expiry
// instead of its bound.
func TestTeardownStageOutlivesACanceledSignalContext(t *testing.T) {
	t.Parallel()

	signalCtx, stop := context.WithCancel(context.Background())
	stop()

	ctx, cancel := runtimeopts.TeardownStage(signalCtx, time.Time{}, time.Hour)
	defer cancel()
	if err := ctx.Err(); err != nil {
		t.Fatalf("teardown stage inherited signal cancellation: %v", err)
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("teardown stage has no deadline")
	}

	spent, spentCancel := runtimeopts.TeardownStage(signalCtx, time.Now().Add(-time.Second), time.Hour)
	defer spentCancel()
	if !errors.Is(spent.Err(), context.DeadlineExceeded) {
		t.Fatalf("spent-grace teardown stage error = %v, want context deadline", spent.Err())
	}
}

func TestValidateGracePeriodChargesForTheTeardownThatFollowsTheDrain(t *testing.T) {
	t.Parallel()

	if err := runtimeopts.ValidateGracePeriod(30*time.Second, "http.shutdown_timeout", 15*time.Second, 15*time.Second); err != nil {
		t.Fatalf("ValidateGracePeriod(exactly enough) error = %v", err)
	}
	// The named leaf is what the operator edits, so a rejection that omitted it
	// would report an unfixable budget.
	short := rejectedGracePeriod(t, 29*time.Second, "http.shutdown_timeout", 15*time.Second, 15*time.Second)
	if !strings.Contains(short, "http.shutdown_timeout") {
		t.Fatalf("ValidateGracePeriod rejection = %q, want it to name the drain setting", short)
	}

	// A drain at the int64 ceiling is what makes the requirement wrap. Summing
	// first would make every grace period satisfy a negative bound and admit a
	// process whose drain no shutdown can complete.
	overflow := rejectedGracePeriod(
		t, time.Minute, "http.shutdown_timeout", time.Duration(math.MaxInt64), 15*time.Second,
	)
	if strings.Contains(overflow, "= -") {
		t.Fatalf("ValidateGracePeriod overflow rejection = %q, want a saturated rather than wrapped budget", overflow)
	}
}

// rejectedGracePeriod is the rendered rejection for a budget that must not be
// admitted, and fails the test when one was admitted instead.
func rejectedGracePeriod(
	t *testing.T,
	gracePeriod time.Duration,
	drainLeaf string,
	drain, cleanupReserve time.Duration,
) string {
	t.Helper()

	err := runtimeopts.ValidateGracePeriod(gracePeriod, drainLeaf, drain, cleanupReserve)
	if err == nil {
		t.Fatalf("ValidateGracePeriod(%s grace, %s drain) error = nil, want ErrValidate", gracePeriod, drain)
	}
	if !errors.Is(err, config.ErrValidate) {
		t.Fatalf("ValidateGracePeriod(%s grace, %s drain) error = %v, want ErrValidate", gracePeriod, drain, err)
	}
	return err.Error()
}
