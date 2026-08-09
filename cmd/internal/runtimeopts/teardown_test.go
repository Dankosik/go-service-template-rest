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
// stage needs, and a stage past the deadline still gets enough to report that it
// was cut short.
//
// The second one is why this is not simply min(want, remaining). A stage handed
// an expired context exports nothing and cleans up nothing, so the last stage of
// a shutdown that ran long — normally the telemetry flush — would lose exactly
// the record of why it ran long.
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
	if got := runtimeopts.TeardownBudget(time.Hour, time.Now().Add(-time.Second)); got != runtimeopts.TeardownFloor {
		t.Fatalf("TeardownBudget(1h, spent) = %s, want the floor %s", got, runtimeopts.TeardownFloor)
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
	deadline, ok := spent.Deadline()
	if !ok {
		t.Fatal("spent-grace teardown stage has no deadline")
	}
	if remaining := time.Until(deadline); remaining > runtimeopts.TeardownFloor {
		t.Fatalf(
			"teardown stage past the grace period = %s, want at most the floor %s",
			remaining, runtimeopts.TeardownFloor,
		)
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
