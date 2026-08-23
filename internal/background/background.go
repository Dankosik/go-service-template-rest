// Package background supervises non-HTTP work for the lifetime of the process.
//
// It exists because a bare `go` against context.Background() is killed
// mid-iteration by process exit on SIGTERM, and a panic inside it takes the
// process down — Recover is HTTP middleware and never sees it.
//
// This is a supervisor, not a job framework. It owns cancellation, panic
// containment, the join, and reporting a task that failed; schedules, retries,
// and locking belong to the task.
package background

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrPanic marks a task failure that came from a recovered panic rather than
	// a returned error, so a caller can tell a bug from an expected failure.
	ErrPanic = errors.New("background task panicked")

	ErrTaskFailed = errors.New("background task failed")

	// ErrTaskStopped reports a task that returned without shutdown asking it to.
	ErrTaskStopped = errors.New("background task stopped unexpectedly")
)

// Task is one supervised unit of work. Run must return when its context is done;
// a task that ignores cancellation is bounded only by the shutdown budget.
type Task struct {
	Name string
	Run  func(context.Context) error
}

// Supervisor runs tasks under one cancelable context and joins them on shutdown.
type Supervisor struct {
	log *slog.Logger

	// group joins the tasks and keeps the first error for Shutdown. It is a bare
	// errgroup.Group rather than errgroup.WithContext on purpose; see taskCtx.
	group *errgroup.Group
	// taskCtx is canceled only by New's parent or by Shutdown.
	//
	// It is deliberately not an errgroup.WithContext context: that one is
	// canceled the first time any task returns an error, so one failing worker
	// stops every other supervised task — including the readiness refresher,
	// which returns nil on cancellation and exits through the ordinary stopped
	// path. Keeping sibling cancellation under Shutdown preserves the ordered
	// drain.
	//nolint:containedctx // A supervisor's whole job is owning the lifetime it hands out.
	taskCtx   context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	stopOnce  sync.Once
	stopErr   error

	// stopping reports that Shutdown has begun, so a task ending during the drain
	// is an ordinary stop rather than something readiness has to report.
	stopping atomic.Bool
	// failure holds the first task that ended with an error while the process was
	// still serving.
	failure atomic.Pointer[taskFailure]
	// failures delivers that same first failure to the process lifecycle owner.
	// It is buffered because the failing task must never wait for that owner.
	failures chan error
}

type taskFailure struct {
	task string
	err  error
}

// New builds a supervisor whose tasks are canceled when ctx is done or Shutdown
// is called, whichever happens first.
func New(ctx context.Context, log *slog.Logger) *Supervisor {
	if log == nil {
		log = slog.Default()
	}
	taskCtx, cancel := context.WithCancel(ctx)

	return &Supervisor{
		log:      log,
		group:    new(errgroup.Group),
		taskCtx:  taskCtx,
		cancel:   cancel,
		failures: make(chan error, 1),
	}
}

// Go starts task. A panic inside Run is recovered and converted into an error so
// the process can run its ordered drain instead of losing shutdown telemetry.
func (s *Supervisor) Go(task Task) {
	name := cmp.Or(task.Name, "unnamed")
	if task.Run == nil {
		s.log.Error("background_task_invalid", "component", "background", "task", name, "reason", "run is nil")
		s.recordStop(name, errors.New("run is nil"))
		return
	}

	s.startOnce.Do(func() {
		s.log.Info("background_supervisor_started", "component", "background")
	})
	s.log.Info("background_task_started", "component", "background", "task", name)

	s.group.Go(func() error {
		err := s.runTask(name, task.Run)
		s.recordStop(name, err)
		return err
	})
}

func (s *Supervisor) runTask(name string, run func(context.Context) error) (runErr error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		// Captured here, inside the deferred function: the only place the
		// panicking goroutine's frames still exist.
		s.log.Error(
			"background_task_panic",
			append(
				[]any{"component", "background", "task", name},
				logctx.PanicAttrs(recovered, debug.Stack())...,
			)...,
		)
		runErr = fmt.Errorf("%w: %s", ErrPanic, name)
	}()

	if err := run(s.taskCtx); err != nil {
		// The parent context and Shutdown own this context's terminal error, so a
		// task returning that same cause stopped as asked rather than failed.
		if taskErr := s.taskCtx.Err(); taskErr != nil && errors.Is(err, taskErr) {
			s.log.Info("background_task_canceled", "component", "background", "task", name)
			return nil
		}
		s.log.Error("background_task_failed", "component", "background", "task", name, "err", err)
		return fmt.Errorf("background task %s: %w", name, err)
	}
	if s.taskCtx.Err() != nil {
		s.log.Info("background_task_stopped", "component", "background", "task", name)
		return nil
	}
	err := fmt.Errorf("%w: %s", ErrTaskStopped, name)
	s.log.Error("background_task_failed", "component", "background", "task", name, "err", err)
	return err
}

// recordStop publishes a task that failed while the process was still serving.
// A task ending during the drain is an ordinary stop; any other return means the
// process is running without work it was built to do.
func (s *Supervisor) recordStop(name string, err error) {
	if err == nil || s.stopping.Load() || s.taskCtx.Err() != nil {
		return
	}
	recorded := &taskFailure{task: name, err: err}
	if s.failure.CompareAndSwap(nil, recorded) {
		s.failures <- taskFailureError(recorded)
	}
}

func (s *Supervisor) Failures() <-chan error {
	return s.failures
}

func (s *Supervisor) Name() string {
	return "background"
}

// Check makes the first failure visible to readiness. Tasks are not restarted
// here: whether a failure is recoverable depends on what the task already did, so
// the honest signal is to stop taking traffic and let the platform's restart
// policy start a clean process.
func (s *Supervisor) Check(context.Context) error {
	recorded := s.failure.Load()
	if recorded == nil {
		return nil
	}
	return taskFailureError(recorded)
}

func taskFailureError(recorded *taskFailure) error {
	return fmt.Errorf("%w: %s: %w", ErrTaskFailed, recorded.task, recorded.err)
}

// Shutdown cancels every task and waits for them to return, bounded by ctx.
//
// A task that outlives ctx is reported rather than waited on: blocking here
// would hold up the rest of shutdown — including the telemetry flush that has to
// record this exact outcome — behind work that has already ignored its
// cancellation once.
func (s *Supervisor) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() {
		// Marked before the cancel, so a task that ends because of it is not
		// recorded as a failure readiness has to report.
		s.stopping.Store(true)
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
			s.log.ErrorContext(ctx, "background_shutdown_failed", "component", "background", "err", s.stopErr)
			return
		}
		s.log.InfoContext(ctx, "background_shutdown_completed", "component", "background")
	})
	return s.stopErr
}
