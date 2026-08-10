package bootstrap

import (
	"context"
	"time"
)

func withStageBudget(parent context.Context, stageBudget time.Duration) (context.Context, context.CancelFunc) {
	if stageBudget <= 0 {
		return context.WithCancel(parent) // #nosec G118 -- cancel function is returned to caller.
	}
	return context.WithTimeout(parent, stageBudget) // #nosec G118 -- cancel function is returned to caller.
}
