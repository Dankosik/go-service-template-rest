package natsjs

import "testing"

func TestBoundedOutcome(t *testing.T) {
	if got := boundedOutcome(outcomeAccepted); got != outcomeAccepted {
		t.Errorf("boundedOutcome(accepted) = %q", got)
	}
	if got := boundedOutcome("unrecognized"); got != boundedOther {
		t.Errorf("boundedOutcome(unrecognized) = %q, want %q", got, boundedOther)
	}
}
