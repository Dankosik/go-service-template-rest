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

// Store owns every outbox statement, for three audiences. Feature code calls
// only Append, inside the transaction that owns its domain mutation. The relay
// owns Claim, the Mark and ScheduleRetry family, CleanupPublished, and Observe,
// plus Get, which it uses to resolve a finalization the batch statement did not
// report. Redrive, and Get again, are operator tooling.
//
// Get therefore has two callers with different needs: Relay.reconcilePublished
// branches on both its error and the returned LeaseToken, so its error
// semantics are relay behavior as well as operator ergonomics.
//
// The Mark family is split by disposition rather than by arity: the relay sorts
// a batch into ordered and unordered claims once, in Relay.classify, and every
// method below is reached already knowing which it holds. No method here
// re-derives that split from the event it was handed.
//
// NewRelay does not mutate the Store it is given. When the relay's telemetry
// differs from the store's, the relay works through a copy carrying its own
// recorder, so a service that builds the two with different recorders keeps
// Append on the store's and the relay's operations on the relay's.
//
// NewStore is the only constructor and it rejects a nil pool, so a store that
// reached a caller has both fields. The remaining guards exist for the zero
// value an exported type cannot prevent: every exported method opens with
// valid(), so a method added later cannot admit a half-built Store, and there
// is no rule to remember about which ones need it.
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

// valid reports whether this Store came from NewStore. It is the one predicate
// every exported method uses, so a method added later cannot accidentally admit
// a half-built Store by testing only the field it happens to read.
func (s *Store) valid() bool {
	return s != nil && s.pool != nil && s.queries != nil
}

// errStoreRequired is what every exported method returns for a Store that never
// came from NewStore. One message, so the guard reads the same everywhere and a
// new method copies a line rather than inventing wording.
func errStoreRequired() error {
	return fmt.Errorf("%w: store is required", ErrConfig)
}

// Append stores every event in the transaction owned by the feature caller. It
// never begins or commits a transaction itself.
//
// One call is one statement and one round trip whatever mix of ordered and
// unordered events it carries, because the events travel as one array per
// column rather than one statement per event. A feature that emits several
// events per business transaction therefore pays the cost of a single append
// and holds its own row locks for that much less time. Nothing is sent unless
// every event is valid, and one rejected ordering sequence stores nothing.
//
// One call is one append operation in telemetry, because that is what the
// recorded duration measures; backlog gauges report events.
func (s *Store) Append(ctx context.Context, tx pgx.Tx, events ...Event) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "append", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	if tx == nil {
		return fmt.Errorf("%w: transaction is required", ErrConfig)
	}
	if len(events) == 0 {
		return nil
	}

	columns, err := newAppendColumns(events)
	if err != nil {
		return err
	}
	queries := sqlcgen.New(tx)
	if !columns.ordered {
		if err := queries.InsertOutboxEvents(ctx, columns.withoutOrdering()); err != nil {
			return fmt.Errorf("insert outbox events: %w", err)
		}
		return nil
	}

	rejected, err := queries.InsertOutboxEventsWithOrdering(ctx, columns.withOrdering())
	if err != nil {
		return fmt.Errorf("insert outbox events: %w", err)
	}
	if len(rejected) > 0 {
		return fmt.Errorf(
			"%w: key %q sequence %d is not above the retained high-water mark",
			ErrOrderingSequence, rejected[0].OrderingKey, rejected[0].FirstSequence,
		)
	}
	return nil
}

// appendColumns is one append laid out the way its statement reads it: one
// array per column instead of one struct per event.
type appendColumns struct {
	ids               []string
	types             []string
	sources           []string
	destinations      []string
	schemas           []string
	occurredAt        []pgtype.Timestamptz
	payloads          [][]byte
	metadatas         [][]byte
	orderingKeys      []string
	orderingSequences []int64
	ordered           bool
}

func (c appendColumns) withoutOrdering() sqlcgen.InsertOutboxEventsParams {
	return sqlcgen.InsertOutboxEventsParams{
		Ids: c.ids, EventTypes: c.types, Sources: c.sources, Destinations: c.destinations,
		SchemaNames: c.schemas, OccurredAts: c.occurredAt, Payloads: c.payloads, Metadatas: c.metadatas,
	}
}

func (c appendColumns) withOrdering() sqlcgen.InsertOutboxEventsWithOrderingParams {
	return sqlcgen.InsertOutboxEventsWithOrderingParams{
		Ids: c.ids, EventTypes: c.types, Sources: c.sources, Destinations: c.destinations,
		SchemaNames: c.schemas, OccurredAts: c.occurredAt, Payloads: c.payloads, Metadatas: c.metadatas,
		OrderingKeys: c.orderingKeys, OrderingSequences: c.orderingSequences,
	}
}

// newAppendColumns validates every event and transposes the call into those
// arrays. An event that owns no ordering head keeps the zero key and sequence
// it already has in Go, which is what the ordered statement reads as an absent
// head; a call where every event is like that skips that statement entirely.
func newAppendColumns(events []Event) (appendColumns, error) {
	columns := appendColumns{
		ids:               make([]string, len(events)),
		types:             make([]string, len(events)),
		sources:           make([]string, len(events)),
		destinations:      make([]string, len(events)),
		schemas:           make([]string, len(events)),
		occurredAt:        make([]pgtype.Timestamptz, len(events)),
		payloads:          make([][]byte, len(events)),
		metadatas:         make([][]byte, len(events)),
		orderingKeys:      make([]string, len(events)),
		orderingSequences: make([]int64, len(events)),
	}
	for index, event := range events {
		event = event.withDefaults()
		if err := event.Validate(); err != nil {
			return appendColumns{}, err
		}
		columns.ids[index] = event.ID
		columns.types[index] = event.Type
		columns.sources[index] = event.Source
		columns.destinations[index] = event.Destination
		columns.schemas[index] = event.Schema
		columns.occurredAt[index] = timestamptz(event.OccurredAt)
		columns.payloads[index] = event.Payload
		columns.metadatas[index] = event.Metadata
		columns.orderingKeys[index] = event.OrderingKey
		columns.orderingSequences[index] = event.OrderingSequence
		columns.ordered = columns.ordered || event.OrderingKey != ""
	}
	return columns, nil
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

// The two single-event finalizations below exist for one caller,
// Relay.reconcilePublished, resolving an event a batch statement did not
// report. They look like one-event wrappers over the batch statements, and
// deleting them as such would lose the signal they exist for: a batch statement
// reports a missing id by omitting it from its result and returns a nil error,
// so recordOperation classes the call a success. These return ErrLeaseLost
// instead, which is what puts error.type="lost_lease" on the reconciliation's
// outbox.relay.operations sample. Reconciliation is exactly where an operator
// needs to see a lost lease, because the relay is about to stop over it.
//
// They are two methods rather than one that branches on the ordering key,
// because the relay has already sorted the batch into ordered and unordered
// claims before either can be reached — see Relay.classify. Re-deriving that
// split here would be a second copy of the same predicate, and a relay that
// ever routed on anything else would silently reach the wrong statement.

// MarkUnorderedPublished finalizes one unordered event of a lease.
func (s *Store) MarkUnorderedPublished(ctx context.Context, token, id string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	if err := validateProgressIdentity(id, token); err != nil {
		return err
	}
	rows, err := s.queries.MarkOutboxPublished(ctx, sqlcgen.MarkOutboxPublishedParams{
		ID: id, LeaseToken: &token,
	})
	return progressResult("mark outbox published", rows, err)
}

// MarkOrderedPublished finalizes one ordered event of a lease, advancing its
// key's head. It routes through the same snapshot retry the batch statement
// uses, so one directive costs up to orderedPublishSnapshots statements.
func (s *Store) MarkOrderedPublished(ctx context.Context, token string, directive OrderedDirective) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	marked, err := s.retryOrderedSnapshotConflicts(ctx, token, []OrderedDirective{directive})
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
	if !s.valid() {
		return nil, errStoreRequired()
	}
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
// key's head and unblocks that key's successor.
//
// A missing id means either that the lease no longer owns that event or that
// its key was still snapshot-conflicted after the retry below. Both leave the
// event's durable state unresolved here, so the caller resolves it per event
// against durable state rather than assuming either outcome.
func (s *Store) MarkOrderedPublishedBatch(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) (marked []string, err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "mark_published", started, err) }()
	if !s.valid() {
		return nil, errStoreRequired()
	}
	return s.retryOrderedSnapshotConflicts(ctx, token, directives)
}

// orderedPublishSnapshots is how many snapshots the ordered finalization gets.
// Each statement takes a fresh one, so a successor that was committed but still
// invisible to the first is visible to the second, and one retry resolves it.
//
// Raising it raises the finalization budget the lease has to cover.
// [ErrProgressUnknown] derives that worst case from this count and
// reconcilePasses, and owns the derivation; state it there, not here.
const orderedPublishSnapshots = 2

// retryOrderedSnapshotConflicts sends the ordered finalization, then resends
// only the directives whose key had a committed successor that the previous
// statement's snapshot could not yet see.
func (s *Store) retryOrderedSnapshotConflicts(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) ([]string, error) {
	var marked []string
	for range orderedPublishSnapshots {
		rows, err := s.sendOrderedPublishBatch(ctx, token, directives)
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
	// A directive still conflicted after both statements simply stays out of
	// marked. The caller cannot tell it apart from a lost lease and resolves it
	// per event against durable state, which ends in ErrProgressUnknown when
	// that resolution also fails.
	return marked, nil
}

// sendOrderedPublishBatch transposes the directives into the statement's column
// arrays and runs it once, reporting each input's own outcome.
func (s *Store) sendOrderedPublishBatch(
	ctx context.Context,
	token string,
	directives []OrderedDirective,
) ([]sqlcgen.MarkOrderedOutboxPublishedBatchRow, error) {
	ids := make([]string, len(directives))
	keys := make([]string, len(directives))
	sequences := make([]int64, len(directives))
	for index, directive := range directives {
		if err := validateText(ErrConfig, "ordering_key", directive.OrderingKey, maxTextBytes); err != nil {
			return nil, err
		}
		if directive.OrderingSequence < 1 {
			return nil, fmt.Errorf("%w: ordering_sequence must be positive", ErrConfig)
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
	if !s.valid() {
		return errStoreRequired()
	}
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
	if !s.valid() {
		return errStoreRequired()
	}
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
	if !s.valid() {
		return Record{}, errStoreRequired()
	}
	if err := validateText(ErrConfig, "id", id, maxTextBytes); err != nil {
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

// Redrive releases one poisoned event for another delivery cycle, recording
// auditID so a repeated call for the same audit id is idempotent.
//
// Unlike Append it owns its transaction, so its locking and audit-conflict
// rules cannot be proven against a stubbed driver; TestPostgresOutboxRedrive in
// test/postgres_outbox_integration_test.go is where its contract is exercised.
func (s *Store) Redrive(ctx context.Context, id, auditID string) (err error) {
	started := time.Now()
	defer func() { s.recordOperation(ctx, "redrive", started, err) }()
	if !s.valid() {
		return errStoreRequired()
	}
	if err := validateText(ErrConfig, "id", id, maxTextBytes); err != nil {
		return err
	}
	if err := validateText(ErrConfig, "audit_id", auditID, maxTextBytes); err != nil {
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

// The nil check here and in recordOperation is for the receiver, not for the
// recorder: a nil *Telemetry is already a working no-op, but these run from
// deferred calls that a zero-value Store can reach before its entry-point guard
// returns.
func (s *Store) recordClaim(ctx context.Context, batch ClaimedBatch, started time.Time, err error) {
	if s == nil {
		return
	}
	if err == nil && len(batch.Events) == 0 {
		s.telemetry.RecordOperation(ctx, "claim", "empty", classNone, time.Since(started))
		return
	}
	s.recordOperation(ctx, "claim", started, err)
}

// storeOutcome classes one store failure into the outcome and error class its
// metric sample carries. It is the only producer of the lost-lease, validation,
// and database classes, so TestErrorClassVocabularyIsBounded drives it rather
// than restating those literals.
//
// Every statement in this file reaches PostgreSQL, so anything this switch does
// not recognize is a database error. A new sentinel the store can distinguish
// belongs here and in [boundedErrorType].
func storeOutcome(err error) (outcome, errorClass string) {
	switch {
	case err == nil:
		return outcomeSuccess, classNone
	case errors.Is(err, ErrLeaseLost):
		return outcomeError, classLostLease
	case errors.Is(err, ErrInvalidEvent), errors.Is(err, ErrConfig), errors.Is(err, ErrOrderingSequence),
		errors.Is(err, ErrRedriveRejected), errors.Is(err, ErrRedriveConflict):
		return outcomeRejected, classValidation
	default:
		return outcomeError, classDatabase
	}
}

func (s *Store) recordOperation(ctx context.Context, operation string, started time.Time, err error) {
	if s == nil {
		return
	}
	outcome, errorClass := storeOutcome(err)
	s.telemetry.RecordOperation(ctx, operation, outcome, errorClass, time.Since(started))
}

// envelopeColumns is the set of columns every outbox row carries. It is a
// struct rather than a parameter list because five of them are adjacent
// strings: transposing source and destination at a call site would compile, and
// only a test that asserts on those two exact values would notice.
type envelopeColumns struct {
	ID               string
	Type             string
	Source           string
	Destination      string
	Schema           string
	OccurredAt       pgtype.Timestamptz
	Payload          []byte
	Metadata         []byte
	OrderingKey      *string
	OrderingSequence *int64
}

// storedEvent maps those columns into an Event. The claim projection and the
// full row are separate generated types with the same columns, so this is the
// one place that knows the mapping: an added Event field costs a field here and
// one line at each call site instead of a second transposition block that can
// silently drift from the first.
//
// It adopts the scanned payload and metadata rather than copying them. pgx
// allocates a fresh slice per row for a bytea scanned into []byte ([pgtype
// bytea scan]), so these bytes are already private to this event and a second
// copy would double the per-event allocation of every claimed batch.
//
// [pgtype bytea scan]: https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype#ByteaCodec
func storedEvent(columns envelopeColumns) Event {
	event := Event{
		ID:          columns.ID,
		Type:        columns.Type,
		Source:      columns.Source,
		Destination: columns.Destination,
		Schema:      columns.Schema,
		OccurredAt:  timeValue(columns.OccurredAt),
		Payload:     columns.Payload,
		Metadata:    columns.Metadata,
	}
	if columns.OrderingKey != nil {
		event.OrderingKey = *columns.OrderingKey
	}
	if columns.OrderingSequence != nil {
		event.OrderingSequence = *columns.OrderingSequence
	}
	return event
}

func eventFromClaimRow(row sqlcgen.ClaimOutboxEventsRow) Event {
	return storedEvent(envelopeColumns{
		ID: row.ID, Type: row.EventType, Source: row.Source, Destination: row.Destination,
		Schema: row.SchemaName, OccurredAt: row.OccurredAt, Payload: row.Payload,
		Metadata: row.Metadata, OrderingKey: row.OrderingKey, OrderingSequence: row.OrderingSequence,
	})
}

// The identity checks below report [ErrConfig], not [ErrInvalidEvent]: an
// empty id, lease token, or error class is a fault in the call rather than in
// an envelope, and it is the caller — the relay or an operator tool — that has
// to fix it. validateText owns the shared rules; see its comment for the split.
func validateProgressIdentity(id, token string) error {
	if err := validateText(ErrConfig, "id", id, maxTextBytes); err != nil {
		return err
	}
	if err := validateText(ErrConfig, "lease_token", token, maxTextBytes); err != nil {
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

// maxErrorClassBytes bounds the stored error class. It is far below the general
// text limit because the class is a metric label as well as a column, and its
// values come from this package's own closed set.
const maxErrorClassBytes = 64

func validateErrorClass(value string) error {
	return validateText(ErrConfig, "error_class", value, maxErrorClassBytes)
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
	record := Record{
		Event: storedEvent(envelopeColumns{
			ID: row.ID, Type: row.EventType, Source: row.Source, Destination: row.Destination,
			Schema: row.SchemaName, OccurredAt: row.OccurredAt, Payload: row.Payload,
			Metadata: row.Metadata, OrderingKey: row.OrderingKey, OrderingSequence: row.OrderingSequence,
		}),
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
