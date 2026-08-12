package postgresidempotency

import (
	"fmt"
	"testing"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestReconcileClassificationErrorDecisions(t *testing.T) {
	decision, handled := decisionForClassificationError(fmt.Errorf("wrapped: %w", ErrIntegrityConflict))
	if !handled || decision.Outcome != httpidempotency.OutcomeIntegrityConflict {
		t.Fatalf("integrity error decision = (%v, %t), want integrity/true", decision.Outcome, handled)
	}
}
