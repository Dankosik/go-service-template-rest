package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

type fakeDrainer struct {
	events  *[]string
	started bool
}

func (f *fakeDrainer) StartDrain() {
	f.started = true
	*f.events = append(*f.events, "drain")
}

type fakeShutdownServer struct {
	events   *[]string
	err      error
	onCalled func(context.Context) error
}

func (f *fakeShutdownServer) Shutdown(ctx context.Context) error {
	*f.events = append(*f.events, "shutdown")
	if f.onCalled != nil {
		if err := f.onCalled(ctx); err != nil {
			return err
		}
	}
	return f.err
}

func shutdownTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestDrainAndShutdownOrdersDrainBeforeShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var events []string
		drainer := &fakeDrainer{events: &events}

		srv := &fakeShutdownServer{
			events: &events,
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

		if got := strings.Join(events, ","); got != "drain,shutdown" {
			t.Fatalf("event order = %q, want %q", got, "drain,shutdown")
		}
	})
}

func TestDrainAndShutdownIgnoresParentCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var events []string
	drainer := &fakeDrainer{events: &events}
	srv := &fakeShutdownServer{
		events: &events,
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

	var events []string
	drainer := &fakeDrainer{events: &events}
	srv := &fakeShutdownServer{
		events: &events,
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

	var events []string
	drainer := &fakeDrainer{events: &events}
	srv := &fakeShutdownServer{
		events: &events,
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
		var events []string
		release := make(chan struct{})
		srv := &fakeShutdownServer{
			events: &events,
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
			&fakeDrainer{events: &events},
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
		var events []string
		drainer := &fakeDrainer{events: &events}
		startedAt := time.Now()

		srv := &fakeShutdownServer{
			events: &events,
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
		var events []string
		drainer := &fakeDrainer{events: &events}

		srv := &fakeShutdownServer{
			events: &events,
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

		var events []string
		drainer := &fakeDrainer{events: &events}
		startedAt := time.Now()

		srv := &fakeShutdownServer{
			events: &events,
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
