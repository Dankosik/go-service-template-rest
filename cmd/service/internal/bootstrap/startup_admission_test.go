package bootstrap

import (
	"context"
	"errors"
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

func TestStartStartupAdmissionRejectsCanceledReadinessContextAfterSuccessfulCheck(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		bootstrapCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		resultCh := startStartupAdmission(bootstrapCtx, func(ctx context.Context) error {
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
