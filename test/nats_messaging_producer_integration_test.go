//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func TestNATSProducerOutcomes(t *testing.T) {
	f := newNATSFixture(t)
	client := f.client(t, natsjs.RoleProducer)
	event := testEvent("accepted")
	first, err := client.Producer().Publish(t.Context(), event)
	if err != nil {
		t.Fatalf("Publish(first) error = %v", err)
	}
	duplicate, err := client.Producer().Publish(t.Context(), event)
	if err != nil {
		t.Fatalf("Publish(duplicate) error = %v", err)
	}
	if !duplicate.Duplicate || duplicate.Sequence != first.Sequence {
		t.Fatalf("duplicate ack = %+v, first = %+v", duplicate, first)
	}

	rejected := testEvent("wrong stream")
	rejected.Subject = "other.test"
	if _, err := client.Producer().Publish(t.Context(), rejected); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Publish(stream mismatch) error = %v, want ErrRejected", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.Producer().Publish(canceled, testEvent("canceled")); !errors.Is(err, natsjs.ErrRejected) {
		t.Fatalf("Publish(pre-canceled) error = %v, want ErrRejected", err)
	}
}
