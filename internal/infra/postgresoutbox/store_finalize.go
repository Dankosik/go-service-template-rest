package postgresoutbox

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
)

// RetryDirective releases one leased event for a later attempt.
type RetryDirective struct {
	ID                   string
	ErrorClass           string
	Delay                time.Duration
	PublicationUncertain bool
	// NotAttempted gives back the attempt the claim charged, for an event the
	// relay never handed to [Publisher]. It is stated negatively so the zero value
	// is the ordinary one: a directive that says nothing keeps its attempt, which
	// is what every real publication failure needs.
	NotAttempted bool
}

// PoisonDirective parks one leased event for operator redrive.
type PoisonDirective struct {
	ID                   string
	ErrorClass           string
	PublicationUncertain bool
}

// OrderedDirective finalizes one acknowledged ordered event. Its key and
// sequence fence the head advance against a lease that was already recovered.
type OrderedDirective struct {
	ID               string
	OrderingKey      string
	OrderingSequence int64
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
// They are two methods rather than one that branches on the ordering key, because
// Relay.classify has already sorted the batch before either can be reached.
// Re-deriving that split here would be a second copy of the same predicate.

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
	return leaseProgressError("mark outbox published", rows, 1, err)
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

// MarkUnorderedPublishedBatch finalizes every unordered event of one lease in a single
// statement and reports the ids it finalized. A missing id means that event's
// lease was lost, which the caller resolves against durable state.
func (s *Store) MarkUnorderedPublishedBatch(ctx context.Context, token string, ids []string) (marked []string, err error) {
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
	uncertain := make([]bool, len(retries))
	notAttempted := make([]bool, len(retries))
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
		uncertain[index] = retry.PublicationUncertain
		notAttempted[index] = retry.NotAttempted
	}
	if err := validateBatchIdentity(token, ids); err != nil {
		return err
	}
	rows, err := s.queries.ScheduleOutboxRetryBatch(ctx, sqlcgen.ScheduleOutboxRetryBatchParams{
		LeaseToken: &token, Ids: ids, DelayMilliseconds: delays, ErrorClasses: classes,
		PublicationUncertain: uncertain, NotAttempted: notAttempted,
	})
	return leaseProgressError("schedule outbox retry batch", rows, len(ids), err)
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
	uncertain := make([]bool, len(poisons))
	for index, poison := range poisons {
		if err := validateErrorClass(poison.ErrorClass); err != nil {
			return err
		}
		ids[index] = poison.ID
		classes[index] = poison.ErrorClass
		uncertain[index] = poison.PublicationUncertain
	}
	if err := validateBatchIdentity(token, ids); err != nil {
		return err
	}
	rows, err := s.queries.MarkOutboxPoisonedBatch(ctx, sqlcgen.MarkOutboxPoisonedBatchParams{
		LeaseToken: &token, Ids: ids, ErrorClasses: classes, PublicationUncertain: uncertain,
	})
	return leaseProgressError("mark outbox poisoned batch", rows, len(ids), err)
}

// leaseProgressError turns a finalization statement's row count into the error
// its caller needs: reaching fewer rows than the lease still owned means this
// lease no longer owns them.
//
// Only the writes that had to reach every event of their lease use it — the
// retry and poison batches, and the two single-event marks with an expected
// count of one. The published batches do not, because a short acknowledgement
// puts an event at risk of duplication and is resolved against durable state
// instead; a short retry or poison only proves the lease was overrun.
func leaseProgressError(operation string, rows int64, expected int, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if rows != int64(expected) {
		return ErrLeaseLost
	}
	return nil
}
