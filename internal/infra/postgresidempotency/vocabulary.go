package postgresidempotency

import "errors"

const (
	vocabularyOther = "other"
	phaseCompleted  = "completed"
	phaseReserved   = "reserved"

	transitionFirstExecution   = "first_execution_started"
	transitionReplay           = "replay_served"
	transitionMismatch         = "mismatch_rejected"
	transitionInProgress       = "in_progress_rejected"
	transitionRollback         = "rollback_released"
	transitionRetry            = "retry_started"
	transitionCommitUnknown    = "commit_unknown"
	transitionCleanupFailed    = "cleanup_failed"
	transitionCleanupRecovered = "cleanup_recovered"

	terminalExecuted      = "executed"
	terminalReplayed      = "replayed"
	terminalMismatch      = "mismatch"
	terminalInProgress    = "in_progress"
	terminalCommitUnknown = "commit_unknown"
	terminalFailed        = "failed"

	stageFirstExecution = "first_execution"
	stageLookup         = "lookup"
	stageReconciliation = "reconciliation"
)

func boundedTransition(value string) string {
	switch value {
	case transitionFirstExecution, transitionReplay, transitionMismatch,
		transitionInProgress, transitionRollback, transitionRetry,
		transitionCommitUnknown, transitionCleanupFailed, transitionCleanupRecovered:
		return value
	default:
		return vocabularyOther
	}
}

func boundedTerminal(value string) string {
	switch value {
	case terminalExecuted, terminalReplayed, terminalMismatch, terminalInProgress,
		terminalCommitUnknown, terminalFailed:
		return value
	default:
		return vocabularyOther
	}
}

func boundedStage(value string) string {
	switch value {
	case stageFirstExecution, stageLookup, stageReconciliation:
		return value
	default:
		return vocabularyOther
	}
}

func boundedErrorClass(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, ErrEpochLost):
		return "epoch_lost"
	case errors.Is(err, ErrIntegrityConflict):
		return "integrity"
	case errors.Is(err, ErrUnavailable):
		return "unavailable"
	case errors.Is(err, ErrConfig):
		return "config"
	default:
		return vocabularyOther
	}
}
