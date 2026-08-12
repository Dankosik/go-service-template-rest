package httpidempotency

import "errors"

// Outcome is the closed set of decisions an idempotency Store may return.
type Outcome uint8

const (
	OutcomeExecute Outcome = iota + 1
	OutcomeReplay
	OutcomeMismatch
	OutcomeInProgress
	OutcomeExpired
	OutcomeRateLimited
	OutcomeUnavailable
	OutcomeUnknown
	OutcomeResultTooLarge
	OutcomeIntegrityConflict
)

// Decision is one Store or admission disposition. Replay carries the retained
// semantic result; every other outcome carries no result.
type Decision struct {
	Outcome Outcome
	Result  *Result
}

// Validate rejects an incomplete or contradictory decision before HTTP maps it.
func (d Decision) Validate() error {
	switch d.Outcome {
	case OutcomeExecute:
		if d.Result != nil {
			return errors.New("idempotency decision: execute has a result")
		}
	case OutcomeReplay:
		if d.Result == nil {
			return errors.New("idempotency decision: replay has no result")
		}
	case OutcomeMismatch, OutcomeInProgress, OutcomeExpired, OutcomeRateLimited, OutcomeUnavailable, OutcomeUnknown, OutcomeResultTooLarge, OutcomeIntegrityConflict:
		if d.Result != nil {
			return errors.New("idempotency decision: non-replay has a result")
		}
	default:
		return errors.New("idempotency decision: unknown outcome")
	}
	return nil
}
