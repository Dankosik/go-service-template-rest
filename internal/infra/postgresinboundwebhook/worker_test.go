// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type memoryStore struct {
	receipt      storedReceipt
	handled      int
	quarantined  int
	failed       int
	handlerCalls int
	failHandled  bool
	failLoad     bool
	failTerminal bool
	terminalOK   bool
}

func (s *memoryStore) Accept(context.Context, receiptRecord) (inboundwebhook.Outcome, error) {
	return inboundwebhook.OutcomeAccepted, nil
}

func (s *memoryStore) loadByID(context.Context, string) (storedReceipt, error) {
	if s.failLoad {
		return storedReceipt{}, errors.New("receipt load failed")
	}
	return s.receipt, nil
}

func (s *memoryStore) MarkHandled(context.Context, string) (bool, error) {
	if s.failHandled {
		return false, errors.New("handled update failed")
	}
	s.handled++
	s.receipt.Outcome = "handled"
	s.receipt.Payload = nil
	return true, nil
}

func (s *memoryStore) MarkQuarantined(_ context.Context, _, _ string) (bool, error) {
	s.quarantined++
	s.receipt.Outcome = "quarantined"
	return true, nil
}

func (s *memoryStore) MarkFailed(context.Context, string) (bool, error) {
	if s.failTerminal && !s.terminalOK {
		s.terminalOK = true
		return false, errors.New("terminal update failed")
	}
	s.failed++
	s.receipt.Outcome = "failed"
	return true, nil
}

func pendingReceipt() storedReceipt {
	return storedReceipt{
		ReceiptID:  "rcpt_1",
		EndpointID: "orders",
		DeliveryID: "msg_123",
		SignedAt:   time.Unix(1700000000, 0).UTC(),
		ReceivedAt: time.Unix(1700000001, 0).UTC(),
		Payload:    []byte(`{"hello":"world"}`),
		Outcome:    "pending",
	}
}

func testRegistry(t *testing.T, handle func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error, decodeErr error) *inboundwebhook.Registry {
	t.Helper()
	reg := inboundwebhook.NewRegistry()
	if err := inboundwebhook.Bind(reg, "orders", func(raw json.RawMessage) (json.RawMessage, error) {
		if decodeErr != nil {
			return nil, decodeErr
		}
		return raw, nil
	}, handle); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestInboundWebhookQuarantine(t *testing.T) {
	t.Parallel()

	store := &memoryStore{receipt: pendingReceipt()}
	var handled int
	worker, err := newWorker(store, testRegistry(t, func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
		handled++
		return nil
	}, inboundwebhook.ErrDecodeRejected), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 25},
	}); err != nil {
		t.Fatal(err)
	}
	if handled != 0 || store.quarantined != 1 || store.receipt.Payload == nil {
		t.Fatalf("handled=%d quarantined=%d payload=%v", handled, store.quarantined, store.receipt.Payload)
	}

	store.receipt.Outcome = "quarantined"
	if err := worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 25},
	}); err != nil || handled != 0 {
		t.Fatalf("terminal replay err=%v handled=%d", err, handled)
	}
}

func TestInboundWebhookHandlerDecodeRejectionRetries(t *testing.T) {
	t.Parallel()

	store := &memoryStore{receipt: pendingReceipt()}
	worker, err := newWorker(store, testRegistry(t, func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
		return fmt.Errorf("handler: %w", inboundwebhook.ErrDecodeRejected)
	}, nil), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	err = worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 25},
	})
	if !errors.Is(err, errHandlerFailed) || store.quarantined != 0 || store.receipt.Outcome != "pending" {
		t.Fatalf("err=%v quarantined=%d outcome=%s", err, store.quarantined, store.receipt.Outcome)
	}
}

func TestInboundWebhookMissingBindingSnoozesAtAttemptLimit(t *testing.T) {
	t.Parallel()

	store := &memoryStore{receipt: pendingReceipt()}
	worker, err := newWorker(store, inboundwebhook.NewRegistry(), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	err = worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 3, MaxAttempts: 3},
	})
	if _, ok := errors.AsType[*rivertype.JobSnoozeError](err); !ok || store.failed != 0 || store.receipt.Outcome != "pending" {
		t.Fatalf("err=%v failed=%d outcome=%s", err, store.failed, store.receipt.Outcome)
	}
}

func TestInboundWebhookStorageFailureSnoozesAtAttemptLimit(t *testing.T) {
	t.Parallel()

	store := &memoryStore{receipt: pendingReceipt(), failLoad: true}
	worker, err := newWorker(store, testRegistry(t, func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
		return nil
	}, nil), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	err = worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 3, MaxAttempts: 3},
	})
	if _, ok := errors.AsType[*rivertype.JobSnoozeError](err); !ok || store.failed != 0 {
		t.Fatalf("err=%v failed=%d", err, store.failed)
	}
}

func TestInboundWebhookHandledLifecycle(t *testing.T) {
	t.Parallel()

	store := &memoryStore{receipt: pendingReceipt()}
	var handled int
	worker, err := newWorker(store, testRegistry(t, func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
		handled++
		return nil
	}, nil), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	job := &river.Job[receiptJobArgs]{Args: receiptJobArgs{ReceiptID: "rcpt_1"}, JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 25}}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if handled != 1 || store.receipt.Payload != nil || store.receipt.Outcome != "handled" {
		t.Fatalf("handled=%d receipt=%+v", handled, store.receipt)
	}
	if err := worker.Work(context.Background(), job); err != nil || handled != 1 {
		t.Fatalf("replay handled=%d err=%v", handled, err)
	}

	lost := &memoryStore{receipt: pendingReceipt(), failHandled: true}
	retrying, err := newWorker(lost, testRegistry(t, func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
		lost.handlerCalls++
		return nil
	}, nil), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := retrying.Work(context.Background(), job); !errors.Is(err, errStorageUnavailable) {
		t.Fatalf("lost handled update err=%v", err)
	}
	lost.failHandled = false
	if err := retrying.Work(context.Background(), job); err != nil || lost.handlerCalls != 2 {
		t.Fatalf("lost update retry calls=%d err=%v", lost.handlerCalls, err)
	}
}

func TestInboundWebhookRetryAndFinalization(t *testing.T) {
	t.Parallel()

	store := &memoryStore{receipt: pendingReceipt()}
	var handled int
	worker, err := newWorker(store, testRegistry(t, func(context.Context, inboundwebhook.VerifiedDelivery, json.RawMessage) error {
		handled++
		return errors.New("provider canary")
	}, nil), newTelemetry(nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	err = worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 3},
	})
	if err == nil || !errors.Is(err, errHandlerFailed) || err.Error() != errHandlerFailed.Error() {
		t.Fatalf("retryable err=%v", err)
	}
	if handled != 1 {
		t.Fatalf("handled=%d", handled)
	}

	store.failTerminal = true
	err = worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 3, MaxAttempts: 3},
	})
	if _, ok := errors.AsType[*rivertype.JobSnoozeError](err); !ok || handled != 1 {
		t.Fatalf("finalizer snooze err=%v handled=%d", err, handled)
	}
	if err := worker.Work(context.Background(), &river.Job[receiptJobArgs]{
		Args:   receiptJobArgs{ReceiptID: "rcpt_1"},
		JobRow: &rivertype.JobRow{Attempt: 3, MaxAttempts: 3},
	}); err != nil || store.failed != 1 || handled != 1 {
		t.Fatalf("finalizer retry err=%v failed=%d handled=%d", err, store.failed, handled)
	}
}

// profile:inbound-webhooks-standard:end
