package postgresidempotency

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestStoreRejectsMissingInputs(t *testing.T) {
	t.Parallel()

	if _, err := NewStore(nil, time.Hour); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewStore(nil) error = %v, want ErrConfig", err)
	}
	store := &Store{}
	if _, _, err := store.execute(t.Context(), httpidempotency.Request{}, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("execute(invalid) error = %v, want ErrConfig", err)
	}
	if _, err := NewExecutor[struct{}, struct{}](nil, nil, httpidempotency.Codec[struct{}]{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewExecutor(invalid) error = %v, want ErrConfig", err)
	}
}

func TestStoreMaintenanceReportsCleanupFailureAndStopsWithContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		var output bytes.Buffer
		done := make(chan error, 1)
		go func() {
			done <- new(Store).Maintain(ctx, slog.New(slog.NewTextHandler(&output, nil)))
		}()

		time.Sleep(maintenanceInterval)
		synctest.Wait()
		if !strings.Contains(output.String(), "http idempotency cleanup failed") {
			t.Fatalf("maintenance log = %q, want cleanup failure", output.String())
		}

		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("Maintain() error = %v, want context cancellation", err)
		}
	})
}
