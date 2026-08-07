package postgresoutbox

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
)

// StateObservation is one sample of everything the relay reports about the
// outbox: how much work sits in each delivery state, how old the oldest of each
// is, and what the pack costs on disk. Relay.runDueMaintenance takes it on a timer
// and Telemetry turns it into gauges; nothing here is read back into a
// delivery decision.
type StateObservation struct {
	EligibleCount             int64
	EligibleOldestAt          time.Time
	InProgressCount           int64
	InProgressOldestAt        time.Time
	RetryWaitCount            int64
	RetryWaitOldestAt         time.Time
	RecoveryDueCount          int64
	RecoveryDueOldestAt       time.Time
	OrderingBlockedCount      int64
	OrderingBlockedOldestAt   time.Time
	PoisonCount               int64
	PoisonOldestAt            time.Time
	PublishedRetainedEstimate int64
	PublishedRetainedOldestAt time.Time
	OrderingHeadCount         int64
	EventsBytes               int64
	EventsIndexBytes          int64
	OrderingHeadsBytes        int64
	OrderingHeadsIndexBytes   int64
	RedrivesBytes             int64
	RedrivesIndexBytes        int64
}

func (s *Store) CleanupPublished(ctx context.Context, retention time.Duration, batchSize int) (deleted int, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "cleanup", started, err) }()
	if !s.valid() {
		return 0, errStoreRequired()
	}
	if retention <= 0 || batchSize < 1 || batchSize > math.MaxInt32 {
		return 0, fmt.Errorf("%w: cleanup retention and batch size are invalid", ErrConfig)
	}
	rows, err := s.queries.CleanupPublishedOutboxEvents(ctx, sqlcgen.CleanupPublishedOutboxEventsParams{
		RetentionMilliseconds: durationMilliseconds(retention),
		BatchSize:             int32(batchSize), // #nosec G115 -- range checked above.
	})
	if err != nil {
		return 0, fmt.Errorf("cleanup published outbox events: %w", err)
	}
	return int(rows), nil
}

func (s *Store) Observe(ctx context.Context) (observation StateObservation, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "observe", started, err) }()
	if !s.valid() {
		return StateObservation{}, errStoreRequired()
	}
	state, err := s.queries.ObserveOutbox(ctx)
	if err != nil {
		return StateObservation{}, fmt.Errorf("observe outbox state: %w", err)
	}
	return StateObservation{
		EligibleCount:             state.EligibleCount,
		EligibleOldestAt:          timeFromUnixSeconds(state.EligibleOldestUnix),
		InProgressCount:           state.InProgressCount,
		InProgressOldestAt:        timeFromUnixSeconds(state.InProgressOldestUnix),
		RetryWaitCount:            state.RetryWaitCount,
		RetryWaitOldestAt:         timeFromUnixSeconds(state.RetryWaitOldestUnix),
		RecoveryDueCount:          state.RecoveryDueCount,
		RecoveryDueOldestAt:       timeFromUnixSeconds(state.RecoveryDueOldestUnix),
		OrderingBlockedCount:      state.OrderingBlockedCount,
		OrderingBlockedOldestAt:   timeFromUnixSeconds(state.OrderingBlockedOldestUnix),
		PoisonCount:               state.PoisonCount,
		PoisonOldestAt:            timeFromUnixSeconds(state.PoisonOldestUnix),
		PublishedRetainedEstimate: state.PublishedRetainedEstimate,
		PublishedRetainedOldestAt: timeFromUnixSeconds(state.PublishedRetainedOldestUnix),
		OrderingHeadCount:         state.OrderingHeadCount,
		EventsBytes:               state.EventsBytes,
		EventsIndexBytes:          state.EventsIndexBytes,
		OrderingHeadsBytes:        state.OrderingHeadsBytes,
		OrderingHeadsIndexBytes:   state.OrderingHeadsIndexBytes,
		RedrivesBytes:             state.RedrivesBytes,
		RedrivesIndexBytes:        state.RedrivesIndexBytes,
	}, nil
}
