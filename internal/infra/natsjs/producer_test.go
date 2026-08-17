package natsjs

import (
	"context"
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
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

func TestProducerAdmissionAndCopy(t *testing.T) {
	p := newProducer(nil, 1, testMaxPayloadBytes)
	if err := p.begin(); err != nil {
		t.Fatalf("begin(first) error = %v", err)
	}
	if err := p.begin(); !errors.Is(err, ErrCapacity) {
		t.Fatalf("begin(over capacity) error = %v, want ErrCapacity", err)
	}
	p.end()
	p.stop()
	if err := p.begin(); !errors.Is(err, ErrDraining) {
		t.Fatalf("begin(after stop) error = %v, want ErrDraining", err)
	}

	event := validTestEvent()
	ctx := reqctx.ContextWithRequestID(context.Background(), "request-1")
	msg, err := buildNATSMessage(ctx, event, testMaxPayloadBytes)
	if err != nil {
		t.Fatalf("buildNATSMessage() error = %v", err)
	}
	event.Payload[0] = 'X'
	if string(msg.Data) != "payload" {
		t.Fatalf("message payload aliased caller memory: %q", msg.Data)
	}
	if got := msg.Header.Get(headerCorrelationID); got != "request-1" {
		t.Fatalf("Correlation-Id = %q, want request-1", got)
	}
}
