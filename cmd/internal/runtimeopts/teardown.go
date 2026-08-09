package runtimeopts

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

// TeardownFloor is the smallest budget a teardown stage is given once the
// process grace period is spent.
//
// A spent grace period yields this rather than zero: the process is about to be
// killed either way, and a stage given no time at all cannot even report that it
// was cut short — a telemetry flush handed an expired context exports nothing,
// and an adapter cleanup handed one is indistinguishable from a cleanup that
// hung. It is long enough for an already-idle stage to finish and record that it
// did, and short enough not to extend a shutdown that is over.
const TeardownFloor = 100 * time.Millisecond

// TeardownBudget reports how long a stage asking for want may actually take.
//
// deadline is the one process-wide teardown deadline every stage draws from.
// Without it the stages are independent ceilings running in sequence, so the
// worst case is their sum — and the loser is always the last stage, which is
// normally the telemetry flush that runs last precisely so it can record what
// the others did. A zero deadline means no process bound has been armed yet,
// which is the startup-failure path: the stage gets what it asked for.
func TeardownBudget(want time.Duration, deadline time.Time) time.Duration {
	if deadline.IsZero() {
		return want
	}
	if remaining := time.Until(deadline); remaining < want {
		return max(remaining, TeardownFloor)
	}
	return want
}

// ArmTeardown opens the one process-wide teardown window and reports the
// deadline every stage then draws from through [TeardownBudget].
//
// [TeardownBudget] already documents that deadline as shared, but arming it was
// still spelled per binary. Both spellings were this arithmetic, so the two ways
// to get it wrong — deriving the window from a context already canceled, and
// reading a deadline the window never carried — are answered once here.
func ArmTeardown(base context.Context, grace time.Duration) (context.Context, context.CancelFunc, time.Time) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(base), grace)
	// Present by construction: WithTimeout always carries one.
	deadline, _ := ctx.Deadline()
	return ctx, cancel, deadline
}

// TeardownStage builds the context one teardown stage runs under, bounded by
// [TeardownBudget].
//
// It is detached from base, which is normally the signal context and therefore
// already canceled by the time any stage runs, so the stage gets the bound
// rather than an instant expiry.
func TeardownStage(
	base context.Context,
	deadline time.Time,
	want time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(base), TeardownBudget(want, deadline))
}

// ValidateGracePeriod rejects a drain budget that cannot fit inside the
// platform's grace period alongside the teardown that follows it.
//
// The stage ceilings summed into cleanupReserve are process structure owned by
// each composition root rather than configuration, which is why the reserve is a
// parameter and only the grace period is read from config. drainLeaf names the
// configuration the operator actually edits to make the sum fit, and differs per
// binary because each drains something different.
//
// The bound is checked as two comparisons rather than one sum: a configured
// drain near the int64 ceiling makes drain+cleanupReserve wrap negative, and a
// wrapped requirement is one every grace period satisfies — which would admit
// the process with a drain no shutdown can ever complete.
func ValidateGracePeriod(gracePeriod time.Duration, drainLeaf string, drain, cleanupReserve time.Duration) error {
	if gracePeriod >= drain && gracePeriod-drain >= cleanupReserve {
		return nil
	}
	return fmt.Errorf(
		"%w: http.grace_period must be >= %s plus the post-drain teardown budget (%s + %s = %s)",
		config.ErrValidate,
		drainLeaf,
		drain,
		cleanupReserve,
		requiredGracePeriod(drain, cleanupReserve),
	)
}

// requiredGracePeriod is the sum for the rejection message alone, saturated so
// the operator is told an unreachable budget rather than a negative one.
func requiredGracePeriod(drain, cleanupReserve time.Duration) time.Duration {
	if required := drain + cleanupReserve; required >= drain {
		return required
	}
	return math.MaxInt64
}
