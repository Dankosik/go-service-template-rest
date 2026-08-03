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

// ClaimedBatch is one lease. Every event in it shares Token, and every
// finalization compares against that token, so the batch is fenced as a unit
// against a relay that overran its own lease.
type ClaimedBatch struct {
	Token  string
	Events []ClaimedEvent
}

type ClaimedEvent struct {
	Event             Event
	CycleAttemptCount int
	TotalAttemptCount int64
	Recovered         bool
}

// RetryDirective releases one leased event for a later attempt.
type RetryDirective struct {
	ID         string
	ErrorClass string
	Delay      time.Duration
}

// PoisonDirective parks one leased event for operator redrive.
type PoisonDirective struct {
	ID         string
	ErrorClass string
}

// OrderedDirective finalizes one acknowledged ordered event. Its key and
// sequence fence the head advance against a lease that was already recovered.
type OrderedDirective struct {
	ID               string
	OrderingKey      string
	OrderingSequence int64
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

func NewStore(pool *postgres.Pool, telemetry *Telemetry) (*Store, error) {
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	return &Store{pool: pool, queries: sqlcgen.New(pool.PGX()), telemetry: telemetry}, nil
}

// Append stores every event in the transaction owned by the feature caller. It
// never begins or commits a transaction itself.
//
// The events are pipelined, so one call costs one network round trip no matter
// how many events it carries: a feature that emits several events per business
// transaction pays the latency of one, and holds its own row locks for that
// much less time. A mixed ordered and unordered call costs two, one per
// statement shape. Nothing is sent unless every event is valid, and events for
// the same ordering key are stored in the order they were passed.
//
// One call is one append operation in telemetry, because that is what the
// recorded duration measures; backlog gauges report events.
func (s *Store) Append(ctx context.Context, tx pgx.Tx, events ...Event) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "append", started, err) }()
	if s == nil || s.pool == nil || tx == nil {
		return fmt.Errorf("%w: store and transaction are required", ErrConfig)
	}
	if len(events) == 0 {
		return nil
	}

	var unordered []sqlcgen.InsertOutboxEventsParams
	var ordered []Event
	for _, event := range events {
		event = event.withDefaults()
		if err := event.Validate(); err != nil {
			return err
		}
		if event.OrderingKey != "" {
			ordered = append(ordered, event)
			continue
		}
		unordered = append(unordered, sqlcgen.InsertOutboxEventsParams{
			ID:          event.ID,
			EventType:   event.Type,
			Source:      event.Source,
			Destination: event.Destination,
			SchemaName:  event.Schema,
			OccurredAt:  timestamptz(event.OccurredAt),
			Payload:     event.Payload,
			Metadata:    event.Metadata,
		})
	}

	queries := sqlcgen.New(tx)
	if len(unordered) > 0 {
		if err := appendUnordered(ctx, queries, unordered); err != nil {
			return err
		}
	}
	if len(ordered) > 0 {
		return appendOrdered(ctx, queries, ordered)
	}
	return nil
}

// appendUnordered stores envelopes that own no ordering head. Every result is
// consumed even after a failure, because pgx requires the pipeline to be read
// out before the connection carries anything else.
func appendUnordered(ctx context.Context, queries *sqlcgen.Queries, params []sqlcgen.InsertOutboxEventsParams) error {
	var failure error
	queries.InsertOutboxEvents(ctx, params).Exec(func(_ int, err error) {
		if err != nil && failure == nil {
			failure = fmt.Errorf("insert outbox event: %w", err)
		}
	})
	return failure
}

// appendOrdered stores each event and advances its key's retained high-water
// mark in the same statement, so the caller's transaction holds a head row lock
// only for the pipeline. A statement that returns no row stored nothing, which
// happens exactly when the sequence is at or below that mark; the ordering
// authority rejects it.
func appendOrdered(ctx context.Context, queries *sqlcgen.Queries, events []Event) error {
	params := make([]sqlcgen.InsertOrderedOutboxEventsParams, len(events))
	for index := range events {
		event := &events[index]
		params[index] = sqlcgen.InsertOrderedOutboxEventsParams{
			ID:               event.ID,
			EventType:        event.Type,
			Source:           event.Source,
			Destination:      event.Destination,
			SchemaName:       event.Schema,
			OccurredAt:       timestamptz(event.OccurredAt),
			Payload:          event.Payload,
			Metadata:         event.Metadata,
			OrderingKey:      &event.OrderingKey,
			OrderingSequence: &event.OrderingSequence,
		}
	}
	var failure error
	queries.InsertOrderedOutboxEvents(ctx, params).QueryRow(func(index int, _ string, err error) {
		switch {
		case err == nil || failure != nil:
		case errors.Is(err, pgx.ErrNoRows):
			failure = fmt.Errorf(
				"%w: key %q sequence %d is not above the retained high-water mark",
				ErrOrderingSequence, events[index].OrderingKey, events[index].OrderingSequence,
			)
		default:
			failure = fmt.Errorf("insert ordered outbox event: %w", err)
		}
	})
	return failure
}

// Claim leases up to batchSize eligible events under one fresh token. An empty
// batch means no eligible work, which is an ordinary outcome rather than an
// error.
func (s *Store) Claim(ctx context.Context, leaseDuration time.Duration, batchSize int) (batch ClaimedBatch, err error) {
	started := time.Now()
	defer func() { s.recordClaim(ctx, batch, started, err) }()
	if s == nil || s.queries == nil {
		return ClaimedBatch{}, fmt.Errorf("%w: store is required", ErrConfig)
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
		if row.RecoveryDue && s.telemetry != nil {
			s.telemetry.RecordOperation(ctx, "recovery", "success", "none", time.Since(started))
			s.telemetry.LogRecovery(ctx, row.ID, int(row.CycleAttemptCount))
		}
	}
	return batch, nil
}

// MarkPublished finalizes one event. Unordered events normally finalize through
// MarkPublishedBatch; this path also serves the ordered head advance and the
// per-event reconciliation of a short batch write.
func (s *Store) MarkPublished(ctx context.Context, token string, claim ClaimedEvent) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	if err := validateProgressIdentity(claim.Event.ID, token); err != nil {
		return err
	}
	if claim.Event.OrderingKey == "" {
		rows, err := s.queries.MarkOutboxPublished(ctx, sqlcgen.MarkOutboxPublishedParams{
			ID: claim.Event.ID, LeaseToken: &token,
		})
		return progressResult("mark outbox published", rows, err)
	}

	marked, err := s.markOrderedPublished(ctx, token, []OrderedDirective{{
		ID:               claim.Event.ID,
		OrderingKey:      claim.Event.OrderingKey,
		OrderingSequence: claim.Event.OrderingSequence,
	}})
	if err != nil {
		return err
	}
	if len(marked) != 1 {
		return ErrLeaseLost
	}
	return nil
}

// MarkPublishedBatch finalizes every unordered event of one lease in a single
// statement and reports the ids it finalized. A missing id means that event's
// lease was lost, which the caller resolves against durable state.
func (s *Store) MarkPublishedBatch(ctx context.Context, token string, ids []string) (marked []string, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	if err := validateBatchIdentity(token, ids); err != nil {
		return nil, err
	}
	marked, err = s.queries.MarkOutboxPublishedBatch(ctx, sqlcgen.MarkOutboxPublishedBatchParams{
		Ids: ids, LeaseToken: &token,
	})
	if err != nil {
		return nil, fmt.Errorf("mark outbox published batch: %w", err)
	}
	return marked, nil
}

// MarkOrderedPublishedBatch finalizes every ordered event of one lease in a
// single statement and reports the ids it finalized. Each one also advances its
// key's head and unblocks that key's successor. A missing id means the lease no
// longer owns that event, which the caller resolves against durable state.
func (s *Store) MarkOrderedPublishedBatch(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) (marked []string, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	return s.markOrderedPublished(ctx, token, directives)
}

// markOrderedPublished retries only the directives whose key had a committed
// successor that the previous statement's snapshot could not yet see. A second
// statement takes a fresh snapshot, so one retry resolves it.
func (s *Store) markOrderedPublished(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) ([]string, error) {
	var marked []string
	for range 2 {
		rows, err := s.markOrderedPublishedOnce(ctx, token, directives)
		if err != nil {
			return marked, err
		}
		for _, row := range rows {
			if row.Marked {
				marked = append(marked, row.ID)
			}
		}
		directives = conflictedDirectives(directives, rows)
		if len(directives) == 0 {
			break
		}
	}
	return marked, nil
}

func (s *Store) markOrderedPublishedOnce(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) ([]sqlcgen.MarkOrderedOutboxPublishedBatchRow, error) {
	ids := make([]string, len(directives))
	keys := make([]string, len(directives))
	sequences := make([]int64, len(directives))
	for index, directive := range directives {
		if err := validateText("ordering_key", directive.OrderingKey, maxTextBytes); err != nil ||
			directive.OrderingSequence < 1 {
			return nil, fmt.Errorf("%w: ordered claim identity is invalid", ErrConfig)
		}
		ids[index] = directive.ID
		keys[index] = directive.OrderingKey
		sequences[index] = directive.OrderingSequence
	}
	if err := validateBatchIdentity(token, ids); err != nil {
		return nil, err
	}
	rows, err := s.queries.MarkOrderedOutboxPublishedBatch(ctx, sqlcgen.MarkOrderedOutboxPublishedBatchParams{
		Ids: ids, OrderingKeys: keys, OrderingSequences: sequences, LeaseToken: &token,
	})
	if err != nil {
		return nil, fmt.Errorf("mark ordered outbox published batch: %w", err)
	}
	return rows, nil
}

// conflictedDirectives selects the directives the statement reported as
// snapshot conflicts. Result rows carry no guaranteed order, so it matches on
// id rather than position.
func conflictedDirectives(
	directives []OrderedDirective,
	rows []sqlcgen.MarkOrderedOutboxPublishedBatchRow,
) []OrderedDirective {
	var retry []OrderedDirective
	for _, row := range rows {
		if !row.SnapshotConflict {
			continue
		}
		for _, directive := range directives {
			if directive.ID == row.ID {
				retry = append(retry, directive)
				break
			}
		}
	}
	return retry
}

// ScheduleRetryBatch releases every failed event of one lease in a single
// statement, each with its own delay and error class.
func (s *Store) ScheduleRetryBatch(ctx context.Context, token string, retries []RetryDirective) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "schedule_retry", started, err) }()
	ids := make([]string, len(retries))
	classes := make([]string, len(retries))
	delays := make([]float64, len(retries))
	for index, retry := range retries {
		if err := validateErrorClass(retry.ErrorClass); err != nil {
			return err
		}
		if retry.Delay < 0 {
			return fmt.Errorf("%w: retry delay cannot be negative", ErrConfig)
		}
		ids[index] = retry.ID
		classes[index] = retry.ErrorClass
		delays[index] = durationMilliseconds(retry.Delay)
	}
	if err := validateBatchIdentity(token, ids); err != nil {
		return err
	}
	rows, err := s.queries.ScheduleOutboxRetryBatch(ctx, sqlcgen.ScheduleOutboxRetryBatchParams{
		LeaseToken: &token, Ids: ids, DelayMilliseconds: delays, ErrorClasses: classes,
	})
	return batchProgressResult("schedule outbox retry batch", rows, len(ids), err)
}

// MarkPoisonedBatch parks every exhausted or permanently rejected event of one
// lease in a single statement.
func (s *Store) MarkPoisonedBatch(ctx context.Context, token string, poisons []PoisonDirective) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "poison", started, err) }()
	ids := make([]string, len(poisons))
	classes := make([]string, len(poisons))
	for index, poison := range poisons {
		if err := validateErrorClass(poison.ErrorClass); err != nil {
			return err
		}
		ids[index] = poison.ID
		classes[index] = poison.ErrorClass
	}
	if err := validateBatchIdentity(token, ids); err != nil {
		return err
	}
	rows, err := s.queries.MarkOutboxPoisonedBatch(ctx, sqlcgen.MarkOutboxPoisonedBatchParams{
		LeaseToken: &token, Ids: ids, ErrorClasses: classes,
	})
	return batchProgressResult("mark outbox poisoned batch", rows, len(ids), err)
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
	if s == nil || s.queries == nil {
		return StateObservation{}, fmt.Errorf("%w: store is required", ErrConfig)
	}
	state, err := s.queries.ObserveOutbox(ctx)
	if err != nil {
		return StateObservation{}, fmt.Errorf("observe outbox state: %w", err)
	}
	return StateObservation{
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
		PublishedRetainedEstimate: state.PublishedRetainedEstimate,
		PublishedRetainedOldestAt: unixTime(state.PublishedRetainedOldestUnix),
		OrderingHeadCount:         state.OrderingHeadCount,
		EventsBytes:               state.EventsBytes,
		EventsIndexBytes:          state.EventsIndexBytes,
		OrderingHeadsBytes:        state.OrderingHeadsBytes,
		OrderingHeadsIndexBytes:   state.OrderingHeadsIndexBytes,
		RedrivesBytes:             state.RedrivesBytes,
		RedrivesIndexBytes:        state.RedrivesIndexBytes,
	}, nil
}

// listenerConfig is the connection the append listener opens for itself. It
// inherits the pool's validated DSN, TLS, and timeouts without ever competing
// for a pooled connection.
func (s *Store) listenerConfig() *pgx.ConnConfig {
	return s.pool.PGX().Config().ConnConfig
}

func (s *Store) withTelemetry(telemetry *Telemetry) *Store {
	if s == nil || telemetry == nil || s.telemetry == telemetry {
		return s
	}
	return &Store{pool: s.pool, queries: s.queries, telemetry: telemetry}
}

func (s *Store) recordClaim(ctx context.Context, batch ClaimedBatch, started time.Time, err error) {
	if s == nil || s.telemetry == nil {
		return
	}
	if err == nil && len(batch.Events) == 0 {
		s.telemetry.RecordOperation(ctx, "claim", "empty", "none", time.Since(started))
		return
	}
	s.recordOperation(ctx, "claim", started, err)
}

func (s *Store) recordOperation(ctx context.Context, operation string, started time.Time, err error) {
	if s == nil || s.telemetry == nil {
		return
	}
	outcome, errorType := operationOutcome(err), operationErrorType(err)
	switch {
	case errors.Is(err, ErrLeaseLost):
		outcome, errorType = outcomeError, "lost_lease"
	case errors.Is(err, ErrInvalidEvent), errors.Is(err, ErrConfig), errors.Is(err, ErrOrderingSequence),
		errors.Is(err, ErrRedriveRejected), errors.Is(err, ErrRedriveConflict):
		outcome, errorType = "rejected", "validation"
	}
	s.telemetry.RecordOperation(ctx, operation, outcome, errorType, time.Since(started))
}

// eventFromClaimRow adopts the scanned payload and metadata rather than copying
// them. pgx allocates a fresh slice per row for a bytea scanned into []byte
// ([pgtype bytea scan]), so these bytes are already private to this event and a
// second copy would double the per-event allocation of every claimed batch.
//
// [pgtype bytea scan]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype#ByteaCodec
func eventFromClaimRow(row sqlcgen.ClaimOutboxEventsRow) Event {
	event := Event{
		ID:          row.ID,
		Type:        row.EventType,
		Source:      row.Source,
		Destination: row.Destination,
		Schema:      row.SchemaName,
		OccurredAt:  timeValue(row.OccurredAt),
		Payload:     row.Payload,
		Metadata:    row.Metadata,
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

func validateBatchIdentity(token string, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("%w: batch requires at least one event", ErrConfig)
	}
	for _, id := range ids {
		if err := validateProgressIdentity(id, token); err != nil {
			return err
		}
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

func batchProgressResult(operation string, rows int64, expected int, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if rows != int64(expected) {
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
		Payload:     row.Payload,
		Metadata:    row.Metadata,
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
