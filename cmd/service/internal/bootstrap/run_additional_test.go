package bootstrap

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOverlayPathsFlagSetAndString(t *testing.T) {
	t.Parallel()

	var f overlayPathsFlag
	if err := f.Set("  a.yaml  "); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}
	if err := f.Set("b.yaml"); err != nil {
		t.Fatalf("Set() second error = %v, want nil", err)
	}
	if got := f.String(); got != "a.yaml,b.yaml" {
		t.Fatalf("String() = %q, want %q", got, "a.yaml,b.yaml")
	}
	if err := f.Set("   "); err == nil {
		t.Fatal("Set(empty) error = nil, want non-nil")
	}
}

func TestParseLoadOptions(t *testing.T) {
	t.Parallel()

	t.Run("parses flags", func(t *testing.T) {
		t.Parallel()

		opts, err := parseLoadOptions([]string{
			"--config", "/tmp/base.yaml",
			"--config-overlay", "/tmp/o1.yaml",
			"--config-overlay", "/tmp/o2.yaml",
			"--config-strict",
		})
		if err != nil {
			t.Fatalf("parseLoadOptions() error = %v, want nil", err)
		}
		if opts.ConfigPath != "/tmp/base.yaml" {
			t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, "/tmp/base.yaml")
		}
		if !opts.Strict {
			t.Fatal("Strict = false, want true")
		}
		if len(opts.ConfigOverlays) != 2 {
			t.Fatalf("ConfigOverlays len = %d, want 2", len(opts.ConfigOverlays))
		}
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--unknown-flag"})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
	})

	t.Run("fails on positional arguments", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--config", "/tmp/base.yaml", "serve"})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
		if !strings.Contains(err.Error(), "serve") {
			t.Fatalf("parseLoadOptions() err = %v, want unexpected positional argument detail", err)
		}
	})
}

func TestRunReturnsParseErrorForInvalidFlags(t *testing.T) {
	t.Parallel()

	err := Run([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse flags") {
		t.Fatalf("Run() err = %v, want parse flags context", err)
	}
}

func TestReleaseSignalNotificationOnDoneReleasesOnceAfterCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var stopCalls atomic.Int32
	stopCalled := make(chan struct{})

	release := releaseSignalNotificationOnDone(ctx, func() {
		if stopCalls.Add(1) == 1 {
			close(stopCalled)
		}
	})
	defer release()

	cancel()

	select {
	case <-stopCalled:
	case <-time.After(time.Second):
		t.Fatal("stop callback was not called after context cancellation")
	}

	release()
	release()
	if got := stopCalls.Load(); got != 1 {
		t.Fatalf("stop callback calls = %d, want 1", got)
	}
}
