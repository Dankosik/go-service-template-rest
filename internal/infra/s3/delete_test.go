package s3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"
	"testing"

	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestDeleteAfterPossibleSendIsOutcomeUnknown(t *testing.T) {
	client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("private transport failure")
	})
	client.transport.base = httpDoerFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", request.Method)
		}
		httptrace.ContextClientTrace(request.Context()).WroteHeaders()
		return nil, errors.New("private transport failure")
	})
	if got := objectstorage.Kind(client.Delete(context.Background(), "object")); got != objectstorage.KindOutcomeUnknown {
		t.Fatalf("Kind(Delete()) = %q, want %q", got, objectstorage.KindOutcomeUnknown)
	}
}
