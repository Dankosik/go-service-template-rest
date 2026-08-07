package natsjs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestWorkerHandleOutcomes(t *testing.T) {
	t.Run("success and ambiguous ack", func(t *testing.T) {
		source := unitSource(t, 1)
		worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
		if err := worker.handle(t.Context(), source); err != nil || source.ackCount != 1 {
			t.Fatalf("handle(success) error = %v, ack count = %d", err, source.ackCount)
		}
		source = unitSource(t, 1)
		source.ackErr = errors.New("ack vanished")
		if err := worker.handle(t.Context(), source); err != nil {
			t.Fatalf("handle(ambiguous ack) error = %v", err)
		}
	})

	t.Run("retry and shutdown", func(t *testing.T) {
		source := unitSource(t, 1)
		worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return errors.New("retry") })
		if err := worker.handle(t.Context(), source); err != nil || source.nakDelay != worker.cfg.RetryDelays[0] {
			t.Fatalf("handle(retry) error = %v, delay = %v", err, source.nakDelay)
		}
		source = unitSource(t, 1)
		source.nakErr = errors.New("redelivery unavailable")
		if err := worker.handle(t.Context(), source); !errors.Is(err, ErrTerminal) {
			t.Fatalf("handle(retry NAK failure) error = %v, want ErrTerminal", err)
		}
		canceled, cancel := context.WithCancel(t.Context())
		cancel()
		source = unitSource(t, 1)
		if err := worker.handle(canceled, source); err != nil || source.nakDelay != 0 {
			t.Fatalf("handle(shutdown) error = %v, delay = %v", err, source.nakDelay)
		}
	})

	t.Run("permanent malformed and exhausted dead-letter", func(t *testing.T) {
		for name, source := range map[string]*fakeMsg{
			"permanent": unitSource(t, 1),
			"malformed": unitSource(t, 1),
			"exhausted": unitSource(t, 6),
		} {
			t.Run(name, func(t *testing.T) {
				broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS_DLQ", Sequence: 1}}
				handler := func(context.Context, Message) error { return Permanent(errors.New("poison")) }
				if name == "malformed" {
					source.header.Del(headerEventType)
					handler = func(context.Context, Message) error { t.Fatal("malformed message reached handler"); return nil }
				}
				if name == "exhausted" {
					handler = func(context.Context, Message) error { t.Fatal("exhausted message reached handler"); return nil }
				}
				worker := unitWorker(t, broker, handler)
				if err := worker.handle(t.Context(), source); err != nil || broker.published == nil || source.ackCount != 1 {
					t.Fatalf("handle(%s) error = %v, published = %#v, ack count = %d", name, err, broker.published, source.ackCount)
				}
			})
		}
	})

	t.Run("terminal inputs", func(t *testing.T) {
		worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { panic("canary") })
		if err := worker.handle(t.Context(), unitSource(t, 1)); !errors.Is(err, ErrTerminal) || strings.Contains(err.Error(), "canary") {
			t.Fatalf("handle(panic) error = %v", err)
		}
		metadataFailure := unitSource(t, 1)
		metadataFailure.metadataErr = errors.New("metadata")
		if err := worker.handle(t.Context(), metadataFailure); !errors.Is(err, ErrTerminal) {
			t.Fatalf("handle(metadata failure) error = %v", err)
		}
		oversized := unitSource(t, 1)
		oversized.data = make([]byte, testMaxPayloadBytes+1)
		if err := worker.handle(t.Context(), oversized); !errors.Is(err, ErrTerminal) {
			t.Fatalf("handle(oversized) error = %v", err)
		}
		oversizedHeaders := unitSource(t, 1)
		oversizedHeaders.header.Set("X-Oversized", strings.Repeat("x", HeaderLimitBytes))
		if err := worker.handle(t.Context(), oversizedHeaders); err != nil {
			t.Fatalf("handle(oversized headers) error = %v", err)
		}
		if oversizedHeaders.ackCount != 1 {
			t.Fatalf("oversized header source ack count = %d, want confirmed DLQ handoff", oversizedHeaders.ackCount)
		}
	})

	t.Run("dead-letter failures", func(t *testing.T) {
		for name, brokerErr := range map[string]error{
			"rejected":  nats.ErrNoResponders,
			"ambiguous": errors.New("ack unavailable"),
		} {
			t.Run(name, func(t *testing.T) {
				source := unitSource(t, 1)
				worker := unitWorker(t, &recordingJetStream{err: brokerErr}, func(context.Context, Message) error {
					return Permanent(errors.New("poison"))
				})
				err := worker.handle(t.Context(), source)
				if name == "rejected" && !errors.Is(err, ErrTerminal) {
					t.Fatalf("handle(rejected DLQ) error = %v", err)
				}
				if name == "ambiguous" && (err != nil || source.nakDelay != worker.cfg.DeadLetterRetryDelay) {
					t.Fatalf("handle(ambiguous DLQ) error = %v, delay = %v", err, source.nakDelay)
				}
				if source.ackCount != 0 {
					t.Fatalf("handle(%s DLQ) source ack count = %d, want 0", name, source.ackCount)
				}
			})
		}
		source := unitSource(t, 1)
		source.nakErr = errors.New("redelivery unavailable")
		worker := unitWorker(t, &recordingJetStream{err: errors.New("ack unavailable")}, func(context.Context, Message) error {
			return Permanent(errors.New("poison"))
		})
		if err := worker.handle(t.Context(), source); !errors.Is(err, ErrTerminal) {
			t.Fatalf("handle(ambiguous DLQ NAK failure) error = %v, want ErrTerminal", err)
		}

		source = unitSource(t, 1)
		source.ackErr = errors.New("source ack unavailable")
		worker = unitWorker(t, &recordingJetStream{ack: &jetstream.PubAck{}}, func(context.Context, Message) error {
			return Permanent(errors.New("poison"))
		})
		if err := worker.handle(t.Context(), source); err != nil || source.nakDelay != worker.cfg.DeadLetterRetryDelay {
			t.Fatalf("handle(ambiguous source ack) error = %v, delay = %v", err, source.nakDelay)
		}
		source = unitSource(t, 1)
		source.ackErr = errors.New("source ack unavailable")
		source.nakErr = errors.New("redelivery unavailable")
		if err := worker.handle(t.Context(), source); !errors.Is(err, ErrTerminal) {
			t.Fatalf("handle(ambiguous source ACK NAK failure) error = %v, want ErrTerminal", err)
		}
	})
}

func TestHandlerPanicFramesAreSanitized(t *testing.T) {
	const panicCanary = "PANIC_VALUE_CANARY"
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error {
		panic(panicCanary)
	})
	frames, _ := worker.invokeHandler(t.Context(), Message{})
	joined := strings.Join(frames, "\n")
	if len(frames) == 0 || !strings.Contains(joined, "TestHandlerPanicFramesAreSanitized") {
		t.Fatalf("panic frames = %q, want test handler location", frames)
	}
	if strings.Contains(joined, panicCanary) {
		t.Fatalf("panic frames leaked recovered value: %q", frames)
	}
}
