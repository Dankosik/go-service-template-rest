package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"
)

func TestStartupAdmissionControllerCheckReady(t *testing.T) {
	t.Parallel()

	admission := new(startupAdmissionController)

	err := admission.CheckReady(context.Background())
	if !errors.Is(err, errStartupAdmissionPending) {
		t.Fatalf("CheckReady() error = %v, want %v", err, errStartupAdmissionPending)
	}

	admission.MarkReady()
	if err := admission.CheckReady(context.Background()); err != nil {
		t.Fatalf("CheckReady() after MarkReady error = %v, want nil", err)
	}
}

func TestStartupAdmissionStartsReadinessWatcherAfterPublishingReady(t *testing.T) {
	t.Parallel()

	admission := new(startupAdmissionController)
	checkReturned := false
	watcherStarts := 0
	args := serveRuntimeArgs{
		log:       slog.New(slog.DiscardHandler),
		admission: admission,
		onReady: func() {
			if !checkReturned || !admission.Ready() {
				t.Fatal("readiness watcher started before successful admission was published")
			}
			watcherStarts++
		},
	}
	admissionErrCh := startStartupAdmission(t.Context(), func(context.Context) error {
		checkReturned = true
		return nil
	}, time.Second)
	ready, err := waitForStartupAdmission(
		context.Background(),
		t.Context(),
		args,
		admissionErrCh,
		make(chan serverResult),
	)
	if err != nil || !ready || watcherStarts != 1 {
		t.Fatalf("admission = ready:%t watcher starts:%d err:%v", ready, watcherStarts, err)
	}
}

func TestStartStartupAdmissionRejectsCanceledReadinessContextAfterSuccessfulCheck(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		startupCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		resultCh := startStartupAdmission(startupCtx, func(ctx context.Context) error {
			cancel()
			<-ctx.Done()
			return nil
		}, time.Second)

		err := <-resultCh
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("startStartupAdmission() error = %v, want wrapped %v", err, context.Canceled)
		}
	})
}
