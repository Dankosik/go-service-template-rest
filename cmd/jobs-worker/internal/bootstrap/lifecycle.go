package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type lifecycleEngine interface {
	Run(ctx context.Context) error
	Terminal() <-chan error
	Facts() postgresjobs.EngineFacts
	StartDrain(ctx context.Context) postgresjobs.DrainResult
}

type lifecycleResult struct {
	CleanupSafe bool
	Deadline    time.Time
	Err         error
}

//nolint:cyclop // The explicit signal, terminal, diagnostics, drain, and cleanup branches are the worker lifecycle contract.
func runLifecycle(signalCtx, startupCtx context.Context, cfg config.Config, metrics *telemetry.Metrics, engine lifecycleEngine) lifecycleResult {
	var ready atomic.Bool
	ready.Store(true)
	diagnostics, err := runtimeopts.ListenDiagnostics(startupCtx, cfg.Observability.Metrics.Addr, "jobs-worker", func() bool {
		facts := engine.Facts()
		return ready.Load() && facts.ClaimAdmissionOpen && facts.Compatible && facts.ObservationFresh
	}, metrics)
	if err != nil {
		return lifecycleResult{CleanupSafe: true, Err: err}
	}
	runCtx, cancelRun := context.WithCancel(context.WithoutCancel(signalCtx))
	defer cancelRun()
	runErr := make(chan error, 1)
	go func() {
		if err := engine.Run(runCtx); err != nil {
			runErr <- err
			return
		}
		ticker := time.NewTicker(cfg.Jobs.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				runErr <- runCtx.Err()
				return
			case <-ticker.C:
				if err := engine.Run(runCtx); err != nil {
					runErr <- err
					return
				}
			}
		}
	}()

	var trigger error
	runnerStopped := false
	select {
	case <-signalCtx.Done():
	case trigger = <-runErr:
		runnerStopped = true
		if trigger == nil || errors.Is(trigger, context.Canceled) {
			trigger = errors.New("jobs worker stopped unexpectedly")
		}
	case trigger = <-engine.Terminal():
	case <-diagnostics.Stopped():
		trigger = errors.New("jobs worker diagnostics stopped unexpectedly")
	}
	// This assignment precedes StartDrain: the diagnostics endpoint must stop
	// advertising readiness before claim quiescence starts.
	ready.Store(false)
	processCtx, cancelProcess, deadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer cancelProcess()
	drain := engine.StartDrain(processCtx)
	if !drain.CleanupSafe {
		recordUnsafeDrain()
		return lifecycleResult{Deadline: deadline, Err: errors.Join(trigger, drain.Err)}
	}
	cancelRun()
	var runnerErr error
	cleanupSafe := true
	if !runnerStopped {
		select {
		case runnerErr = <-runErr:
			if errors.Is(runnerErr, context.Canceled) {
				runnerErr = nil
			}
		case <-processCtx.Done():
			cleanupSafe = false
			runnerErr = fmt.Errorf("join jobs worker coordinator: %w", processCtx.Err())
		}
	}
	diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
	return lifecycleResult{CleanupSafe: cleanupSafe, Deadline: deadline, Err: errors.Join(trigger, drain.Err, runnerErr, diagnosticsErr)}
}
