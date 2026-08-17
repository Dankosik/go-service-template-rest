package postgresidempotency

import (
	"testing"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
)

func TestEpochLossMapsToUnavailable(t *testing.T) {
	t.Parallel()
	decision, handled := decisionForClassificationError(ErrEpochLost)
	if !handled || decision.Outcome != httpidempotency.OutcomeUnavailable {
		t.Fatalf("epoch-loss decision = (%v, %t), want unavailable/true", decision.Outcome, handled)
	}
}
