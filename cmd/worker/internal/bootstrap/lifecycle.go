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
// by handlerStoppedBeforeReturn, which asks the drain instead, because the
// handlers a panicking fetch loop leaves running are the ones that matter.
var errWorkerPanic = errors.New("worker run loop panicked")

// The post-drain budgets, kept together because validateWorkerShutdownBudget
// charges http.grace_period for their sum and every one of them is spent on the
// same shutdown path.
const (
	diagnosticsClose = 2 * time.Second
	backgroundClose  = 5 * time.Second
	handlerClose     = 5 * time.Second
	telemetryClose   = 5 * time.Second
)

func validateWorkerShutdownBudget(gracePeriod, drainTimeout time.Duration) error {
	return runtimeopts.ValidateGracePeriod(
		gracePeriod,
		"messaging.worker.drain_timeout",
		drainTimeout,
		diagnosticsClose+backgroundClose+handlerClose+telemetryClose,
	)
}

func runWorkerLifecycle(
	signalCtx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
	metrics *telemetry.Metrics,
	client *natsjs.Client,
	worker *natsjs.Worker,
) (bool, time.Time, error) {
	healthSvc := health.New(client)
	if err := healthSvc.Refresh(startupCtx, cfg.HTTP.ReadinessTimeout, cfg.Health.FailureThreshold); err != nil {
		return true, time.Time{}, fmt.Errorf("admit worker readiness: %w", err)
	}
	diagnostics, err := runtimeopts.ListenDiagnostics(
		startupCtx, cfg.Observability.Metrics.Addr, "worker", workerReady(client.Ready, healthSvc), metrics,
	)
	if err != nil {
		return true, time.Time{}, err
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
	workerResultRead := false
	select {
	case <-signalCtx.Done():
	case triggerErr = <-supervisor.Failures():
	case triggerErr = <-workerResult:
		workerResultRead = true
	case <-diagnostics.Stopped():
		// diagnostics.Stop below carries whatever Serve reported.
		triggerErr = errors.New("worker diagnostics stopped unexpectedly")
	}
	if signalCtx.Err() == nil {
		if triggerErr == nil {
			triggerErr = errors.New("worker runtime stopped unexpectedly")
		}
	}
	healthSvc.StartDrain()
	worker.StartDrain()
	processCtx, processCancel, shutdownDeadline := runtimeopts.ArmTeardown(signalCtx, cfg.HTTP.GracePeriod)
	defer processCancel()
	workerCtx, workerCancel := context.WithTimeout(processCtx, cfg.Messaging.Worker.DrainTimeout)
	workerErr := worker.Shutdown(workerCtx)
	workerCancel()
	diagnosticsErr := diagnostics.Stop(processCtx, diagnosticsClose)
	backgroundCtx, backgroundCancel := context.WithTimeout(processCtx, backgroundClose)
	backgroundErr := supervisor.Shutdown(backgroundCtx)
	backgroundCancel()
	cleanupSafe := handlerStoppedBeforeReturn(workerErr, workerDone)
	if !workerResultRead {
		select {
		case runErr := <-workerResult:
			if triggerErr == nil {
				triggerErr = runErr
			}
		default:
		}
	}
	return cleanupSafe, shutdownDeadline, errors.Join(triggerErr, diagnosticsErr, workerErr, backgroundErr)
}

// workerReady is this binary's readiness verdict, which runtimeopts.DiagnosticsServer
// serves as one answer.
//
// The two conditions are separate facts and both have to hold. The broker state
// is read live rather than through a probe, because a consumer that lost its
// connection is not ready now and waiting for the next refresh would keep it in
// rotation for up to one interval. The cached verdict covers everything else the
// process probes.
func workerReady(messagingReady func() bool, healthSvc *health.Service) func() bool {
	return func() bool {
		return messagingReady() && healthSvc.Cached() == nil
	}
}

// superviseWorkerRun contains a panic in the loop this process exists to run.
// runWorkerLifecycle's two background helpers are background.Supervisor tasks,
// which recover for themselves; this loop cannot be one, because the drain reads
// its exit through done and charges it messaging.worker.drain_timeout rather
// than the supervisor's own budget. Left bare, a panic here ends the process
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
				append([]any{"component", "worker"}, logctx.PanicAttrs(recovered, debug.Stack())...)...,
			)
			runErr = errWorkerPanic
		}
		result <- runErr
	}()
	runErr = run(ctx)
}

func handlerStoppedBeforeReturn(workerErr error, workerDone <-chan struct{}) bool {
	if workerErr == nil {
		<-workerDone
		return true
	}
	select {
	case <-workerDone:
		return true
	default:
		// A handler that ignored forced cancellation may still use its
		// dependencies. Process exit owns their cleanup in this path.
		return false
	}
}
