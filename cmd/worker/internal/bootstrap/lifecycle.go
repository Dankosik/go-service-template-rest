package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

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
	go func() {
		defer close(workerDone)
		workerResult <- worker.Run(runtimeCtx)
	}()
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
