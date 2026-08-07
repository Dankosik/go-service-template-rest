package postgresoutbox

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
)

// ClaimedBatch is one lease. Every event in it shares Token, and every
// finalization compares against that token, so the batch is fenced as a unit
// against a relay that overran its own lease.
type ClaimedBatch struct {
	Token  string
	Events []ClaimedEvent
}

// ClaimedEvent is one leased event. Only the first two fields drive delivery:
// Relay.classify reads CycleAttemptCount against RelayConfig.MaxAttempts and
// nothing else here. The last two are the claim's report to whoever is watching
// it, which is why no relay code reads them back.
type ClaimedEvent struct {
	Event Event
	// CycleAttemptCount counts attempts since the last redrive reset it, and is
	// what the attempt cap is measured against.
	CycleAttemptCount int
	// TotalAttemptCount counts every attempt this event has ever had, across
	// redrives. It is the one attempt figure a reset cannot hide.
	TotalAttemptCount int64
	// Recovered means this claim picked the event up from an expired lease
	// rather than from ordinary eligibility — a crashed or overrun relay's work
	// coming back. Claim already counts and logs it; the field is what lets a
	// caller assert the same transition without reading telemetry.
	Recovered bool
}

// Claim leases up to batchSize eligible events under one fresh token. An empty
// batch means no eligible work, which is an ordinary outcome rather than an
// error.
func (s *Store) Claim(ctx context.Context, leaseDuration time.Duration, batchSize int) (batch ClaimedBatch, err error) {
	started := time.Now()
	defer func() { s.recordClaim(ctx, batch, started, err) }()
	if !s.valid() {
		return ClaimedBatch{}, errStoreRequired()
	}
	if leaseDuration <= 0 {
		return ClaimedBatch{}, fmt.Errorf("%w: lease duration must be positive", ErrConfig)
	}
	if batchSize < 1 || batchSize > math.MaxInt32 {
		return ClaimedBatch{}, fmt.Errorf("%w: claim batch size must be positive", ErrConfig)
	}

	token := NewID()
	rows, err := s.queries.ClaimOutboxEvents(ctx, sqlcgen.ClaimOutboxEventsParams{
		LeaseToken:        &token,
		LeaseMilliseconds: durationMilliseconds(leaseDuration),
		BatchSize:         int32(batchSize), // #nosec G115 -- range checked above.
	})
	if err != nil {
		return ClaimedBatch{}, fmt.Errorf("claim outbox events: %w", err)
	}
	if len(rows) == 0 {
		return ClaimedBatch{}, nil
	}
	batch = ClaimedBatch{Token: token, Events: make([]ClaimedEvent, 0, len(rows))}
	for _, row := range rows {
		batch.Events = append(batch.Events, ClaimedEvent{
			Event:             eventFromClaimRow(row),
			CycleAttemptCount: int(row.CycleAttemptCount),
			TotalAttemptCount: row.TotalAttemptCount,
			Recovered:         row.RecoveryDue,
		})
		if row.RecoveryDue {
			// One recovered event, counted inside the claim that found it. The
			// claim already owns the only duration either of them has.
			s.telemetry.CountOperation(ctx, "recovery", outcomeSuccess, classNone)
			s.telemetry.LogRecovery(ctx, row.ID, int(row.CycleAttemptCount))
		}
	}
	return batch, nil
}
