package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// poisonedEvent pairs the parked claim with the class that parked it, so the
// operator log keeps the attempt count the store type has no use for.
type poisonedEvent struct {
	claim                ClaimedEvent
	errorClass           string
	publicationUncertain bool
}

// batchOutcomes groups one batch by the durable transition each event needs.
type batchOutcomes struct {
	published []ClaimedEvent
	ordered   []ClaimedEvent
	retries   []RetryDirective
	poisoned  []poisonedEvent
}

// finalize records the durable transition every event of one batch earned. Each
// of the four statements can come back short — reporting fewer events than it
// was given — and the two halves answer that differently on purpose:
//
//   - An acknowledgement the statement did not report may already be at the
//     broker, and leaving it unmarked creates a duplicate, so finalizeUnordered
//     and finalizeOrdered resolve each missing event against durable state. That
//     resolution costs up to reconcilePasses statements per leftover event, and
//     for an ordered one each pass is itself worth orderedPublishSnapshots
//     statements inside the store; [ErrProgressUnknown] owns what that product
//     means for the lease.
//   - A retry or poison the statement did not report means another relay owns
//     that event and will deliver it. Nothing is at risk, but the lease was
//     overrun, which is a lease or replica misconfiguration rather than a
//     transient fault: Store reports ErrLeaseLost, which the loop surfaces to an
//     operator through [finalizationFaultTolerance].
//
// Every error returned here ends the current cycle, and the two finalization
// faults below end the relay once they repeat. failureClass in
// cmd/outbox-relay/main.go turns each stop reason into its operator-facing exit
// class, so a new one belongs in that switch too.
func (r *Relay) finalize(ctx context.Context, batch ClaimedBatch, publications []error) error {
	outcomes := r.classify(batch.Events, publications)
	if err := r.finalizeUnordered(ctx, batch.Token, outcomes.published); err != nil {
		return err
	}
	if err := r.finalizeOrdered(ctx, batch.Token, outcomes.ordered); err != nil {
		return err
	}
	if len(outcomes.retries) > 0 {
		if err := r.store.ScheduleRetryBatch(ctx, batch.Token, outcomes.retries); err != nil {
			// Named per transition: both this and the poison write below stop the
			// relay under the same exit class, so the message is all an operator
			// has to tell which statement failed.
			return fmt.Errorf("schedule outbox retry: %w", err)
		}
	}
	return r.finalizePoisoned(ctx, batch.Token, outcomes.poisoned)
}

// classify groups one batch by the durable transition each event needs.
// publications is indexed by claims: each entry is that event's publication
// error, or nil once the broker acknowledged it under the publisher's
// configured durability contract.
func (r *Relay) classify(claims []ClaimedEvent, publications []error) batchOutcomes {
	var outcomes batchOutcomes
	for index, claim := range claims {
		publication := publications[index]
		atLimit := claim.CycleAttemptCount >= r.config.MaxAttempts
		permanent := errors.Is(publication, ErrPermanentPublication)
		notAccepted := errors.Is(publication, ErrPublicationNotAccepted)
		switch {
		case publication == nil && claim.Event.OrderingKey == "":
			outcomes.published = append(outcomes.published, claim)
		case publication == nil:
			outcomes.ordered = append(outcomes.ordered, claim)
		case errors.Is(publication, errPublicationNotAttempted):
			// Above every failure arm on purpose: this event was never handed to
			// the publisher, so neither the attempt cap nor sticky uncertainty may
			// act on it — including for an event that arrived already uncertain,
			// which the arm below would otherwise poison.
			//
			// No delay, because nothing failed and nothing is being spared. The
			// event becomes eligible immediately, and the claim order puts it ahead
			// of whatever is retried for a real failure on the next cycle.
			outcomes.retries = append(outcomes.retries, RetryDirective{
				ID:                   claim.Event.ID,
				ErrorClass:           classPublisherNotAttempted,
				PublicationUncertain: claim.PublicationUncertain,
				NotAttempted:         true,
			})
		case claim.PublicationUncertain && (permanent || atLimit):
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{
				claim: claim, errorClass: classOutcomeUnknown, publicationUncertain: true,
			})
		case permanent:
			// The adapter's own class, so a poisoned row and the publish metric
			// that recorded the same failure agree without a second literal.
			outcomes.poisoned = append(
				outcomes.poisoned,
				poisonedEvent{claim: claim, errorClass: publicationErrorClass(publication)},
			)
		case notAccepted && atLimit:
			// Exhaustion is this package's policy rather than the adapter's
			// verdict, so it is the one class publicationErrorClass cannot name.
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{claim: claim, errorClass: classAttemptExhausted})
		case atLimit:
			outcomes.poisoned = append(outcomes.poisoned, poisonedEvent{
				claim: claim, errorClass: classOutcomeUnknown, publicationUncertain: true,
			})
		default:
			outcomes.retries = append(outcomes.retries, RetryDirective{
				ID:                   claim.Event.ID,
				ErrorClass:           publicationErrorClass(publication),
				Delay:                retryDelay(r.config.RetryBase, r.config.RetryMax, claim.CycleAttemptCount, r.jitter),
				PublicationUncertain: claim.PublicationUncertain || !notAccepted,
			})
		}
	}
	return outcomes
}

// finalizeUnordered finalizes every unordered acknowledgement of one lease in a
// single statement.
func (r *Relay) finalizeUnordered(ctx context.Context, token string, claims []ClaimedEvent) error {
	if len(claims) == 0 {
		return nil
	}
	ids := make([]string, len(claims))
	for index, claim := range claims {
		ids[index] = claim.Event.ID
	}
	marked, err := r.store.MarkUnorderedPublishedBatch(ctx, token, ids)
	return r.reconcileRemainder(ctx, token, claims, marked, err, r.markOneUnordered)
}

// finalizeOrdered finalizes every ordered acknowledgement of one lease in a
// single statement, which also advances each event's key head and unblocks that
// key's successor.
func (r *Relay) finalizeOrdered(ctx context.Context, token string, claims []ClaimedEvent) error {
	if len(claims) == 0 {
		return nil
	}
	directives := make([]OrderedDirective, len(claims))
	for index, claim := range claims {
		directives[index] = orderedDirective(claim)
	}
	marked, err := r.store.MarkOrderedPublishedBatch(ctx, token, directives)
	return r.reconcileRemainder(ctx, token, claims, marked, err, r.markOneOrdered)
}

// The two single-event marks reconciliation falls back to. Each caller above
// supplies the one matching the batch it just sent, so the ordered/unordered
// split stays where classify made it: nothing below re-derives it from the
// event, and a batch routed on some other rule cannot reach the wrong
// statement.
//
// Both pass the store's error through unchanged: reconcilePublished decides
// what it means, either discarding it once durable state proves the event
// published or joining it into [ErrProgressUnknown]. A wrap here would only
// prefix a message that ends up inside that join.
func (r *Relay) markOneUnordered(ctx context.Context, token string, claim ClaimedEvent) error {
	//nolint:wrapcheck // reconcilePublished classifies this error; see above.
	return r.store.MarkUnorderedPublished(ctx, token, claim.Event.ID)
}

func (r *Relay) markOneOrdered(ctx context.Context, token string, claim ClaimedEvent) error {
	//nolint:wrapcheck // As above.
	return r.store.MarkOrderedPublished(ctx, token, orderedDirective(claim))
}

func orderedDirective(claim ClaimedEvent) OrderedDirective {
	return OrderedDirective{
		ID:               claim.Event.ID,
		OrderingKey:      claim.Event.OrderingKey,
		OrderingSequence: claim.Event.OrderingSequence,
	}
}

// reconcileRemainder closes out a batch finalization. A short or failed write
// means a lost lease, or a crash window between broker acknowledgement and this
// update, so every event the statement did not report as finalized is resolved
// against durable state rather than assumed either way. markOne is the
// single-event statement matching the batch that came back short.
func (r *Relay) reconcileRemainder(
	ctx context.Context,
	token string,
	claims []ClaimedEvent,
	marked []string,
	markErr error,
	markOne markOneFunc,
) error {
	if len(marked) > 0 {
		r.telemetry.RecordProgress(time.Now())
	}
	if markErr == nil && len(marked) == len(claims) {
		return nil
	}
	finalized := make(map[string]struct{}, len(marked))
	for _, id := range marked {
		finalized[id] = struct{}{}
	}
	for _, claim := range claims {
		if _, ok := finalized[claim.Event.ID]; ok {
			continue
		}
		if err := r.reconcilePublished(ctx, token, claim, markOne); err != nil {
			return err
		}
	}
	return nil
}

func (r *Relay) finalizePoisoned(ctx context.Context, token string, poisoned []poisonedEvent) error {
	if len(poisoned) == 0 {
		return nil
	}
	directives := make([]PoisonDirective, len(poisoned))
	for index, event := range poisoned {
		directives[index] = PoisonDirective{
			ID: event.claim.Event.ID, ErrorClass: event.errorClass,
			PublicationUncertain: event.publicationUncertain,
		}
	}
	if err := r.store.MarkPoisonedBatch(ctx, token, directives); err != nil {
		return fmt.Errorf("poison outbox events: %w", err)
	}
	for _, event := range poisoned {
		r.telemetry.LogPoison(ctx, event.errorClass, event.claim.CycleAttemptCount)
	}
	return nil
}

// finalizationFaultTolerance is how many consecutive cycles may end in a
// finalization fault before the relay stops over it.
//
// Neither fault is a correctness problem the stop prevents. [ErrLeaseLost] means
// another relay already owns those events, and [ErrProgressUnknown] means lease
// recovery will republish one — a duplicate this package's delivery contract
// already permits. Stopping only makes an operator look, and stopping on the
// first occurrence makes them look at a crash loop: a single ambiguous event
// costs a whole process restart, and a lease that is persistently too short
// halts delivery entirely instead of degrading its throughput.
//
// A tolerance keeps both properties. Repetition is what distinguishes a
// misconfigured lease from a lost race, so a fault that repeats this many times
// in a row still exits with its own class, while an isolated one costs one
// poll interval and a counted sample. The count resets on any cycle that
// finalized cleanly, so the exit means consecutive faults rather than a total
// reached over an uptime.
//
// Three is the smallest count that can distinguish the two. Raising it raises
// how long a real misconfiguration keeps republishing before an operator is
// told; the duplicate rate it can produce meanwhile is bounded by this count
// times one batch.
const finalizationFaultTolerance = 3

// toleratesFinalizationFault reports whether a cycle's stop reason is one the
// relay absorbs and continues from, counting the ones it does. consecutive is
// how many cycles in a row have now ended in a fault, this one included.
//
// Only the two finalization faults are tolerable. Everything else that stops a
// cycle — a failed claim, a stuck publisher, a panicking adapter, a retention
// delete that failed — is either unsafe to continue under or not a fault at all,
// and reaches its caller unchanged.
func (r *Relay) toleratesFinalizationFault(ctx context.Context, err error, consecutive int) bool {
	class := finalizationFaultClass(err)
	if class == boundedOther || consecutive >= finalizationFaultTolerance {
		return false
	}
	r.telemetry.CountOperation(ctx, "finalize", "tolerated", class)
	return true
}

// finalizationFaultClass names the tolerable finalization faults for telemetry,
// and answers boundedOther for everything else — which is also how the caller
// above asks "is this one of them" without a second predicate.
func finalizationFaultClass(err error) string {
	switch {
	case errors.Is(err, ErrLeaseLost):
		return classLostLease
	case errors.Is(err, ErrProgressUnknown):
		return classProgressUnknown
	default:
		return boundedOther
	}
}

// reconcilePasses is how many times reconcilePublished re-resolves one event
// against durable state. A second pass is worth exactly one case: the mark
// failed, the event is still unpublished, and this lease still owns it, which is
// a lost race rather than a lost lease. Every other combination returns inside
// the first pass, so a third would only repeat a settled verdict.
//
// [ErrProgressUnknown] derives its worst case from this count and
// orderedPublishSnapshots, and owns the derivation; state it there, not here.
const reconcilePasses = 2

// markOneFunc is one lease's single-event finalization for one disposition.
// reconcileRemainder takes it rather than picking a statement itself, so the
// ordered/unordered split is written once, in Relay.classify.
type markOneFunc func(ctx context.Context, token string, claim ClaimedEvent) error

// reconcilePublished resolves one event whose batch finalization did not report
// it.
func (r *Relay) reconcilePublished(
	ctx context.Context,
	token string,
	claim ClaimedEvent,
	markOne markOneFunc,
) error {
	for range reconcilePasses {
		err := markOne(ctx, token, claim)
		if err == nil {
			r.telemetry.RecordProgress(time.Now())
			return nil
		}
		record, getErr := r.store.Get(ctx, claim.Event.ID)
		if getErr == nil && !record.PublishedAt.IsZero() {
			r.telemetry.CountOperation(ctx, "mark_published", "reconciled", classNone)
			r.telemetry.RecordProgress(time.Now())
			return nil
		}
		if getErr != nil || record.LeaseToken != token {
			return fmt.Errorf("%w: reconcile mark: %w", ErrProgressUnknown, errors.Join(err, getErr))
		}
	}
	return ErrProgressUnknown
}
