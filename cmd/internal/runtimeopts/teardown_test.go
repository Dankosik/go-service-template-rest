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

	unarmed := runtimeopts.UnarmedTeardown(context.Background())
	if got := runtimeopts.TeardownBudget(unarmed, time.Second); got != time.Second {
		t.Fatalf("Budget(1s, unarmed) = %s, want the full stage budget", got)
	}
	window, cancel := runtimeopts.ArmTeardown(context.Background(), time.Hour)
	defer cancel()
	if got := runtimeopts.TeardownBudget(window, time.Second); got != time.Second {
		t.Fatalf("Budget(1s, an hour left) = %s, want the full stage budget", got)
	}
	short, cancelShort := runtimeopts.ArmTeardown(context.Background(), 2*time.Second)
	defer cancelShort()
	if got := runtimeopts.TeardownBudget(short, time.Hour); got > 2*time.Second {
		t.Fatalf("Budget(1h, 2s left) = %s, want at most what is left", got)
	}
	spent, cancelSpent := runtimeopts.ArmTeardown(context.Background(), -time.Second)
	defer cancelSpent()
	if got := runtimeopts.TeardownBudget(spent, time.Hour); got != 0 {
		t.Fatalf("Budget(1h, spent) = %s, want zero", got)
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

	unarmed := runtimeopts.UnarmedTeardown(signalCtx)
	ctx, cancel := runtimeopts.TeardownStage(unarmed, time.Hour)
	defer cancel()
	if err := ctx.Err(); err != nil {
		t.Fatalf("teardown stage inherited signal cancellation: %v", err)
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("teardown stage has no deadline")
	}
	armed, cancelArmed := runtimeopts.ArmTeardown(signalCtx, time.Hour)
	cancelArmed()
	afterCancel, cancelAfter := runtimeopts.TeardownStage(armed, time.Minute)
	defer cancelAfter()
	if err := afterCancel.Err(); err != nil {
		t.Fatalf("deferred teardown stage inherited process cancellation: %v", err)
	}
	if remaining, ok := afterCancel.Deadline(); !ok || time.Until(remaining) > time.Minute {
		t.Fatalf("deferred teardown deadline = %v, %t, want at most one minute", remaining, ok)
	}

	window, cancelWindow := runtimeopts.ArmTeardown(signalCtx, -time.Second)
	defer cancelWindow()
	spent, spentCancel := runtimeopts.TeardownStage(window, time.Hour)
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
