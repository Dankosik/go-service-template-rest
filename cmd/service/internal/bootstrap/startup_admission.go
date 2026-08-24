package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

var errStartupAdmissionPending = errors.New("startup admission is not ready")

type startupAdmissionController struct {
	ready atomic.Bool
}

// The methods below carry no nil-receiver guard. The controller is constructed
// unconditionally in Run and reaches every caller through a field that is always
// set, so a nil check here is an unreachable branch on the path a readiness probe
// takes for the life of the pod — and one that would turn a wiring mistake into a
// permanent 503 instead of a panic on the first request.

func (c *startupAdmissionController) MarkReady() {
	c.ready.Store(true)
}

func (c *startupAdmissionController) Ready() bool {
	return c.ready.Load()
}

func (c *startupAdmissionController) CheckReady(context.Context) error {
	if !c.Ready() {
		return errStartupAdmissionPending
	}
	return nil
}

// startStartupAdmission runs the readiness check once, off the serve path, and
// reports its verdict on the returned channel. serveRuntime is already selecting
// on the signal, the startup budget, and the servers by the time this resolves.
func startStartupAdmission(
	bootstrapCtx context.Context,
	readinessCheck func(context.Context) error,
	readinessTimeout time.Duration,
) <-chan error {
	resultCh := make(chan error, 1)

	go func() {
		readyCtx, cancel := withStageBudget(bootstrapCtx, readinessTimeout)
		defer cancel()

		if err := readyCtx.Err(); err != nil {
			resultCh <- fmt.Errorf("startup admission context: %w", err)
			return
		}
		if readinessCheck != nil {
			if err := readinessCheck(readyCtx); err != nil {
				resultCh <- err
				return
			}
		}
		if err := readyCtx.Err(); err != nil {
			resultCh <- fmt.Errorf("startup admission context: %w", err)
			return
		}
		resultCh <- nil
	}()

	return resultCh
}

// waitForStartupAdmission decides whether this instance may start answering live
// traffic. It races the readiness verdict against the shutdown signal, the
// startup budget, an early server exit, and a background task failure, and flips
// the controller above only when readiness wins outright.
//
// The nested select after a successful verdict is not redundant: readiness can
// resolve in the same scheduling window as a server exit or a background
// failure, and admitting traffic to a process that is already failing is the one
// outcome this function exists to prevent.
func waitForStartupAdmission(
	signalCtx context.Context,
	bootstrapCtx context.Context,
	args serveRuntimeArgs,
	admissionErrCh <-chan error,
	runErrCh <-chan serverResult,
) (ready bool, stopRequested bool, terminalErr error) {
	select {
	case err := <-admissionErrCh:
		if err != nil {
			return false, false, rejectHTTPStartup(
				bootstrapCtx,
				args.log,
				"startup.readiness",
				fmt.Errorf("startup readiness check failed: %w", err),
			)
		}
		select {
		case result := <-runErrCh:
			return false, false, serverStoppedBeforeReadiness(bootstrapCtx, args, result)
		case err := <-args.backgroundFailures:
			return false, false, rejectHTTPStartup(
				bootstrapCtx,
				args.log,
				"startup.background",
				fmt.Errorf("background task failed before readiness: %w", err),
			)
		default:
			// profile:grpc:start
			if args.grpcSrv != nil {
				args.grpcSrv.SetServing(true)
			}
			// profile:grpc:end
			args.admission.MarkReady()
			if args.onReady != nil {
				args.onReady()
			}
			return true, false, nil
		}
	case <-signalCtx.Done():
		args.log.InfoContext(signalCtx, "shutdown signal received")
		return false, true, nil
	case <-bootstrapCtx.Done():
		select {
		case <-signalCtx.Done():
			args.log.InfoContext(signalCtx, "shutdown signal received")
			return false, true, nil
		default:
		}
		err := fmt.Errorf("startup budget exhausted before readiness: %w", bootstrapCtx.Err())
		args.log.ErrorContext(bootstrapCtx, "startup budget exhausted before readiness", "err", err)
		return false, false, err
	case result := <-runErrCh:
		return false, false, serverStoppedBeforeReadiness(bootstrapCtx, args, result)
	case err := <-args.backgroundFailures:
		return false, false, rejectHTTPStartup(
			bootstrapCtx,
			args.log,
			"startup.background",
			fmt.Errorf("background task failed before readiness: %w", err),
		)
	}
}

func serverStoppedBeforeReadiness(ctx context.Context, args serveRuntimeArgs, result serverResult) error {
	err := fmt.Errorf("%s server stopped before readiness", result.name)
	if result.err != nil {
		err = fmt.Errorf("%s server stopped before readiness: %w", result.name, result.err)
	}
	args.log.ErrorContext(
		ctx,
		"startup_blocked",
		startupLogArgs(
			startupLogComponentStartupProbes,
			result.name+"_serve",
			"error",
			"error.type", "startup_error",
			"err", err,
		)...,
	)
	return err
}
