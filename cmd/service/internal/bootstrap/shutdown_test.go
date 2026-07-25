package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// eventRecorder orders the shutdown steps a test observes.
//
// It is mutex-guarded because drainAndShutdown genuinely calls into the servers
// from several goroutines: Shutdown runs on one per server, and the force-close
// path runs on the caller's.
type eventRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *eventRecorder) record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *eventRecorder) observed() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

func (r *eventRecorder) count(event string) int {
	total := 0
	for _, observed := range r.observed() {
		if observed == event {
			total++
		}
	}
	return total
}

type fakeDrainer struct {
	events  *eventRecorder
	started bool
}

func (f *fakeDrainer) StartDrain() {
	f.started = true
	f.events.record("drain")
}

type fakeShutdownServer struct {
	events   *eventRecorder
	err      error
	onCalled func(context.Context) error
	closeErr error
}

func (f *fakeShutdownServer) Shutdown(ctx context.Context) error {
	f.events.record("shutdown")
	if f.onCalled != nil {
		if err := f.onCalled(ctx); err != nil {
			return err
		}
	}
	return f.err
}

// Close records that the drain gave up and abandoned in-flight connections.
func (f *fakeShutdownServer) Close() error {
	f.events.record("close")
	return f.closeErr
}

func shutdownTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestDrainAndShutdownOrdersDrainBeforeShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}

		srv := &fakeShutdownServer{
			events: events,
			onCalled: func(ctx context.Context) error {
				if !drainer.started {
					t.Fatal("shutdown called before drain started")
				}

				deadline, ok := ctx.Deadline()
				if !ok {
					t.Fatal("shutdown context has no deadline")
				}
				if remaining := time.Until(deadline); remaining != 30*time.Second {
					t.Fatalf("shutdown deadline remaining = %s, want exactly 30s", remaining)
				}

				return nil
			},
		}

		if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, 30*time.Second, drainer, srv); err != nil {
			t.Fatalf("drainAndShutdown() error = %v, want nil", err)
		}

		if got := strings.Join(events.observed(), ","); got != "drain,shutdown" {
			t.Fatalf("event order = %q, want %q", got, "drain,shutdown")
		}
	})
}

func TestDrainAndShutdownIgnoresParentCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events := &eventRecorder{}
	drainer := &fakeDrainer{events: events}
	srv := &fakeShutdownServer{
		events: events,
		onCalled: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				t.Fatalf("shutdown context err = %v, want nil", err)
			}
			return nil
		},
	}

	if err := drainAndShutdown(ctx, shutdownTestLogger(), 0, time.Second, drainer, srv); err != nil {
		t.Fatalf("drainAndShutdown() error = %v, want nil", err)
	}
}

func TestDrainAndShutdownPropagatesShutdownFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")

	events := &eventRecorder{}
	drainer := &fakeDrainer{events: events}
	srv := &fakeShutdownServer{
		events: events,
		err:    wantErr,
	}

	err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, time.Second, drainer, srv)
	if err == nil {
		t.Fatal("drainAndShutdown() error = nil, want non-nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("drainAndShutdown() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestDrainAndShutdownPropagatesContextCanceledError(t *testing.T) {
	t.Parallel()

	events := &eventRecorder{}
	drainer := &fakeDrainer{events: events}
	srv := &fakeShutdownServer{
		events: events,
		err:    context.Canceled,
	}

	err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, time.Second, drainer, srv)
	if err == nil {
		t.Fatal("drainAndShutdown() error = nil, want non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("drainAndShutdown() error = %v, want wrapped context.Canceled", err)
	}
}

func TestDrainAndShutdownRemainsBoundedWhenServerIgnoresContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		release := make(chan struct{})
		srv := &fakeShutdownServer{
			events: events,
			onCalled: func(context.Context) error {
				<-release
				return nil
			},
		}

		err := drainAndShutdown(
			context.Background(),
			shutdownTestLogger(),
			0,
			20*time.Millisecond,
			&fakeDrainer{events: events},
			srv,
		)
		close(release)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("drainAndShutdown() error = %v, want context deadline", err)
		}
	})
}

func TestDrainAndShutdownWaitsForPropagationDelay(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}
		startedAt := time.Now()

		srv := &fakeShutdownServer{
			events: events,
			onCalled: func(context.Context) error {
				if elapsed := time.Since(startedAt); elapsed != 20*time.Millisecond {
					t.Fatalf("shutdown called after %s, want exactly the 20ms propagation delay", elapsed)
				}
				return nil
			},
		}

		if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 20*time.Millisecond, time.Second, drainer, srv); err != nil {
			t.Fatalf("drainAndShutdown() error = %v, want nil", err)
		}
	})
}

func TestDrainAndShutdownCountsPropagationDelayAgainstShutdownTimeout(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}

		srv := &fakeShutdownServer{
			events: events,
			onCalled: func(ctx context.Context) error {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Fatal("shutdown context has no deadline")
				}
				if remaining := time.Until(deadline); remaining != 20*time.Millisecond {
					t.Fatalf("shutdown deadline remaining = %s, want exactly the 20ms left after the propagation delay", remaining)
				}
				return nil
			},
		}

		if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 20*time.Millisecond, 40*time.Millisecond, drainer, srv); err != nil {
			t.Fatalf("drainAndShutdown() error = %v, want nil", err)
		}
	})
}

func TestDrainAndShutdownWaitsForPropagationDelayDespiteCanceledParent(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}
		startedAt := time.Now()

		srv := &fakeShutdownServer{
			events: events,
			onCalled: func(ctx context.Context) error {
				if err := ctx.Err(); err != nil {
					t.Fatalf("shutdown context err = %v, want nil", err)
				}
				if elapsed := time.Since(startedAt); elapsed != 20*time.Millisecond {
					t.Fatalf("shutdown called after %s, want exactly the 20ms propagation delay", elapsed)
				}
				return nil
			},
		}

		if err := drainAndShutdown(ctx, shutdownTestLogger(), 20*time.Millisecond, time.Second, drainer, srv); err != nil {
			t.Fatalf("drainAndShutdown() error = %v, want nil", err)
		}
	})
}

// TestDrainAndShutdownForceClosesOnDeadline covers the case a graceful drain
// cannot finish.
//
// http.Server.Shutdown closes the listeners but then waits indefinitely for
// active connections, so on deadline the handlers it gave up on are still
// running and still holding pooled resources the next shutdown stage has to
// release. Without Close, the process parks in the dependency release until the
// platform SIGKILLs it, discarding the shutdown telemetry.
func TestDrainAndShutdownForceClosesOnDeadline(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}

		srv := &fakeShutdownServer{
			events: events,
			onCalled: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}

		err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, 20*time.Millisecond, drainer, srv)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("drainAndShutdown() error = %v, want context.DeadlineExceeded", err)
		}
		if want := []string{"drain", "shutdown", "close"}; !slices.Equal(events.observed(), want) {
			t.Fatalf("events = %v, want %v", events.observed(), want)
		}
	})
}

// TestDrainAndShutdownDoesNotForceCloseOnCleanDrain keeps the abrupt path scoped
// to the failure that needs it.
func TestDrainAndShutdownDoesNotForceCloseOnCleanDrain(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}
		srv := &fakeShutdownServer{events: events}

		if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, time.Second, drainer, srv); err != nil {
			t.Fatalf("drainAndShutdown() error = %v, want nil", err)
		}
		if want := []string{"drain", "shutdown"}; !slices.Equal(events.observed(), want) {
			t.Fatalf("events = %v, want %v", events.observed(), want)
		}
	})
}

// TestDrainAndShutdownForceClosesEveryServer keeps the metrics listener from
// being left behind when the API listener is what timed out.
func TestDrainAndShutdownForceClosesEveryServer(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}

		hang := func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}
		api := &fakeShutdownServer{events: events, onCalled: hang}
		diagnostics := &fakeShutdownServer{events: events, onCalled: hang}

		if err := drainAndShutdown(
			context.Background(),
			shutdownTestLogger(),
			0,
			20*time.Millisecond,
			drainer,
			api,
			diagnostics,
		); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("drainAndShutdown() error = %v, want context.DeadlineExceeded", err)
		}

		closes := events.count("close")
		if closes != 2 {
			t.Fatalf("close events = %d, want one per server", closes)
		}
	})
}

// TestDrainAndShutdownReportsForceCloseFailure keeps a failing Close from being
// swallowed by the deadline error that triggered it.
func TestDrainAndShutdownReportsForceCloseFailure(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		closeErr := errors.New("listener already gone")
		events := &eventRecorder{}
		drainer := &fakeDrainer{events: events}
		srv := &fakeShutdownServer{
			events:   events,
			closeErr: closeErr,
			onCalled: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}

		err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, 20*time.Millisecond, drainer, srv)
		if !errors.Is(err, closeErr) {
			t.Fatalf("drainAndShutdown() error = %v, want wrapped %v", err, closeErr)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("drainAndShutdown() error = %v, want it to keep the deadline cause", err)
		}
	})
}
