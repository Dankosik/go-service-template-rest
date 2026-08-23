package runtimeopts

import (
	"context"
	"fmt"
)

// StartRuntime lets startup cancellation stop a blocking start without making
// the long-lived runtime inherit the startup deadline after admission succeeds.
// started reports that start returned nil, even when cancellation won the race
// to detach and the caller must stop and join that admitted runtime.
func StartRuntime(
	startupCtx context.Context,
	runtimeCtx context.Context,
	cancelRuntime context.CancelFunc,
	start func(context.Context) error,
) (started bool, err error) {
	if err := startupCtx.Err(); err != nil {
		cancelRuntime()
		return false, fmt.Errorf("start runtime: %w", err)
	}
	stopStartupCancel := context.AfterFunc(startupCtx, cancelRuntime)
	err = start(runtimeCtx)
	detached := stopStartupCancel()
	if err != nil {
		return false, err
	}
	if !detached {
		return true, fmt.Errorf("start runtime: %w", startupCtx.Err())
	}
	return true, nil
}
