package postgresjobs

import (
	"fmt"
	"slices"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
)

var databaseStates = []jobs.State{
	jobs.StateReady,
	jobs.StateScheduled,
	jobs.StateRetryWait,
	jobs.StateRunning,
	jobs.StateCancelRequested,
	jobs.StateSucceeded,
	jobs.StateCancelled,
	jobs.StateExhausted,
	jobs.StatePermanent,
	jobs.StatePoison,
	jobs.StateOutcomeUnknown,
}

var databaseOutcomes = []jobs.OutcomeClass{
	jobs.OutcomeSuccess,
	jobs.OutcomeRetryable,
	jobs.OutcomePermanent,
	jobs.OutcomePoison,
	jobs.OutcomeTimeout,
	jobs.OutcomeCancelled,
	jobs.OutcomePanic,
	jobs.OutcomeLost,
	jobs.OutcomeUnknown,
}

var databaseEffects = []jobs.EffectStatus{
	jobs.EffectNone,
	jobs.EffectCompleted,
	jobs.EffectPartial,
	jobs.EffectUnknown,
}

func stateFromDatabase(value string) (jobs.State, error) {
	return vocabularyValue("state", value, databaseStates)
}

func outcomeFromDatabase(value string) (jobs.OutcomeClass, error) {
	return vocabularyValue("outcome", value, databaseOutcomes)
}

func effectFromDatabase(value string) (jobs.EffectStatus, error) {
	return vocabularyValue("effect_status", value, databaseEffects)
}

func acceptanceIdentityFromReadback(row sqlcgen.ReadPostgresJobsAcceptanceRow) (jobs.AcceptanceIdentity, bool) {
	if row.LogicalJobID == nil || row.ProducerScope == nil || row.ProducerKey == nil ||
		row.OccurrenceScope == nil || row.OccurrenceID == nil || row.EffectScope == nil || row.EffectKey == nil {
		return jobs.AcceptanceIdentity{}, false
	}
	return jobs.AcceptanceIdentity{
		LogicalJobID:    jobs.LogicalJobID(*row.LogicalJobID),
		ProducerScope:   jobs.ProducerScope(*row.ProducerScope),
		ProducerKey:     jobs.ProducerKey(*row.ProducerKey),
		OccurrenceScope: jobs.OccurrenceScope(*row.OccurrenceScope),
		OccurrenceID:    jobs.OccurrenceID(*row.OccurrenceID),
		EffectScope:     jobs.EffectScope(*row.EffectScope),
		EffectKey:       jobs.EffectKey(*row.EffectKey),
	}, true
}

func vocabularyValue[T ~string](name, value string, allowed []T) (T, error) {
	candidate := T(value)
	if slices.Contains(allowed, candidate) {
		return candidate, nil
	}
	var zero T
	return zero, fmt.Errorf("%w: %s %q", ErrUnknownVocabulary, name, value)
}
