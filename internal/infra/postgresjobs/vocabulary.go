package postgresjobs

import "github.com/example/go-service-template-rest/internal/jobs"

const (
	metricOutcomeOther   = "other"
	metricOperationOther = "other"
	metricStateOther     = "other"
)

func metricEvent(event string) string {
	switch event {
	case "acceptance", "acceptance_duplicate", "acceptance_conflict", "acceptance_rejected",
		"claim", "attempt", "retry", "rescue", "cancellation", "drain", "terminal_failure":
		return event
	default:
		return metricOutcomeOther
	}
}

func metricOperation(operation string) string {
	switch operation {
	case "claim", "resolve_claims", "check_schema", "renew", "rescue_candidates", "rescue", "finalize", "observe", "producer_probe":
		return operation
	default:
		return metricOperationOther
	}
}

func metricOutcome(outcome jobs.OutcomeClass) string {
	switch outcome {
	case jobs.OutcomeSuccess, jobs.OutcomeRetryable, jobs.OutcomePermanent, jobs.OutcomePoison,
		jobs.OutcomeTimeout, jobs.OutcomeCancelled, jobs.OutcomePanic, jobs.OutcomeLost, jobs.OutcomeUnknown:
		return string(outcome)
	default:
		return metricOutcomeOther
	}
}

func metricState(state jobs.State) string {
	switch state {
	case jobs.StateReady, jobs.StateScheduled, jobs.StateRetryWait, jobs.StateRunning, jobs.StateCancelRequested,
		jobs.StateSucceeded, jobs.StateCancelled, jobs.StateExhausted, jobs.StatePermanent, jobs.StatePoison, jobs.StateOutcomeUnknown:
		return string(state)
	default:
		return metricStateOther
	}
}
