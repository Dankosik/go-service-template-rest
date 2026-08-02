package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrConfig           = errors.New("outbox store config")
	ErrNoWork           = errors.New("outbox has no eligible work")
	ErrNotFound         = errors.New("outbox event not found")
	ErrLeaseLost        = errors.New("outbox lease lost")
	ErrOrderingSequence = errors.New("outbox ordering sequence rejected")
	ErrRedriveRejected  = errors.New("outbox redrive rejected")
	ErrRedriveConflict  = errors.New("outbox redrive audit conflict")
)

type Store struct {
	pool      *postgres.Pool
	queries   *sqlcgen.Queries
	telemetry *Telemetry
}

type ClaimedEvent struct {
	Event             Event
	Token             string
	CycleAttemptCount int
	TotalAttemptCount int64
	LeaseExpiresAt    time.Time
	Recovered         bool
}

type Record struct {
	Event             Event
	CreatedAt         time.Time
	AvailableAt       time.Time
	CycleAttemptCount int
	TotalAttemptCount int64
	LastAttemptAt     time.Time
	LeaseToken        string
	LeaseExpiresAt    time.Time
	PublishedAt       time.Time
	PoisonedAt        time.Time
	LastErrorClass    string
	RedriveCount      int
	LastRedriveID     string
	LastRedrivenAt    time.Time
}

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
	PublishedRetainedCount    int64
	PublishedRetainedOldestAt time.Time
	OrderingHeadCount         int64
	EventsBytes               int64
	EventsIndexBytes          int64
	OrderingHeadsBytes        int64
	OrderingHeadsIndexBytes   int64
}

func NewStore(pool *postgres.Pool, telemetry *Telemetry) (*Store, error) {
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	return &Store{pool: pool, queries: sqlcgen.New(pool.PGX()), telemetry: telemetry}, nil
}

// Append participates in the transaction owned by the feature caller. It never
// begins or commits a transaction itself.
func (s *Store) Append(ctx context.Context, tx pgx.Tx, event Event) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "append", started, err) }()
	if s == nil || s.pool == nil || tx == nil {
		return fmt.Errorf("%w: store and transaction are required", ErrConfig)
	}
	event = event.withDefaults()
	if err := event.Validate(); err != nil {
		return err
	}

	queries := sqlcgen.New(tx)
	var orderingKey *string
	var orderingSequence *int64
	if event.OrderingKey != "" {
		if _, err := queries.AdvanceOutboxOrderingHead(ctx, sqlcgen.AdvanceOutboxOrderingHeadParams{
			OrderingKey:      event.OrderingKey,
			OrderingSequence: event.OrderingSequence,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: key %q sequence %d is not above the retained high-water mark", ErrOrderingSequence, event.OrderingKey, event.OrderingSequence)
			}
			return fmt.Errorf("advance outbox ordering head: %w", err)
		}
		orderingKey = &event.OrderingKey
		orderingSequence = &event.OrderingSequence
	}

	if err := queries.InsertOutboxEvent(ctx, sqlcgen.InsertOutboxEventParams{
		ID:               event.ID,
		EventType:        event.Type,
		Source:           event.Source,
		Destination:      event.Destination,
		SchemaName:       event.Schema,
		OccurredAt:       timestamptz(event.OccurredAt),
		Payload:          event.Payload,
		Metadata:         event.Metadata,
		OrderingKey:      orderingKey,
		OrderingSequence: orderingSequence,
	}); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func (s *Store) Claim(ctx context.Context, leaseDuration time.Duration) (claimed ClaimedEvent, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "claim", started, err) }()
	if s == nil || s.queries == nil {
		return ClaimedEvent{}, fmt.Errorf("%w: store is required", ErrConfig)
	}
	if leaseDuration <= 0 {
		return ClaimedEvent{}, fmt.Errorf("%w: lease duration must be positive", ErrConfig)
	}

	token := NewID()
	row, err := s.queries.ClaimOutboxEvent(ctx, sqlcgen.ClaimOutboxEventParams{
		LeaseToken:        &token,
		LeaseMilliseconds: durationMilliseconds(leaseDuration),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClaimedEvent{}, ErrNoWork
		}
		return ClaimedEvent{}, fmt.Errorf("claim outbox event: %w", err)
	}
	claimed = ClaimedEvent{
		Event:             eventFromClaimRow(row),
		Token:             token,
		CycleAttemptCount: int(row.CycleAttemptCount),
		TotalAttemptCount: row.TotalAttemptCount,
		LeaseExpiresAt:    timeValue(row.LeaseExpiresAt),
		Recovered:         row.RecoveryDue,
	}
	if row.RecoveryDue && s.telemetry != nil {
		s.telemetry.RecordOperation(ctx, "recovery", "success", "none", time.Since(started))
		s.telemetry.LogRecovery(ctx, row.ID, int(row.CycleAttemptCount))
	}
	return claimed, nil
}

func (s *Store) MarkPublished(ctx context.Context, id, token string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	if err := validateProgressIdentity(id, token); err != nil {
		return err
	}
	rows, err := s.queries.MarkOutboxPublished(ctx, sqlcgen.MarkOutboxPublishedParams{ID: id, LeaseToken: &token})
	return progressResult("mark outbox published", rows, err)
}

func (s *Store) ScheduleRetry(ctx context.Context, id, token, errorClass string, delay time.Duration) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "schedule_retry", started, err) }()
	if err := validateProgressIdentity(id, token); err != nil {
		return err
	}
	if delay < 0 {
		return fmt.Errorf("%w: retry delay cannot be negative", ErrConfig)
	}
	if err := validateErrorClass(errorClass); err != nil {
		return err
	}
	rows, err := s.queries.ScheduleOutboxRetry(ctx, sqlcgen.ScheduleOutboxRetryParams{
		DelayMilliseconds: durationMilliseconds(delay),
		ErrorClass:        &errorClass,
		ID:                id,
		LeaseToken:        &token,
	})
	return progressResult("schedule outbox retry", rows, err)
}

func (s *Store) MarkPoisoned(ctx context.Context, id, token, errorClass string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "poison", started, err) }()
	if err := validateProgressIdentity(id, token); err != nil {
		return err
	}
	if err := validateErrorClass(errorClass); err != nil {
		return err
	}
	rows, err := s.queries.MarkOutboxPoisoned(ctx, sqlcgen.MarkOutboxPoisonedParams{
		ErrorClass: &errorClass,
		ID:         id,
		LeaseToken: &token,
	})
	return progressResult("mark outbox poisoned", rows, err)
}

func (s *Store) Get(ctx context.Context, id string) (Record, error) {
	if err := validateText("id", id, maxTextBytes); err != nil {
		return Record{}, err
	}
	row, err := s.queries.GetOutboxEvent(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("get outbox event: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) Redrive(ctx context.Context, id, auditID string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "redrive", started, err) }()
	if err := validateText("id", id, maxTextBytes); err != nil {
		return err
	}
	if err := validateText("audit_id", auditID, maxTextBytes); err != nil {
		return err
	}

	err = s.pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		row, err := queries.LockOutboxEventForRedrive(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("lock outbox event for redrive: %w", err)
		}

		earlierEventID, err := queries.FindOutboxRedrive(ctx, auditID)
		switch {
		case err == nil && earlierEventID == id:
			return nil
		case err == nil:
			return fmt.Errorf("%w: audit id belongs to another event", ErrRedriveConflict)
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("find outbox redrive: %w", err)
		}

		if !row.PoisonedAt.Valid || row.PublishedAt.Valid {
			return ErrRedriveRejected
		}
		if row.RedriveCount == math.MaxInt32 {
			return fmt.Errorf("%w: redrive count exhausted", ErrRedriveRejected)
		}
		cycle := row.RedriveCount + 1
		if err := queries.InsertOutboxRedrive(ctx, sqlcgen.InsertOutboxRedriveParams{
			AuditID:     auditID,
			EventID:     id,
			CycleNumber: cycle,
		}); err != nil {
			return fmt.Errorf("insert outbox redrive: %w", err)
		}
		rows, err := queries.RedriveOutboxEvent(ctx, sqlcgen.RedriveOutboxEventParams{AuditID: &auditID, ID: id})
		if err != nil {
			return fmt.Errorf("redrive outbox event: %w", err)
		}
		if rows != 1 {
			return ErrRedriveRejected
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("redrive outbox event: %w", err)
	}
	return nil
}

func (s *Store) CleanupPublished(ctx context.Context, retention time.Duration, batchSize int) (deleted int, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "cleanup", started, err) }()
	if retention <= 0 || batchSize < 1 || batchSize > math.MaxInt32 {
		return 0, fmt.Errorf("%w: cleanup retention and batch size are invalid", ErrConfig)
	}
	ids, err := s.queries.CleanupPublishedOutboxEvents(ctx, sqlcgen.CleanupPublishedOutboxEventsParams{
		RetentionMilliseconds: durationMilliseconds(retention),
		BatchSize:             int32(batchSize), // #nosec G115 -- range checked above.
	})
	if err != nil {
		return 0, fmt.Errorf("cleanup published outbox events: %w", err)
	}
	return len(ids), nil
}

func (s *Store) Observe(ctx context.Context) (observation StateObservation, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "observe", started, err) }()
	err = s.pool.InTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		state, err := queries.ObserveOutbox(ctx)
		if err != nil {
			return fmt.Errorf("observe outbox state: %w", err)
		}
		storage, err := queries.ObserveOutboxStorage(ctx)
		if err != nil {
			return fmt.Errorf("observe outbox storage: %w", err)
		}
		observation = StateObservation{
			EligibleCount:             state.EligibleCount,
			EligibleOldestAt:          unixTime(state.EligibleOldestUnix),
			InProgressCount:           state.InProgressCount,
			InProgressOldestAt:        unixTime(state.InProgressOldestUnix),
			RetryWaitCount:            state.RetryWaitCount,
			RetryWaitOldestAt:         unixTime(state.RetryWaitOldestUnix),
			RecoveryDueCount:          state.RecoveryDueCount,
			RecoveryDueOldestAt:       unixTime(state.RecoveryDueOldestUnix),
			OrderingBlockedCount:      state.OrderingBlockedCount,
			OrderingBlockedOldestAt:   unixTime(state.OrderingBlockedOldestUnix),
			PoisonCount:               state.PoisonCount,
			PoisonOldestAt:            unixTime(state.PoisonOldestUnix),
			PublishedRetainedCount:    state.PublishedRetainedCount,
			PublishedRetainedOldestAt: unixTime(state.PublishedRetainedOldestUnix),
			OrderingHeadCount:         storage.OrderingHeadCount,
			EventsBytes:               storage.EventsBytes,
			EventsIndexBytes:          storage.EventsIndexBytes,
			OrderingHeadsBytes:        storage.OrderingHeadsBytes,
			OrderingHeadsIndexBytes:   storage.OrderingHeadsIndexBytes,
		}
		return nil
	})
	if err != nil {
		return StateObservation{}, fmt.Errorf("observe outbox state: %w", err)
	}
	return observation, nil
}

func (s *Store) withTelemetry(telemetry *Telemetry) *Store {
	if s == nil || telemetry == nil || s.telemetry == telemetry {
		return s
	}
	return &Store{pool: s.pool, queries: s.queries, telemetry: telemetry}
}

func (s *Store) recordOperation(ctx context.Context, operation string, started time.Time, err error) {
	if s == nil || s.telemetry == nil {
		return
	}
	outcome, errorType := operationOutcome(err), operationErrorType(err)
	switch {
	case errors.Is(err, ErrNoWork):
		outcome, errorType = "empty", "none"
	case errors.Is(err, ErrLeaseLost):
		outcome, errorType = "error", "lost_lease"
	case errors.Is(err, ErrInvalidEvent), errors.Is(err, ErrConfig), errors.Is(err, ErrOrderingSequence),
		errors.Is(err, ErrRedriveRejected), errors.Is(err, ErrRedriveConflict):
		outcome, errorType = "rejected", "validation"
	}
	s.telemetry.RecordOperation(ctx, operation, outcome, errorType, time.Since(started))
}

func eventFromClaimRow(row sqlcgen.ClaimOutboxEventRow) Event {
	event := Event{
		ID:          row.ID,
		Type:        row.EventType,
		Source:      row.Source,
		Destination: row.Destination,
		Schema:      row.SchemaName,
		OccurredAt:  timeValue(row.OccurredAt),
		Payload:     append([]byte(nil), row.Payload...),
		Metadata:    append([]byte(nil), row.Metadata...),
	}
	if row.OrderingKey != nil {
		event.OrderingKey = *row.OrderingKey
	}
	if row.OrderingSequence != nil {
		event.OrderingSequence = *row.OrderingSequence
	}
	return event
}

func validateProgressIdentity(id, token string) error {
	if err := validateText("id", id, maxTextBytes); err != nil {
		return err
	}
	if err := validateText("lease_token", token, maxTextBytes); err != nil {
		return err
	}
	return nil
}

func validateErrorClass(value string) error {
	if err := validateText("error_class", value, 64); err != nil {
		return err
	}
	return nil
}

func progressResult(operation string, rows int64, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if rows != 1 {
		return ErrLeaseLost
	}
	return nil
}

func recordFromRow(row sqlcgen.OutboxEvent) Record {
	event := Event{
		ID:          row.ID,
		Type:        row.EventType,
		Source:      row.Source,
		Destination: row.Destination,
		Schema:      row.SchemaName,
		OccurredAt:  timeValue(row.OccurredAt),
		Payload:     append([]byte(nil), row.Payload...),
		Metadata:    append([]byte(nil), row.Metadata...),
	}
	if row.OrderingKey != nil {
		event.OrderingKey = *row.OrderingKey
	}
	if row.OrderingSequence != nil {
		event.OrderingSequence = *row.OrderingSequence
	}
	record := Record{
		Event:             event,
		CreatedAt:         timeValue(row.CreatedAt),
		AvailableAt:       timeValue(row.AvailableAt),
		CycleAttemptCount: int(row.CycleAttemptCount),
		TotalAttemptCount: row.TotalAttemptCount,
		LastAttemptAt:     timeValue(row.LastAttemptAt),
		LeaseExpiresAt:    timeValue(row.LeaseExpiresAt),
		PublishedAt:       timeValue(row.PublishedAt),
		PoisonedAt:        timeValue(row.PoisonedAt),
		RedriveCount:      int(row.RedriveCount),
		LastRedrivenAt:    timeValue(row.LastRedrivenAt),
	}
	if row.LeaseToken != nil {
		record.LeaseToken = *row.LeaseToken
	}
	if row.LastErrorClass != nil {
		record.LastErrorClass = *row.LastErrorClass
	}
	if row.LastRedriveID != nil {
		record.LastRedriveID = *row.LastRedriveID
	}
	return record
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}

func unixTime(value float64) time.Time {
	if value == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64(value*float64(time.Second))).UTC()
}

func durationMilliseconds(value time.Duration) float64 {
	return float64(value) / float64(time.Millisecond)
}
