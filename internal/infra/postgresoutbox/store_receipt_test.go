package postgresoutbox

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestCommitReceipt(t *testing.T) {
	event := Event{
		ID:               "evt-1",
		Type:             "order.created",
		Source:           "orders",
		Destination:      "orders.events",
		Schema:           "v1",
		OccurredAt:       time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC),
		Payload:          []byte(`{"id":"order-1"}`),
		OrderingKey:      "order-1",
		OrderingSequence: 7,
	}
	fingerprint, err := commitReceiptFingerprint(event)
	if err != nil {
		t.Fatalf("commitReceiptFingerprint() error = %v", err)
	}
	want, err := hex.DecodeString("e5ab0fe21fc3ae1c8f28d7ad5603bacebc8f202a85b0ce4c54424258b4546d9a")
	if err != nil {
		t.Fatalf("decode golden fingerprint: %v", err)
	}
	if !bytes.Equal(fingerprint[:], want) {
		t.Fatalf("commitReceiptFingerprint() = %x, want %x", fingerprint, want)
	}

	explicitMetadata := event
	explicitMetadata.Metadata = []byte(`{}`)
	explicitFingerprint, err := commitReceiptFingerprint(explicitMetadata)
	if err != nil {
		t.Fatalf("commitReceiptFingerprint(explicit metadata) error = %v", err)
	}
	if fingerprint != explicitFingerprint {
		t.Fatal("absent metadata and explicit empty metadata produced different receipts")
	}

	t.Run("outcomes", func(t *testing.T) {
		tests := []struct {
			name       string
			row        pgx.Row
			want       CommitOutcome
			wantErr    error
			wantAnyErr bool
		}{
			{name: "applied", row: commitReceiptRow{exists: true, version: 1, fingerprint: fingerprint[:], writerPrimary: true}, want: CommitApplied},
			{name: "applied read only", row: commitReceiptRow{exists: true, version: 1, fingerprint: fingerprint[:]}, want: CommitApplied},
			{name: "not applied", row: commitReceiptRow{writerPrimary: true}, want: CommitNotApplied},
			{name: "absence without authority", row: commitReceiptRow{}, want: CommitStillUnknown, wantAnyErr: true},
			{name: "unsupported version", row: commitReceiptRow{exists: true, version: 2, fingerprint: fingerprint[:], writerPrimary: true}, want: CommitStillUnknown, wantAnyErr: true},
			{name: "conflict", row: commitReceiptRow{exists: true, version: 1, fingerprint: bytes.Repeat([]byte{1}, 32), writerPrimary: true}, want: CommitStillUnknown, wantErr: ErrReceiptConflict},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				driver := &commitReceiptDriver{row: test.row}
				outcome, reconcileErr := stubbedStore(driver).ReconcileCommit(t.Context(), event)
				if outcome != test.want {
					t.Fatalf("ReconcileCommit() outcome = %v, want %v", outcome, test.want)
				}
				if test.wantAnyErr && reconcileErr == nil {
					t.Fatal("ReconcileCommit() error = nil")
				}
				if test.wantErr != nil && !errors.Is(reconcileErr, test.wantErr) {
					t.Fatalf("ReconcileCommit() error = %v, want %v", reconcileErr, test.wantErr)
				}
				if !test.wantAnyErr && test.wantErr == nil && reconcileErr != nil {
					t.Fatalf("ReconcileCommit() error = %v", reconcileErr)
				}
				if driver.calls != 1 || len(driver.arguments) != 1 || driver.arguments[0] != event.ID {
					t.Fatalf("ReconcileCommit() queries = %d args = %v", driver.calls, driver.arguments)
				}
			})
		}
	})

	t.Run("invalid event stays off wire", func(t *testing.T) {
		driver := &commitReceiptDriver{row: commitReceiptRow{}}
		invalid := event
		invalid.ID = ""
		outcome, reconcileErr := stubbedStore(driver).ReconcileCommit(t.Context(), invalid)
		if outcome != CommitStillUnknown || !errors.Is(reconcileErr, ErrInvalidEvent) {
			t.Fatalf("ReconcileCommit(invalid) = %v, %v", outcome, reconcileErr)
		}
		if driver.calls != 0 {
			t.Fatalf("ReconcileCommit(invalid) sent %d queries, want 0", driver.calls)
		}
	})

	t.Run("read failure stays unknown", func(t *testing.T) {
		databaseErr := errors.New("database unavailable")
		driver := &commitReceiptDriver{row: rowStub{err: databaseErr}}
		outcome, reconcileErr := stubbedStore(driver).ReconcileCommit(t.Context(), event)
		if outcome != CommitStillUnknown || !errors.Is(reconcileErr, databaseErr) {
			t.Fatalf("ReconcileCommit(read failure) = %v, %v", outcome, reconcileErr)
		}
	})
}

type commitReceiptDriver struct {
	databaseStub

	row       pgx.Row
	calls     int
	arguments []any
}

//nolint:ireturn // The pgx DBTX test double must return pgx's interface.
func (driver *commitReceiptDriver) QueryRow(_ context.Context, _ string, arguments ...any) pgx.Row {
	driver.calls++
	driver.arguments = arguments
	return driver.row
}

type commitReceiptRow struct {
	exists        bool
	version       int16
	fingerprint   []byte
	writerPrimary bool
}

func (row commitReceiptRow) Scan(destinations ...any) error {
	if len(destinations) != 4 {
		return errors.New("unexpected commit receipt scan destinations")
	}
	exists, existsOK := destinations[0].(*bool)
	version, versionOK := destinations[1].(*int16)
	fingerprint, fingerprintOK := destinations[2].(*[]byte)
	writerPrimary, writerPrimaryOK := destinations[3].(*bool)
	if !existsOK || !versionOK || !fingerprintOK || !writerPrimaryOK {
		return errors.New("unexpected commit receipt scan destination types")
	}
	*exists = row.exists
	*version = row.version
	*fingerprint = row.fingerprint
	*writerPrimary = row.writerPrimary
	return nil
}
