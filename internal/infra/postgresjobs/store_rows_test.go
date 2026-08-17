package postgresjobs

import (
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestStoreRowsVocabularyIsClosedAndBijective(t *testing.T) {
	for _, state := range databaseStates {
		got, err := stateFromDatabase(string(state))
		if err != nil || got != state {
			t.Fatalf("stateFromDatabase(%q) = %q, %v", state, got, err)
		}
	}
	for _, outcome := range databaseOutcomes {
		got, err := outcomeFromDatabase(string(outcome))
		if err != nil || got != outcome {
			t.Fatalf("outcomeFromDatabase(%q) = %q, %v", outcome, got, err)
		}
	}
	for _, effect := range databaseEffects {
		got, err := effectFromDatabase(string(effect))
		if err != nil || got != effect {
			t.Fatalf("effectFromDatabase(%q) = %q, %v", effect, got, err)
		}
	}
	if _, err := stateFromDatabase("future"); !errors.Is(err, ErrUnknownVocabulary) {
		t.Fatalf("unknown state error = %v", err)
	}
	if _, err := outcomeFromDatabase("future"); !errors.Is(err, ErrUnknownVocabulary) {
		t.Fatalf("unknown outcome error = %v", err)
	}
	if _, err := effectFromDatabase("future"); !errors.Is(err, ErrUnknownVocabulary) {
		t.Fatalf("unknown effect error = %v", err)
	}
}

func TestStoreRowsAcceptanceReadbackRequiresCompleteIdentity(t *testing.T) {
	logicalJobID := "job-1"
	producerScope := "producer-scope"
	producerKey := "producer-key"
	occurrenceScope := "occurrence-scope"
	occurrenceID := "occurrence-id"
	effectScope := "effect-scope"
	effectKey := "effect-key"
	complete := sqlcgen.ReadPostgresJobsAcceptanceRow{
		LogicalJobID: &logicalJobID, ProducerScope: &producerScope, ProducerKey: &producerKey,
		OccurrenceScope: &occurrenceScope, OccurrenceID: &occurrenceID,
		EffectScope: &effectScope, EffectKey: &effectKey,
	}
	want := jobs.AcceptanceIdentity{
		LogicalJobID: jobs.LogicalJobID(logicalJobID), ProducerScope: jobs.ProducerScope(producerScope),
		ProducerKey: jobs.ProducerKey(producerKey), OccurrenceScope: jobs.OccurrenceScope(occurrenceScope),
		OccurrenceID: jobs.OccurrenceID(occurrenceID), EffectScope: jobs.EffectScope(effectScope),
		EffectKey: jobs.EffectKey(effectKey),
	}
	if got, ok := acceptanceIdentityFromReadback(complete); !ok || got != want {
		t.Fatalf("acceptanceIdentityFromReadback(complete) = %+v, %t, want %+v, true", got, ok, want)
	}
	complete.EffectKey = nil
	if got, ok := acceptanceIdentityFromReadback(complete); ok || got != (jobs.AcceptanceIdentity{}) {
		t.Fatalf("acceptanceIdentityFromReadback(incomplete) = %+v, %t, want zero, false", got, ok)
	}
}
