package bootstrap

import (
	"context"
	"fmt"
	"time"
)

func withStageBudget(parent context.Context, stageBudget time.Duration) (context.Context, context.CancelFunc) {
	if stageBudget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	return context.WithTimeout(parent, stageBudget) // #nosec G118 -- cancel function is returned to caller.
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("sleep canceled: %w", ctx.Err())
	case <-time.After(wait):
		return nil
	}
}
