package postgresjobs

import "github.com/example/go-service-template-rest/internal/jobs"

const (
	metricOutcomeOther = "other"
	metricStateOther   = "other"
)

func metricEvent(event string) string {
	switch event {
	case "acceptance", "claim", "attempt", "retry", "rescue", "cancellation", "recovery", "action", "drain":
		return event
	default:
		return metricOutcomeOther
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
