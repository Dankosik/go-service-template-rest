// Package background supervises non-HTTP work for the lifetime of the process.
//
// Without a seam here, the first scheduled job a service adds gets started with
// a bare `go` against context.Background(). Two things follow. On SIGTERM the
// HTTP server drains correctly and the job is killed mid-iteration by process
// exit, so a job that writes to a database leaves a partial batch with no
// compensating record. And a panic inside it takes the whole process down —
// Recover is HTTP middleware and never sees it — during a window that may carry
// no HTTP traffic at all, so the only artifact is a restart.
//
// This is a supervisor, not a job framework. It owns cancellation, panic
// containment, and the join; schedules, retries, and locking belong to the task.
package background

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ErrPanic marks a task failure that came from a recovered panic rather than a
// returned error, so a caller can tell a bug from an expected failure.
var ErrPanic = errors.New("background task panicked")

// Task is one supervised unit of work. Run must return when its context is done;
// a task that ignores cancellation is bounded only by the shutdown budget.
type Task struct {
	Name string
	Run  func(context.Context) error
}

// Supervisor runs tasks under one cancelable context and joins them on shutdown.
type Supervisor struct {
	log *slog.Logger

	group *errgroup.Group
	// groupCtx is the context handed to every task. It is stored rather than
	// passed to Go because cancellation has to reach tasks registered at
	// different times, and because errgroup.WithContext cancels it when any task
	// fails — which is the coordination this type exists to provide.
	//nolint:containedctx // A supervisor's whole job is owning the lifetime it hands out.
	groupCtx  context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	stopOnce  sync.Once
	stopErr   error
}

// New builds a supervisor whose tasks are canceled when ctx is done or Shutdown
// is called, whichever happens first.
func New(ctx context.Context, log *slog.Logger) *Supervisor {
	if log == nil {
		log = slog.Default()
	}
	taskCtx, cancel := context.WithCancel(ctx)
	group, groupCtx := errgroup.WithContext(taskCtx)

	return &Supervisor{
		log:      log,
		group:    group,
		groupCtx: groupCtx,
		cancel:   cancel,
	}
}

// Go starts task. A panic inside Run is recovered and converted into an error so
// one bad iteration cannot take the process down, while a task that keeps
// panicking still surfaces through Shutdown rather than being swallowed.
func (s *Supervisor) Go(task Task) {
	name := task.Name
	if name == "" {
		name = "unnamed"
	}
	if task.Run == nil {
		s.log.Error("background_task_invalid", "component", "background", "task", name, "reason", "run is nil")
		return
	}

	s.startOnce.Do(func() {
		s.log.Info("background_supervisor_started", "component", "background")
	})
	s.log.Info("background_task_started", "component", "background", "task", name)

	s.group.Go(func() (runErr error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			// The stack is captured here, inside the deferred function, because
			// it is the only place the panicking goroutine's frames still exist.
			s.log.Error(
				"background_task_panic",
				"component", "background",
				"task", name,
				"panic_type", fmt.Sprintf("%T", recovered),
				"stack", string(debug.Stack()),
			)
			runErr = fmt.Errorf("%w: %s", ErrPanic, name)
		}()

		if err := task.Run(s.groupCtx); err != nil {
			// Cancellation is how shutdown asks a task to stop, so it is not a
			// task failure and must not poison the group's error.
			if errors.Is(err, context.Canceled) && s.groupCtx.Err() != nil {
				s.log.Info("background_task_canceled", "component", "background", "task", name)
				return nil
			}
			s.log.Error("background_task_failed", "component", "background", "task", name, "err", err)
			return fmt.Errorf("background task %s: %w", name, err)
		}
		s.log.Info("background_task_stopped", "component", "background", "task", name)
		return nil
	})
}

// Shutdown cancels every task and waits for them to return, bounded by ctx.
//
// A task that outlives ctx is reported rather than waited on: blocking here
// would hold up the rest of shutdown — including the telemetry flush that has to
// record this exact outcome — behind work that has already ignored its
// cancellation once.
func (s *Supervisor) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() {
		s.cancel()

		joined := make(chan error, 1)
		go func() { joined <- s.group.Wait() }()

		select {
		case err := <-joined:
			s.stopErr = err
		case <-ctx.Done():
			s.stopErr = fmt.Errorf("background tasks did not stop within the shutdown budget: %w", ctx.Err())
		}

		if s.stopErr != nil {
			s.log.Error("background_shutdown_failed", "component", "background", "err", s.stopErr)
			return
		}
		s.log.Info("background_shutdown_completed", "component", "background")
	})
	return s.stopErr
}
