package natsjs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestPermanent(t *testing.T) {
	t.Parallel()

	want := errors.New("poison")
	err := Permanent(want)
	if !IsPermanent(err) || !errors.Is(err, want) {
		t.Fatalf("Permanent() = %v", err)
	}
	if Permanent(nil) != nil {
		t.Fatal("Permanent(nil) must be nil")
	}
}

// TestWorkerHandleBrokerFaults covers the settlement faults only an in-process
// broker can inject. The ordinary outcomes — ack, retry, dead-letter, oversize,
// and panic — are proved against a real broker in integration_test.go and
// test/nats_messaging_*_integration_test.go.
func TestWorkerHandleBrokerFaults(t *testing.T) {
	t.Parallel()

	poison := Permanent(errors.New("poison"))
	cases := []struct {
		name        string
		cancelRoot  bool
		metadataErr error
		sourceAck   error
		sourceNAK   error
		dlqAck      *jetstream.PubAck
		dlqErr      error
		handlerErr  error
		wantErr     error
		wantNAK     time.Duration
		wantAcks    int
	}{
		{
			name:        "source metadata unavailable",
			metadataErr: errors.New("metadata"),
			wantErr:     ErrTerminal,
		},
		{
			name:       "shutdown leaves the source for redelivery",
			cancelRoot: true,
			handlerErr: errors.New("retry"),
		},
		{
			name:       "retry redelivery refused",
			sourceNAK:  errors.New("redelivery unavailable"),
			handlerErr: errors.New("retry"),
			wantErr:    ErrTerminal,
			wantNAK:    time.Second,
		},
		{
			name:       "dead-letter stream rejects the transfer",
			dlqErr:     nats.ErrNoResponders,
			handlerErr: poison,
			wantErr:    ErrTerminal,
		},
		{
			name:       "dead-letter acknowledgement lost",
			dlqErr:     errors.New("ack unavailable"),
			handlerErr: poison,
			wantNAK:    30 * time.Second,
		},
		{
			name:       "dead-letter acknowledgement lost and redelivery refused",
			sourceNAK:  errors.New("redelivery unavailable"),
			dlqErr:     errors.New("ack unavailable"),
			handlerErr: poison,
			wantErr:    ErrTerminal,
			wantNAK:    30 * time.Second,
		},
		{
			name:       "source acknowledgement lost after transfer and redelivery refused",
			sourceAck:  errors.New("source ack unavailable"),
			sourceNAK:  errors.New("redelivery unavailable"),
			dlqAck:     &jetstream.PubAck{Stream: "EVENTS_DLQ", Sequence: 1},
			handlerErr: poison,
			wantErr:    ErrTerminal,
			wantNAK:    30 * time.Second,
			wantAcks:   1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancelRoot {
				cancel()
			}
			source := unitSource(t, 1)
			source.metadataErr = tc.metadataErr
			source.ackErr = tc.sourceAck
			source.nakErr = tc.sourceNAK
			worker := unitWorker(t, &recordingJetStream{ack: tc.dlqAck, err: tc.dlqErr}, func(context.Context, Message) error {
				return tc.handlerErr
			})

			if err := worker.handle(root, source); !errors.Is(err, tc.wantErr) {
				t.Fatalf("handle() error = %v, want %v", err, tc.wantErr)
			}
			if source.nakDelay != tc.wantNAK || source.ackCount != tc.wantAcks {
				t.Fatalf("source NAK delay = %v, ack count = %d; want %v, %d", source.nakDelay, source.ackCount, tc.wantNAK, tc.wantAcks)
			}
		})
	}
}

func TestHandlerPanicFramesAreSanitized(t *testing.T) {
	const panicCanary = "PANIC_VALUE_CANARY"
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error {
		panic(panicCanary)
	})
	panicked, _ := worker.invokeHandler(t.Context(), Message{})
	if panicked == nil {
		t.Fatal("invokeHandler() reported no panic")
	}
	joined := strings.Join(panicked.frames, "\n")
	if len(panicked.frames) == 0 || !strings.Contains(joined, "TestHandlerPanicFramesAreSanitized") {
		t.Fatalf("panic frames = %q, want test handler location", panicked.frames)
	}
	if strings.Contains(joined, panicCanary) {
		t.Fatalf("panic frames leaked recovered value: %q", panicked.frames)
	}
	// The class is what an operator alerts on, and it is the same vocabulary
	// every other recovery site in this repository publishes.
	if panicked.class != "string" {
		t.Fatalf("panic class = %q, want the class internal/failure assigns a string panic", panicked.class)
	}
}
