package postgresoutbox

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const commitReceiptFingerprintVersion int16 = 1

// CommitOutcome is the only result that decides whether an operation whose
// commit response was lost may be retried. Its zero value is deliberately
// inconclusive.
type CommitOutcome uint8

const (
	CommitStillUnknown CommitOutcome = iota
	CommitApplied
	CommitNotApplied
)

// ReconcileCommit reads the immutable receipt on the configured writer pool.
// Only an absence observed by a writable primary authorizes the caller to retry
// its mutation with this same event.
func (s *Store) ReconcileCommit(ctx context.Context, event Event) (outcome CommitOutcome, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "reconcile_commit", started, err) }()
	if !s.valid() {
		return CommitStillUnknown, errStoreRequired()
	}

	fingerprint, err := commitReceiptFingerprint(event)
	if err != nil {
		return CommitStillUnknown, err
	}
	receipt, err := s.queries.ReconcileOutboxCommit(ctx, event.ID)
	if err != nil {
		return CommitStillUnknown, fmt.Errorf("reconcile outbox commit: %w", err)
	}
	if !receipt.ReceiptExists {
		if receipt.WriterPrimary {
			return CommitNotApplied, nil
		}
		return CommitStillUnknown, errors.New("reconcile outbox commit: PostgreSQL endpoint is not a writable primary")
	}
	if receipt.FingerprintVersion != commitReceiptFingerprintVersion {
		return CommitStillUnknown, fmt.Errorf(
			"reconcile outbox commit: unsupported fingerprint version %d",
			receipt.FingerprintVersion,
		)
	}
	if !bytes.Equal(receipt.EnvelopeFingerprint, fingerprint[:]) {
		return CommitStillUnknown, fmt.Errorf("%w: event %q has different immutable evidence", ErrReceiptConflict, event.ID)
	}
	return CommitApplied, nil
}

// commitReceiptFingerprint is shared by append and reconciliation so the
// encoding cannot drift between the evidence producer and its reader.
func commitReceiptFingerprint(event Event) ([sha256.Size]byte, error) {
	event = event.withDefaults()
	if err := event.Validate(); err != nil {
		return [sha256.Size]byte{}, err
	}

	hash := sha256.New()
	_, _ = hash.Write([]byte("postgresoutbox-receipt-v1\x00"))
	var length [8]byte
	write := func(value []byte) {
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(value)
	}
	write([]byte(event.ID))
	write([]byte(event.Type))
	write([]byte(event.Source))
	write([]byte(event.Destination))
	write([]byte(event.Schema))
	var integer [8]byte
	binary.BigEndian.PutUint64(integer[:], uint64(event.OccurredAt.UTC().UnixMicro()))
	write(integer[:])
	write(event.Payload)
	write(event.Metadata)
	write([]byte(event.OrderingKey))
	binary.BigEndian.PutUint64(integer[:], uint64(event.OrderingSequence))
	write(integer[:])

	var fingerprint [sha256.Size]byte
	hash.Sum(fingerprint[:0])
	return fingerprint, nil
}
