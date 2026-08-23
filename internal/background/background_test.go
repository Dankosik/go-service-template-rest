package background

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestSupervisorRunsTaskUntilShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var iterations atomic.Int64
		sup := New(context.Background(), discardLogger())
		sup.Go(Task{Name: "ticker", Run: func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
					iterations.Add(1)
				}
			}
		}})

		synctest.Sleep(3 * time.Second)
		if got := iterations.Load(); got != 3 {
			t.Fatalf("iterations = %d, want 3", got)
		}

		if err := sup.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v, want nil", err)
		}
	})
}

// TestPanicInTaskDoesNotKillProcess is the property this package exists for:
// Recover is HTTP middleware and never sees a background goroutine.
func TestPanicInTaskDoesNotKillProcess(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	sup := New(context.Background(), slog.New(slog.NewJSONHandler(&logged, nil)))
	sup.Go(Task{Name: "exploder", Run: func(context.Context) error {
		panic("boom")
	}})

	err := sup.Shutdown(context.Background())
	if !errors.Is(err, ErrPanic) {
		t.Fatalf("Shutdown() error = %v, want ErrPanic", err)
	}
	if !strings.Contains(err.Error(), "exploder") {
		t.Fatalf("error = %q, want it to name the task", err.Error())
	}

	line := logged.String()
	if !strings.Contains(line, `"background_task_panic"`) {
		t.Fatalf("log = %q, want a panic record", line)
	}
	if !strings.Contains(line, `"stack"`) {
		t.Fatalf("log = %q, want a captured stack", line)
	}
}

// TestOnePanicDoesNotStopSiblingWork keeps a single bad task from silently
// disarming the rest of the process's background work before shutdown.
func TestOnePanicDoesNotStopSiblingWork(t *testing.T) {
	t.Parallel()

	healthyStopped := make(chan struct{})
	sup := New(context.Background(), discardLogger())
	sup.Go(Task{Name: "exploder", Run: func(context.Context) error {
		panic("boom")
	}})
	sup.Go(Task{Name: "healthy", Run: func(ctx context.Context) error {
		<-ctx.Done()
		close(healthyStopped)
		return ctx.Err()
	}})

	if err := sup.Shutdown(context.Background()); !errors.Is(err, ErrPanic) {
		t.Fatalf("Shutdown() error = %v, want ErrPanic", err)
	}
	waittest.ReceiveSignal(t, healthyStopped, 2*time.Second, "sibling task cancellation")
}

// TestFailedTaskLeavesSiblingsRunning is the isolation property, asserted before
// any Shutdown so it cannot be satisfied by the shutdown cancel.
//
// The supervisor used to hand tasks an errgroup.WithContext context, which is
// canceled the first time any task returns an error. One failing worker therefore
// stopped every other supervised task in the process, including the readiness
// refresher — which returns nil on cancellation and so left no trace beyond an
// INFO line, while readiness kept serving the verdict it happened to hold.
func TestFailedTaskLeavesSiblingsRunning(t *testing.T) {
	t.Parallel()

	sup := New(context.Background(), discardLogger())

	siblingCtx := make(chan context.Context, 1)
	sup.Go(Task{Name: "sibling", Run: func(ctx context.Context) error {
		siblingCtx <- ctx
		<-ctx.Done()
		return nil
	}})
	ctx := <-siblingCtx
	taskErr := errors.New("consumer lost its lease")
	sup.Go(Task{Name: "failing", Run: func(context.Context) error { return taskErr }})

	if err := <-sup.Failures(); !errors.Is(err, taskErr) {
		t.Fatalf("Failures() error = %v, want %v", err, taskErr)
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("sibling task was canceled by an unrelated task failure: %v", err)
	}

	if err := sup.Shutdown(context.Background()); err == nil {
		t.Fatal("Shutdown() error = nil, want the failing task's error")
	}
}

// TestCheckReportsAFailedTask is what makes a dead worker reach readiness instead
// of only the shutdown log.
func TestCheckReportsAFailedTask(t *testing.T) {
	t.Parallel()

	taskErr := errors.New("outbox publish failed")
	sup := New(context.Background(), discardLogger())

	if err := sup.Check(context.Background()); err != nil {
		t.Fatalf("Check() before any failure = %v, want nil", err)
	}

	sup.Go(Task{Name: "outbox", Run: func(context.Context) error { return taskErr }})
	<-sup.Failures()

	err := sup.Check(context.Background())
	if !errors.Is(err, ErrTaskFailed) || !errors.Is(err, taskErr) {
		t.Fatalf("Check() = %v, want ErrTaskFailed wrapping %v", err, taskErr)
	}
	if !strings.Contains(err.Error(), "outbox") {
		t.Fatalf("Check() = %q, want it to name the task", err.Error())
	}
	if sup.Name() != "background" {
		t.Fatalf("Name() = %q, want \"background\"", sup.Name())
	}
}

// TestCheckIgnoresTasksStoppedByShutdown keeps an ordinary drain from being
// reported as a readiness failure, which would fail the probe of every instance
// that is shutting down cleanly.
func TestCheckIgnoresTasksStoppedByShutdown(t *testing.T) {
	t.Parallel()

	sup := New(context.Background(), discardLogger())
	sup.Go(Task{Name: "worker", Run: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}})

	if err := sup.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
	if err := sup.Check(context.Background()); err != nil {
		t.Fatalf("Check() after a clean shutdown = %v, want nil", err)
	}
}

func TestUnexpectedTaskStopFailsTheProcess(t *testing.T) {
	t.Parallel()

	sup := New(context.Background(), discardLogger())
	sup.Go(Task{Name: "one_shot", Run: func(context.Context) error { return nil }})

	err := <-sup.Failures()
	if !errors.Is(err, ErrTaskFailed) || !errors.Is(err, ErrTaskStopped) {
		t.Fatalf("Failures() = %v, want ErrTaskFailed wrapping ErrTaskStopped", err)
	}
}

func TestTaskErrorIsReported(t *testing.T) {
	t.Parallel()

	taskErr := errors.New("consumer lost its lease")
	sup := New(context.Background(), discardLogger())
	sup.Go(Task{Name: "consumer", Run: func(context.Context) error { return taskErr }})

	err := sup.Shutdown(context.Background())
	if !errors.Is(err, taskErr) {
		t.Fatalf("Shutdown() error = %v, want wrapped %v", err, taskErr)
	}
}

// TestCancellationIsNotATaskFailure keeps ordinary shutdown from being reported
// as an error, which would make every clean stop look like a fault.
func TestCancellationIsNotATaskFailure(t *testing.T) {
	t.Parallel()

	sup := New(context.Background(), discardLogger())
	sup.Go(Task{Name: "worker", Run: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}})

	if err := sup.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}

// TestShutdownIsBoundedByItsContext keeps a task that ignores cancellation from
// holding up the rest of shutdown, including the telemetry flush that has to
// record this outcome.
func TestShutdownIsBoundedByItsContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		stuck := make(chan struct{})
		defer close(stuck)

		sup := New(context.Background(), discardLogger())
		sup.Go(Task{Name: "stuck", Run: func(context.Context) error {
			<-stuck
			return nil
		}})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := sup.Shutdown(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Shutdown() error = %v, want context.DeadlineExceeded", err)
		}
		if !strings.Contains(err.Error(), "did not stop within the shutdown budget") {
			t.Fatalf("error = %q, want it to name the budget", err.Error())
		}
	})
}

func TestParentDeadlineStopsTasksWithoutFailure(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		parent, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		sup := New(parent, discardLogger())
		sup.Go(Task{Name: "worker", Run: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}})

		time.Sleep(time.Second)
		synctest.Wait()
		if err := sup.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() after parent deadline error = %v, want nil", err)
		}
	})
}

// TestParentCancellationStopsTasks keeps the supervisor honest about the context
// it was built with, not only about explicit Shutdown calls.
func TestParentCancellationStopsTasks(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})

	sup := New(ctx, discardLogger())
	sup.Go(Task{Name: "worker", Run: func(taskCtx context.Context) error {
		<-taskCtx.Done()
		close(stopped)
		return taskCtx.Err()
	}})

	cancel()
	waittest.ReceiveSignal(t, stopped, 2*time.Second, "task cancellation by parent context")
	if err := sup.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	t.Parallel()

	taskErr := errors.New("failed once")
	sup := New(context.Background(), discardLogger())
	sup.Go(Task{Name: "worker", Run: func(context.Context) error { return taskErr }})

	first := sup.Shutdown(context.Background())
	second := sup.Shutdown(context.Background())
	if !errors.Is(first, taskErr) || !errors.Is(second, taskErr) {
		t.Fatalf("Shutdown() results = %v / %v, want both to wrap %v", first, second, taskErr)
	}
}

func TestNilRunIsRejectedWithoutStarting(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	sup := New(context.Background(), slog.New(slog.NewJSONHandler(&logged, nil)))
	sup.Go(Task{Name: "broken"})

	if err := <-sup.Failures(); !errors.Is(err, ErrTaskFailed) {
		t.Fatalf("Failures() = %v, want ErrTaskFailed", err)
	}
	if !strings.Contains(logged.String(), "background_task_invalid") {
		t.Fatalf("log = %q, want an invalid-task record", logged.String())
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
