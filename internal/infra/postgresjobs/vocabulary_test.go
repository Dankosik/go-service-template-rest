package postgresjobs

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestVocabularyFallsBackForUnknownValues(t *testing.T) {
	if got := metricEvent("sentinel"); got != metricOutcomeOther {
		t.Fatalf("metricEvent() = %q, want %q", got, metricOutcomeOther)
	}
	if got := metricOutcome(jobs.OutcomeClass("sentinel")); got != metricOutcomeOther {
		t.Fatalf("metricOutcome() = %q, want %q", got, metricOutcomeOther)
	}
	if got := metricOperation("sentinel"); got != metricOperationOther {
		t.Fatalf("metricOperation() = %q, want %q", got, metricOperationOther)
	}
	if got := metricState(jobs.State("sentinel")); got != metricStateOther {
		t.Fatalf("metricState() = %q, want %q", got, metricStateOther)
	}
}
