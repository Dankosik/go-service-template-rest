package postgresjobs

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestVocabularyFallsBackForUnknownValues(t *testing.T) {
	if got := metricOutcome(jobs.OutcomeClass("sentinel")); got != metricOutcomeOther {
		t.Fatalf("metricOutcome() = %q, want %q", got, metricOutcomeOther)
	}
	if got := metricState(jobs.State("sentinel")); got != metricStateOther {
		t.Fatalf("metricState() = %q, want %q", got, metricStateOther)
	}
}
