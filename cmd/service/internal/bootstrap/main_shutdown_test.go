package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
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
			remaining := time.Until(deadline)
			if remaining < 29*time.Second || remaining > 31*time.Second {
				t.Fatalf("shutdown deadline remaining = %s, want around 30s", remaining)
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

func TestDrainAndShutdownIgnoresContextCanceledError(t *testing.T) {
	t.Parallel()

	var events []string
	drainer := &fakeDrainer{events: &events}
	srv := &fakeShutdownServer{
		events: &events,
		err:    context.Canceled,
	}

	if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 0, time.Second, drainer, srv); err != nil {
		t.Fatalf("drainAndShutdown() error = %v, want nil", err)
	}
}

func TestDrainAndShutdownWaitsForPropagationDelay(t *testing.T) {
	t.Parallel()

	var events []string
	drainer := &fakeDrainer{events: &events}
	startedAt := time.Now()

	srv := &fakeShutdownServer{
		events: &events,
		onCalled: func(context.Context) error {
			if elapsed := time.Since(startedAt); elapsed < 20*time.Millisecond {
				t.Fatalf("shutdown called too early: %s", elapsed)
			}
			return nil
		},
	}

	if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 20*time.Millisecond, time.Second, drainer, srv); err != nil {
		t.Fatalf("drainAndShutdown() error = %v, want nil", err)
	}
}

func TestDrainAndShutdownCountsPropagationDelayAgainstShutdownTimeout(t *testing.T) {
	t.Parallel()

	var events []string
	drainer := &fakeDrainer{events: &events}

	srv := &fakeShutdownServer{
		events: &events,
		onCalled: func(ctx context.Context) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("shutdown context has no deadline")
			}

			remaining := time.Until(deadline)
			if remaining >= 35*time.Millisecond {
				t.Fatalf("shutdown deadline remaining = %s, want propagation delay to consume part of timeout", remaining)
			}
			if remaining <= 0 {
				t.Fatalf("shutdown deadline remaining = %s, want positive budget for shutdown", remaining)
			}
			return nil
		},
	}

	if err := drainAndShutdown(context.Background(), shutdownTestLogger(), 20*time.Millisecond, 40*time.Millisecond, drainer, srv); err != nil {
		t.Fatalf("drainAndShutdown() error = %v, want nil", err)
	}
}

func TestDrainAndShutdownWaitsForPropagationDelayDespiteCanceledParent(t *testing.T) {
	t.Parallel()

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
			if elapsed := time.Since(startedAt); elapsed < 20*time.Millisecond {
				t.Fatalf("shutdown called too early: %s", elapsed)
			}
			return nil
		},
	}

	if err := drainAndShutdown(ctx, shutdownTestLogger(), 20*time.Millisecond, time.Second, drainer, srv); err != nil {
		t.Fatalf("drainAndShutdown() error = %v, want nil", err)
	}
}
