package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

// errWorkerPanic reports a run loop that ended in a recovered panic rather than
// by returning. It explains the exit; whether cleanup is still safe is decided
// by runtimeopts.StoppedBeforeReturn, which asks the drain instead, because the
// handlers a panicking consume loop leaves running are the ones that matter.
var errWorkerPanic = errors.New("worker run loop panicked")

const (
	diagnosticsShutdownTimeout = 5 * time.Second
	backgroundShutdownTimeout  = 5 * time.Second
	handlerShutdownTimeout     = 5 * time.Second
	telemetryShutdownTimeout   = 5 * time.Second
	workerTailBudget           = diagnosticsShutdownTimeout + backgroundShutdownTimeout + handlerShutdownTimeout + telemetryShutdownTimeout
)

// runWorkerLifecycle admits, serves, and drains the consumer. cleanupSafe
// reports whether the handler joined, so run may release what it uses. The
// returned window is already canceled and serves only as the parent for
// runtimeopts.TeardownStage in deferred cleanup.
func runWorkerLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	client *natsjs.Client,
	worker *natsjs.Worker,
) (cleanupSafe bool, window context.Context, err error) {
	unarmed := runtimeopts.UnarmedTeardown(signalCtx)
	healthSvc := health.New(client)
	if err := healthSvc.Refresh(startupCtx, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold); err != nil {
		return true, unarmed, fmt.Errorf("admit worker readiness: %w", err)
	}
	diagnostics, err := runtimeopts.ListenDiagnostics(
		startupCtx,
		cfg.Observability.Metrics.Addr,
		"worker",
		func() bool { return healthSvc.Cached() == nil },
		metrics,
		cfg.Observability.Pprof.Enabled,
	)
	if err != nil {
		return true, unarmed, err
	}

	runtimeCtx := context.WithoutCancel(signalCtx)
	supervisor := background.New(runtimeCtx, log)
	supervisor.Go(background.Task{Name: "messaging_connection", Run: client.Run})
	supervisor.Go(background.Task{
		Name: "messaging_readiness",
		Run: func(ctx context.Context) error {
			return healthSvc.Watch(ctx, cfg.Health.RefreshInterval, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold, nil)
		},
	})
	workerResult := make(chan error, 1)
	workerDone := make(chan struct{})
	go superviseWorkerRun(runtimeCtx, log, worker.Run, workerResult, workerDone)
	var triggerErr error
	select {
	case <-signalCtx.Done():
	case triggerErr = <-supervisor.FirstFailure():
	case triggerErr = <-workerResult:
	case <-diagnostics.Stopped():
		// diagnostics.Stop below carries whatever Serve reported.
		triggerErr = errors.New("worker diagnostics stopped unexpectedly")
	}
	if triggerErr == nil && signalCtx.Err() == nil {
		triggerErr = errors.New("worker runtime stopped unexpectedly")
	}
	healthSvc.StartDrain()
	worker.StartDrain()
	var processCancel context.CancelFunc
	window, processCancel = runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer processCancel()
	workerCtx, workerCancel := runtimeopts.TeardownStage(window, cfg.HTTP.ShutdownTimeout)
	workerErr := worker.Shutdown(workerCtx)
	workerCancel()
	diagnosticsErr := diagnostics.Stop(window, diagnosticsShutdownTimeout)
	backgroundCtx, backgroundCancel := runtimeopts.TeardownStage(window, backgroundShutdownTimeout)
	backgroundErr := supervisor.Shutdown(backgroundCtx)
	backgroundCancel()
	cleanupSafe = runtimeopts.StoppedBeforeReturn(workerErr, workerDone)
	select {
	case runErr := <-workerResult:
		if triggerErr == nil {
			triggerErr = runErr
		}
	default:
	}
	return cleanupSafe, window, errors.Join(
		triggerErr,
		workerLifecycleError("worker shutdown", workerErr),
		workerLifecycleError("diagnostics shutdown", diagnosticsErr),
		workerLifecycleError("background shutdown", backgroundErr),
	)
}

func workerLifecycleError(stage string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", stage, err)
}

// superviseWorkerRun contains a panic in the loop this process exists to run.
// runWorkerLifecycle's two background helpers are background.Supervisor tasks,
// which recover for themselves; this loop cannot be one, because the drain reads
// its exit through done under the process grace budget. Left bare, a panic here ends the process
// where it happened, without the ordered drain, the handler join, or the
// telemetry flush that would record why.
//
// The result is assigned and then sent from the deferred call, so the normal
// path and the recovery send exactly once between them, and done closes after
// that send for the caller reading them in that order.
func superviseWorkerRun(
	ctx context.Context,
	log *slog.Logger,
	run func(context.Context) error,
	result chan<- error,
	done chan<- struct{},
) {
	var runErr error
	defer func() {
		defer close(done)
		if recovered := recover(); recovered != nil {
			// Written from inside the deferred recovery, the one point the
			// panicking frames still exist. The sentinel names only that a panic
			// happened; this is the only record of the defect behind it.
			log.ErrorContext(
				ctx,
				"worker_run_loop_panic",
				append([]any{"component", "worker"}, logctx.PanicArgs(recovered, debug.Stack())...)...,
			)
			runErr = errWorkerPanic
		}
		result <- runErr
	}()
	runErr = run(ctx)
}
