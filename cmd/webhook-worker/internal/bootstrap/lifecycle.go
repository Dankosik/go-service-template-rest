package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

type lifecycleResult struct {
	CleanupSafe bool
	Err         error
}

var errWebhookWorkerPanic = errors.New("webhook worker panicked")

func runLifecycle(signalCtx, startupCtx context.Context, cfg config.Config, metrics *telemetry.Metrics, worker *postgreswebhook.Worker) lifecycleResult {
	diagnostics, err := runtimeopts.ListenDiagnostics(startupCtx, cfg.Observability.Metrics.Addr, "webhook-worker", worker.Ready, metrics)
	if err != nil {
		return lifecycleResult{CleanupSafe: true, Err: err}
	}
	runCtx, cancelRun := context.WithCancel(context.WithoutCancel(signalCtx))
	runResult := make(chan postgreswebhook.WorkerResult, 1)
	go func() {
		var result postgreswebhook.WorkerResult
		defer func() {
			if recover() != nil {
				result = postgreswebhook.WorkerResult{Err: errWebhookWorkerPanic, CleanupUnsafe: true}
			}
			runResult <- result
		}()
		result = worker.Run(runCtx)
	}()
	var trigger error
	var result postgreswebhook.WorkerResult
	workerStopped := false
	select {
	case <-signalCtx.Done():
	case result = <-runResult:
		workerStopped = true
		trigger = result.Err
		if trigger == nil {
			trigger = errors.New("webhook worker stopped unexpectedly")
		}
	case <-diagnostics.Stopped():
		trigger = errors.New("webhook worker diagnostics stopped unexpectedly")
	}
	processCtx, cancelProcess, _ := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer cancelProcess()
	cancelRun()
	cleanupSafe := true
	var workerErr error
	if !workerStopped {
		select {
		case result = <-runResult:
			workerErr = result.Err
		case <-processCtx.Done():
			cleanupSafe = false
			workerErr = fmt.Errorf("join webhook worker: %w", processCtx.Err())
		}
	}
	if result.CleanupUnsafe || errors.Is(workerErr, postgreswebhook.ErrDrainUnsafe) {
		cleanupSafe = false
	}
	diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
	return lifecycleResult{CleanupSafe: cleanupSafe, Err: errors.Join(trigger, workerErr, diagnosticsErr)}
}
