package postgresidempotency

import (
	"errors"
	"testing"
)

func TestHTTPIdempotencyVocabularyIsClosed(t *testing.T) {
	for _, value := range []string{
		transitionFirstExecution, transitionReplay, transitionMismatch,
		transitionInProgress, transitionRollback, transitionRetry,
		transitionCommitUnknown, transitionCleanupFailed, transitionCleanupRecovered,
	} {
		if got := boundedTransition(value); got != value {
			t.Fatalf("boundedTransition(%q) = %q", value, got)
		}
	}
	for _, value := range []string{terminalExecuted, terminalReplayed, terminalMismatch, terminalInProgress, terminalCommitUnknown, terminalFailed} {
		if got := boundedTerminal(value); got != value {
			t.Fatalf("boundedTerminal(%q) = %q", value, got)
		}
	}
	for _, value := range []string{stageFirstExecution, stageLookup, stageReconciliation} {
		if got := boundedStage(value); got != value {
			t.Fatalf("boundedStage(%q) = %q", value, got)
		}
	}
	if boundedTransition("sentinel") != vocabularyOther || boundedTerminal("sentinel") != vocabularyOther || boundedStage("sentinel") != vocabularyOther {
		t.Fatal("unknown telemetry vocabulary did not collapse to other")
	}
	if boundedErrorClass(errors.New("sentinel")) != vocabularyOther {
		t.Fatal("unknown error class did not collapse to other")
	}
}
