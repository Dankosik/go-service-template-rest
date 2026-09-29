package runtimeopts

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
)

// UnarmedTeardown leaves startup-failure cleanup its full stage budget. The
// signal's cancellation and any startup deadline do not enter that cleanup.
func UnarmedTeardown(base context.Context) context.Context {
	return context.WithoutCancel(base)
}

// ArmTeardown starts the one process-wide grace period when serving ends.
// Cancel releases the process timer after ordered teardown. Stage retains the
// deadline for deferred cleanup even if that process context is canceled first.
func ArmTeardown(base context.Context, grace time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(base), grace)
}

// TeardownStage gives one teardown operation its ceiling within the process
// window. The window context carries its process deadline to every stage.
// It detaches cancellation so a deferred cleanup can still run after the
// process context has been canceled, while retaining the original deadline.
func TeardownStage(window context.Context, want time.Duration) (context.Context, context.CancelFunc) {
	base := context.WithoutCancel(window)
	if deadline, armed := window.Deadline(); armed {
		stageDeadline := time.Now().Add(want)
		if deadline.Before(stageDeadline) {
			stageDeadline = deadline
		}
		return context.WithDeadline(base, stageDeadline)
	}
	return context.WithTimeout(base, want)
}

// TeardownBudget reports the time a stage may still spend within the process window.
// An unarmed startup-cleanup window grants the whole requested stage budget.
func TeardownBudget(window context.Context, want time.Duration) time.Duration {
	deadline, armed := window.Deadline()
	if !armed {
		return want
	}
	if remaining := time.Until(deadline); remaining < want {
		return max(remaining, 0)
	}
	return want
}

// ValidateGracePeriod rejects a drain budget that cannot fit inside the
// platform's grace period alongside the teardown that follows it.
//
// The stage ceilings summed into cleanupReserve are process structure owned by
// each composition root rather than configuration, which is why the reserve is a
// parameter and only the grace period is read from config. drainName is the
// configuration key that sets the drain when it has one, otherwise a description
// of the code-owned drain budget; it differs per binary because each drains
// something different.
//
// The bound is checked as two comparisons rather than one sum: a configured
// drain near the int64 ceiling makes drain+cleanupReserve wrap negative, and a
// wrapped requirement is one every grace period satisfies — which would admit
// the process with a drain no shutdown can ever complete.
func ValidateGracePeriod(gracePeriod time.Duration, drainName string, drain, cleanupReserve time.Duration) error {
	if gracePeriod >= drain && gracePeriod-drain >= cleanupReserve {
		return nil
	}
	return fmt.Errorf(
		"%w: http.grace_period must be >= %s plus the post-drain teardown budget (%s + %s = %s)",
		config.ErrValidate,
		drainName,
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
