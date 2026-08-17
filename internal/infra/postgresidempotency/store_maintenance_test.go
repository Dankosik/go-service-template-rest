package postgresidempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestMaintenanceScheduleAndBounds(t *testing.T) {
	t.Parallel()
	store := maintenanceUnitStore()
	now := time.Now()
	for _, tc := range []struct {
		name     string
		snapshot *maintenanceSnapshot
		wantErr  error
	}{
		{name: "unobserved", snapshot: &maintenanceSnapshot{}, wantErr: ErrUnavailable},
		{name: "writer safe", snapshot: &maintenanceSnapshot{observedAt: now, writer: true}},
		{name: "non writer", snapshot: &maintenanceSnapshot{observedAt: now}, wantErr: ErrUnavailable},
		{name: "stale", snapshot: &maintenanceSnapshot{observedAt: now.Add(-2 * time.Minute), writer: true}, wantErr: ErrUnavailable},
		{name: "headroom boundary", snapshot: &maintenanceSnapshot{observedAt: now, writer: true, relationBytes: 900}, wantErr: ErrUnavailable},
		{name: "terminal epoch", snapshot: &maintenanceSnapshot{observedAt: now, writer: true, terminal: ErrEpochLost}, wantErr: ErrEpochLost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := store.snapshotError(tc.snapshot)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("snapshotError() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestMaintenanceFailureAndCapacityClosure(t *testing.T) {
	t.Parallel()
	store := maintenanceUnitStore()
	if store.allowsFirstExecution() {
		t.Fatal("unobserved Store admitted first execution")
	}
	store.safety.Store(&maintenanceSnapshot{observedAt: time.Now(), writer: true, relationBytes: 899})
	if !store.allowsFirstExecution() {
		t.Fatal("safe writer observation rejected first execution")
	}
	store.safety.Store(&maintenanceSnapshot{observedAt: time.Now(), writer: true, relationBytes: 900})
	if store.allowsFirstExecution() {
		t.Fatal("headroom boundary admitted first execution")
	}
	store.ObserveTerminal(context.Background(), httpidempotency.Decision{}, ErrIntegrityConflict)
	if !errors.Is(store.snapshotError(store.safety.Load()), ErrIntegrityConflict) {
		t.Fatalf("terminal snapshot = %v, want integrity conflict", store.snapshotError(store.safety.Load()))
	}
	if got := <-store.TerminalErrors(); !errors.Is(got, ErrIntegrityConflict) {
		t.Fatalf("first terminal wakeup = %v, want integrity conflict", got)
	}
	store.markTerminal(ErrEpochLost)
	if !errors.Is(store.snapshotError(store.safety.Load()), ErrIntegrityConflict) {
		t.Fatalf("later terminal overwrote the first integrity stop: %v", store.snapshotError(store.safety.Load()))
	}
	if err := store.publishSnapshot(&maintenanceSnapshot{observedAt: time.Now(), writer: true}); !errors.Is(err, ErrIntegrityConflict) {
		t.Fatalf("safe observation replaced terminal snapshot: %v", err)
	}
	select {
	case got := <-store.TerminalErrors():
		t.Fatalf("later terminal wakeup = %v, want none", got)
	default:
	}
}

func TestMaintenanceFailureStateFailsClosedBeforeAnotherCycle(t *testing.T) {
	t.Parallel()
	store := maintenanceUnitStore()
	if err := store.Maintain(t.Context()); !errors.Is(err, ErrConfig) {
		t.Fatalf("Maintain(unbound Store) error = %v, want ErrConfig", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := store.Check(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Check(cancelled) error = %v, want context.Canceled", err)
	}
	if err := store.finishMaintenanceError(t.Context(), nil, ErrUnavailable); !errors.Is(err, ErrUnavailable) || !store.maintenanceFailed {
		t.Fatalf("finishMaintenanceError(unavailable) = %v, failed = %t", err, store.maintenanceFailed)
	}
	if err := store.finishMaintenanceError(t.Context(), nil, ErrEpochLost); !errors.Is(err, ErrEpochLost) {
		t.Fatalf("finishMaintenanceError(epoch) = %v, want ErrEpochLost", err)
	}
	if snapshot := store.safety.Load(); snapshot == nil || !errors.Is(snapshot.terminal, ErrEpochLost) {
		t.Fatalf("terminal maintenance snapshot = %#v, want ErrEpochLost", snapshot)
	}
	if got := <-store.TerminalErrors(); !errors.Is(got, ErrEpochLost) {
		t.Fatalf("terminal maintenance wakeup = %v, want ErrEpochLost", got)
	}
	if err := store.publishSnapshot(&maintenanceSnapshot{observedAt: time.Now(), writer: true}); !errors.Is(err, ErrEpochLost) {
		t.Fatalf("publishSnapshot() replaced terminal error: %v", err)
	}
}

func maintenanceUnitStore() *Store {
	store := &Store{options: StoreOptions{
		OwnerRecoveryDelay:     time.Second,
		CleanupBatchSize:       10,
		MaxMaintenanceLag:      time.Minute,
		MaxRelationBytes:       1000,
		AdmissionHeadroomBytes: 100,
	}}
	store.safety.Store(&maintenanceSnapshot{})
	store.terminalErrors = make(chan error, 1)
	return store
}
