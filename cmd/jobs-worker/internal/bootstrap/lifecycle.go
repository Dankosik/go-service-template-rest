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
	Run(context.Context) error
	Facts() postgresjobs.EngineFacts
	StartDrain(context.Context) postgresjobs.DrainResult
}

type lifecycleResult struct {
	CleanupSafe bool
	Err         error
}

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
	case <-diagnostics.Stopped():
		trigger = errors.New("jobs worker diagnostics stopped unexpectedly")
	}
	// This assignment precedes StartDrain: the diagnostics endpoint must stop
	// advertising readiness before claim quiescence starts.
	ready.Store(false)
	processCtx, cancelProcess, _ := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer cancelProcess()
	drain := engine.StartDrain(processCtx)
	cancelRun()
	var runnerErr error
	cleanupSafe := drain.CleanupSafe
	if cleanupSafe && !runnerStopped {
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
	return lifecycleResult{CleanupSafe: cleanupSafe, Err: errors.Join(trigger, drain.Err, runnerErr, diagnosticsErr)}
}
