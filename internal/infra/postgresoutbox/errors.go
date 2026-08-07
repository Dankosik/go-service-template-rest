package postgresoutbox

import "errors"

// Every failure this package produces wraps one of the sentinels below, and
// both the relay's stop path and operator tooling route on them rather than on
// message text. This file exists so a reader looking for the set has somewhere
// to find it: they are produced across five files, and failureClass in
// cmd/outbox-relay/main.go — the complete operator-facing exit vocabulary — has
// to know all of them. A sentinel added here belongs in that switch too when it
// can stop the relay.
//
// They are grouped by who is expected to see one. The groups are a reading aid,
// not a contract; the doc comment on each sentinel owns its meaning.

// Rejected before any statement is sent: a fault in the call or in the envelope
// it was given.
var (
	// ErrConfig reports a fault in the call rather than in any event — a missing
	// store, pool, or transaction, or an empty id, lease token, or error class.
	ErrConfig = errors.New("outbox store config")
	// ErrInvalidEvent means one [Event] failed [Event.Validate], and nothing
	// else. See validateText for why it is kept distinct from ErrConfig.
	ErrInvalidEvent = errors.New("invalid outbox event")
)

// Returned by the append and lookup statements themselves.
var (
	// ErrNotFound means no row carries the requested id.
	ErrNotFound = errors.New("outbox event not found")
	// ErrOrderingSequence means an ordering sequence was not above its key's
	// retained high-water mark. Only PostgreSQL holds that mark, so this comes
	// back from the append statement rather than from validation, and the call
	// stores nothing.
	ErrOrderingSequence = errors.New("outbox ordering sequence rejected")
	// ErrOrderingKeyActive means [Store.RetireOrderingKeys] was asked to retire a
	// key that still owns unpublished events, so its retained high-water mark is
	// still protecting them. It is an ordinary domain outcome rather than a
	// fault: a caller may absorb it and retire the key later. It is a distinct
	// sentinel for exactly that reason — "this aggregate is not finished
	// draining" must be distinguishable from a database or configuration
	// failure, which a caller cannot absorb.
	ErrOrderingKeyActive = errors.New("outbox ordering key still has unpublished events")
)

// The relay's stop reasons. Each one ends [Relay.Run] and has a case in
// failureClass in cmd/outbox-relay/main.go.
var (
	// ErrLeaseLost means a finalization reached fewer rows than the lease still
	// owned, so this relay no longer owns them. It is the ordinary way a lease
	// that is too short, or a second replica, shows up.
	ErrLeaseLost = errors.New("outbox lease lost")
	// ErrProgressUnknown means the relay could neither record nor disprove a
	// publication: the event may already be at the broker while PostgreSQL still
	// shows it unpublished, so lease recovery will republish it. It stops the
	// relay, and two paths reach it. The ordinary one is a lost lease that
	// reconcilePublished could not resolve against durable state. The rare one is
	// an ordered acknowledgement whose key stayed snapshot-conflicted on every
	// statement it was sent on: orderedPublishSnapshots inside
	// Store.MarkOrderedPublishedBatch, then that many again on each of
	// reconcilePasses passes. This product is the whole worst case, and this is
	// the only place that states it; both constants point here rather than
	// restating it, and no comment or test writes the number down.
	//
	// Those statements share one deadline, the lease the batch was claimed under
	// — see publishBatch. Nothing sizes the lease for that worst case, and
	// deliberately so: a finalization that runs out of lease ends here, which is
	// the outcome this error exists to report, rather than claiming a publication
	// it cannot prove.
	ErrProgressUnknown = errors.New("outbox published-state progress is unknown")
	// ErrPublisherStuck means a publisher goroutine ignored cancellation past
	// the join bound. Its durable work is unresolved and it can still reach the
	// process's dependencies, which is why it also makes cleanup unsafe.
	ErrPublisherStuck = errors.New("outbox publisher did not stop after cancellation")
	// ErrPublisherPanic means an adapter panicked. That event is released for
	// retry and the rest of its batch still finalizes, but the relay stops so a
	// broken adapter surfaces as a process failure instead of a silent backlog.
	ErrPublisherPanic = errors.New("outbox publisher panicked")
)

// The two an adapter returns. [Publisher] owns the whole acceptance contract
// they belong to, including what returning one instead of the other costs.
var (
	// ErrPermanentPublication lets an adapter reject an occurrence without using
	// transport-specific error types in the relay. It poisons the event on the
	// first occurrence: the row is never retried, it blocks later work for its
	// ordering key, and only an operator redrive releases it. Return it only for a
	// rejection that retrying the same bytes cannot fix.
	ErrPermanentPublication = errors.New("permanent outbox publication failure")
	// ErrPublicationNotAccepted lets an adapter prove that the broker did not
	// durably accept an occurrence. It remains retryable and is the only failure
	// the attempt threshold poisons on; unclassified errors stay ambiguous and
	// keep retrying rather than risking loss at a strict attempt cap.
	ErrPublicationNotAccepted = errors.New("outbox publication was not accepted")
)

// Operator redrive. Neither reaches the relay: [Store.Redrive] is tooling.
var (
	// ErrRedriveRejected means the row is not in a state a redrive may release —
	// not poisoned, already published, or out of redrive count.
	ErrRedriveRejected = errors.New("outbox redrive rejected")
	// ErrRedriveConflict means the audit id already belongs to another event,
	// which is a caller fault rather than a race. See auditIDAlreadySpent.
	ErrRedriveConflict = errors.New("outbox redrive audit conflict")
)
