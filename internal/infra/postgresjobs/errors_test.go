package postgresjobs

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorsKeepStableIdentity(t *testing.T) {
	for _, sentinel := range []error{
		ErrConfig,
		ErrSchemaIncompatible,
		ErrOperationTimeout,
		ErrSessionTerminal,
		ErrUnknownVocabulary,
	} {
		if !errors.Is(fmt.Errorf("wrapped: %w", sentinel), sentinel) {
			t.Fatalf("wrapped %v lost identity", sentinel)
		}
	}
}
