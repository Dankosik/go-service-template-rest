package natsjs

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestProducerPublishAndLifecycleOutcomes(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		broker := &recordingJetStream{ack: &jetstream.PubAck{Stream: "EVENTS", Sequence: 7, Duplicate: true}}
		client := unitClient(t, broker, RoleProducer)
		result, err := client.Producer().Publish(t.Context(), validTestEvent())
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
		if result.Stream != "EVENTS" || result.Sequence != 7 || !result.Duplicate || broker.published == nil {
			t.Fatalf("Publish() result = %+v, message = %#v", result, broker.published)
		}
	})

	t.Run("rejections", func(t *testing.T) {
		client := unitClient(t, &recordingJetStream{}, RoleProducer)
		invalid := validTestEvent()
		invalid.Subject = "events.*"
		if _, err := client.Producer().Publish(t.Context(), invalid); !errors.Is(err, ErrRejected) {
			t.Fatalf("Publish(invalid) error = %v, want ErrRejected", err)
		}
		canceled, cancel := context.WithCancel(t.Context())
		cancel()
		if _, err := client.Producer().Publish(canceled, validTestEvent()); !errors.Is(err, ErrRejected) {
			t.Fatalf("Publish(canceled) error = %v, want ErrRejected", err)
		}
		if err := client.producer.begin(); err != nil {
			t.Fatalf("begin() error = %v", err)
		}
		if _, err := client.Producer().Publish(t.Context(), validTestEvent()); !errors.Is(err, ErrCapacity) {
			t.Fatalf("Publish(at capacity) error = %v, want ErrCapacity", err)
		}
		client.producer.end()
		client.StopPublish()
		if _, err := client.Producer().Publish(t.Context(), validTestEvent()); !errors.Is(err, ErrDraining) {
			t.Fatalf("Publish(during drain) error = %v, want ErrDraining", err)
		}
	})

	for name, tc := range map[string]struct {
		brokerErr error
		want      error
	}{
		"broker rejection": {brokerErr: nats.ErrNoResponders, want: ErrRejected},
		"ambiguous ack":    {brokerErr: errors.New("connection vanished"), want: ErrAmbiguous},
	} {
		t.Run(name, func(t *testing.T) {
			client := unitClient(t, &recordingJetStream{err: tc.brokerErr}, RoleProducer)
			_, err := client.Producer().Publish(t.Context(), validTestEvent())
			if !errors.Is(err, tc.want) {
				t.Fatalf("Publish() error = %v, want %v", err, tc.want)
			}
		})
	}

	t.Run("wait", func(t *testing.T) {
		client := unitClient(t, &recordingJetStream{}, RoleProducer)
		if err := client.producer.begin(); err != nil {
			t.Fatalf("begin() error = %v", err)
		}
		canceled, cancel := context.WithCancel(t.Context())
		cancel()
		if err := client.producer.wait(canceled); err == nil {
			t.Fatal("wait(canceled) error = nil")
		}
		client.producer.end()
		if err := client.producer.wait(t.Context()); err != nil {
			t.Fatalf("wait(idle) error = %v", err)
		}
		var producer *Producer
		producer.stop()
		if err := producer.wait(t.Context()); err != nil {
			t.Fatalf("nil producer wait error = %v", err)
		}
	})
}

func TestClientStateTransitions(t *testing.T) {
	client := unitClient(t, &recordingJetStream{}, RoleProducer)
	client.ready.Store(true)
	if client.Name() != "messaging" || client.Producer() == nil || !client.Ready() {
		t.Fatal("client accessors did not expose admitted producer state")
	}
	client.draining.Store(true)
	if client.Ready() {
		t.Fatal("draining client remained ready")
	}
	client.draining.Store(false)
	if err := client.Check(t.Context()); err == nil || client.Ready() {
		t.Fatalf("Check(without connection) error = %v, ready = %t", err, client.Ready())
	}
	var nilClient *Client
	if err := nilClient.Check(t.Context()); err == nil {
		t.Fatal("nil Client.Check() error = nil")
	}
	if err := nilClient.Shutdown(t.Context()); err != nil {
		t.Fatalf("nil Client.Shutdown() error = %v", err)
	}
	nilClient.StopPublish()
	nilClient.Close()

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := client.Run(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run(canceled) error = %v", err)
	}
	want := errors.New("terminal")
	client.signalTerminal(want)
	client.signalTerminal(errors.New("ignored duplicate"))
	if err := client.Run(t.Context()); !errors.Is(err, want) {
		t.Fatalf("Run(terminal) error = %v", err)
	}
	if client.waitForReconnect(t.Context(), errors.New("failure")) {
		t.Fatal("client without connection reported reconnect")
	}
	if err := client.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown(without connection) error = %v", err)
	}
	client.Close()

	expired, expiredCancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer expiredCancel()
	if got := boundedTimeout(expired); got != time.Nanosecond {
		t.Fatalf("boundedTimeout(expired) = %v", got)
	}
	short, shortCancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer shortCancel()
	if got := boundedTimeout(short); got <= 0 || got > 10*time.Millisecond {
		t.Fatalf("boundedTimeout(short) = %v", got)
	}
	if got := boundedTimeout(t.Context()); got != operationTimeout {
		t.Fatalf("boundedTimeout(no deadline) = %v", got)
	}
}

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

func TestWorkerShutdownStateTransitions(t *testing.T) {
	worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	close(worker.runDone)
	if err := worker.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown(graceful) error = %v", err)
	}

	forced := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := forced.Shutdown(canceled); !errors.Is(err, context.Canceled) || !forced.client.draining.Load() {
		t.Fatalf("Shutdown(forced) error = %v, draining = %t", err, forced.client.draining.Load())
	}
}

func TestWorkerShutdownWaitsForRunCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := unitWorker(t, &recordingJetStream{}, func(context.Context, Message) error { return nil })
		result := make(chan error, 1)
		go func() { result <- worker.Shutdown(t.Context()) }()
		synctest.Wait()
		select {
		case err := <-result:
			t.Fatalf("Shutdown() returned before Run completion: %v", err)
		default:
		}
		close(worker.runDone)
		synctest.Wait()
		if err := <-result; err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
}

type recordingJetStream struct {
	jetstream.JetStream

	ack       *jetstream.PubAck
	err       error
	published *nats.Msg
}

func (s *recordingJetStream) PublishMsg(_ context.Context, msg *nats.Msg, _ ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	s.published = &nats.Msg{Subject: msg.Subject, Header: maps.Clone(msg.Header), Data: append([]byte(nil), msg.Data...)}
	return s.ack, s.err
}

func unitClient(t *testing.T, broker jetstream.JetStream, role Role) *Client {
	t.Helper()
	sig, err := newSignals(Observability{Logger: slog.New(slog.DiscardHandler)}, role, func() bool { return false })
	if err != nil {
		t.Fatalf("newSignals() error = %v", err)
	}
	t.Cleanup(sig.close)
	cfg := testConfig()
	cfg.Stream = "EVENTS"
	client := &Client{
		cfg:         cfg,
		js:          broker,
		signals:     sig,
		events:      make(chan connectionEvent, 1),
		reconnected: make(chan struct{}, 1),
		terminal:    make(chan error, 1),
		closed:      make(chan struct{}),
	}
	client.producer = newProducer(client, 1, cfg.MaxPayloadBytes)
	return client
}

func unitWorker(t *testing.T, broker jetstream.JetStream, handler Handler) *Worker {
	t.Helper()
	client := unitClient(t, broker, RoleWorker)
	cfg := testWorkerConfig()
	cfg.Consumer = "events-worker"
	cfg.FilterSubject = "events.>"
	cfg.DeadLetterSubject = "dead.events"
	return &Worker{
		client:    client,
		cfg:       cfg,
		dlqStream: "EVENTS_DLQ",
		handler:   handler,
		fatal:     make(chan error, 1),
		runDone:   make(chan struct{}),
	}
}

func unitSource(t *testing.T, delivered uint64) *fakeMsg {
	t.Helper()
	msg, err := buildNATSMessage(t.Context(), validTestEvent(), testMaxPayloadBytes)
	if err != nil {
		t.Fatalf("buildNATSMessage() error = %v", err)
	}
	return &fakeMsg{
		subject: msg.Subject,
		header:  msg.Header,
		data:    msg.Data,
		metadata: &jetstream.MsgMetadata{
			Sequence:     jetstream.SequencePair{Stream: 3, Consumer: 2},
			NumDelivered: delivered,
			Timestamp:    time.Unix(100, 0).UTC(),
			Stream:       "EVENTS",
			Consumer:     "events-worker",
		},
	}
}
