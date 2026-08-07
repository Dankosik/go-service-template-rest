package postgresoutbox

// This file is the label vocabulary: every literal this package puts on a
// metric attribute, and every error class it also writes to an operator log and
// the stored last_error_class column, is named here — so a maintainer adding one
// has a list to join rather than a string to invent.
//
// All three attributes are bounded below, because an unrecognized value on a
// metric mints a time series. The bounding functions live beside the constants
// they close over, so a producer and its vocabulary reference one identifier
// rather than two literals that can drift apart.
//
// One closed vocabulary is deliberately not here: listenerStage in notify.go
// labels a single log field of a single function and never reaches a metric, so
// it stays beside the failures it names.
//
// telemetry.go holds the instruments, the snapshot, the recorders, and the
// scrape callback that put these values on a sample.

// The outcome labels every operation site chooses between. Attribute values are
// a fixed enum, so they are named rather than repeated.
const (
	outcomeSuccess  = "success"
	outcomeError    = "error"
	outcomeRejected = "rejected"
	// boundedOther is what every closed vocabulary in this package collapses an
	// unrecognized value to, so an unexpected string cannot mint a time series.
	boundedOther = "other"
)

// The closed error.type vocabulary. Every class this package produces is named
// here and used at its producing site — storeOutcome in store.go,
// publicationErrorClass in relay_publish.go, and Relay.classify in
// relay_finalize.go. A class added at a call site without a constant here is
// what TestErrorClassVocabularyIsBounded fails on.
const (
	classNone               = "none"
	classDatabase           = "database"
	classLostLease          = "lost_lease"
	classValidation         = "validation"
	classStuck              = "stuck"
	classPanic              = "panic"
	classPublisherPermanent = "publisher_permanent"
	classPublisherRejected  = "publisher_rejected"
	classPublisherTimeout   = "publisher_timeout"
	classPublisherCanceled  = "publisher_canceled"
	classPublisherTemporary = "publisher_temporary"
	classAttemptExhausted   = "attempt_exhausted"
)

// boundedOperation is the closed vocabulary for the operation attribute. The
// unit differs by operation, and the duration histogram is only meaningful per
// operation because of it:
//
//   - append, claim, mark_published, schedule_retry, poison, redrive, cleanup,
//     and observe are one statement each, and carry that statement's duration.
//   - publish is one event, not one batch.
//   - recovery and drain, and the reconciled outcome of mark_published, are
//     counted through CountOperation and carry no duration.
//
// A new operation states its unit here and picks the matching recorder.
func boundedOperation(value string) string {
	switch value {
	case "append", "claim", "recovery", "publish", "mark_published", "schedule_retry", "poison", "redrive", "cleanup", "observe", "drain":
		return value
	default:
		return boundedOther
	}
}

func boundedOutcome(value string) string {
	switch value {
	case outcomeSuccess, outcomeError, outcomeRejected, "empty", "reconciled", "started":
		return value
	default:
		return boundedOther
	}
}

// boundedErrorType is the closed vocabulary for error.type. One class describes
// a failure in three places — this metric attribute, the error.type field of
// LogPoison, and the stored last_error_class column — and only this function
// bounds it, because the column's CHECK constraint bounds length alone. Any
// class the package produces must therefore appear here, or the metric silently
// reports "other" while the log and the row report the real value.
// TestErrorClassVocabularyIsBounded drives the producers and fails on a class
// this list forgot.
func boundedErrorType(value string) string {
	switch value {
	case classNone, classDatabase, classLostLease, classValidation, classStuck, classPanic,
		classPublisherPermanent, classPublisherRejected, classPublisherTimeout,
		classPublisherCanceled, classPublisherTemporary, classAttemptExhausted:
		return value
	default:
		return boundedOther
	}
}
