package natsjs

import "testing"

func TestBoundedOutcome(t *testing.T) {
	for _, outcome := range []string{
		outcomeAccepted, outcomeRejected, outcomeAmbiguous,
		outcomeSuccess, outcomeTerminal, outcomePermanent, outcomeCanceled,
		outcomeTimeout, outcomeRetryable,
	} {
		if got := boundedOutcome(outcome); got != outcome {
			t.Errorf("boundedOutcome(%q) = %q", outcome, got)
		}
	}
	if got := boundedOutcome("unrecognized"); got != boundedOther {
		t.Errorf("boundedOutcome(unrecognized) = %q, want %q", got, boundedOther)
	}
}
