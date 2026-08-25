package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrackResponseCommitRequiresFinalStatus(t *testing.T) {
	t.Parallel()

	t.Run("informational status", func(t *testing.T) {
		t.Parallel()

		writer, committed := trackResponseCommit(httptest.NewRecorder())

		writer.WriteHeader(http.StatusEarlyHints)
		if committed() {
			t.Fatal("103 Early Hints marked the response committed")
		}

		writer.WriteHeader(http.StatusNoContent)
		if !committed() {
			t.Fatal("204 No Content did not mark the response committed")
		}
	})

	t.Run("switching protocols", func(t *testing.T) {
		t.Parallel()

		writer, committed := trackResponseCommit(httptest.NewRecorder())

		writer.WriteHeader(http.StatusSwitchingProtocols)
		if !committed() {
			t.Fatal("101 Switching Protocols did not mark the response committed")
		}
	})

	t.Run("rejected status", func(t *testing.T) {
		t.Parallel()

		writer, committed := trackResponseCommit(httptest.NewRecorder())

		defer func() {
			if recover() == nil {
				t.Fatal("WriteHeader(99) did not panic")
			}
			if committed() {
				t.Fatal("rejected status marked the response committed")
			}
		}()
		writer.WriteHeader(99)
	})
}
