package config

import (
	"context"
	"fmt"
	"time"
)

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: nil context", ErrLoad)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrLoad, err)
	}
	return nil
}

func withContextBudget(parent context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	if budget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < budget {
			budget = remaining
		}
	}
	if budget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	return context.WithTimeout(parent, budget) // #nosec G118 -- cancel function is returned to caller.
}
