package natsjs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/trace"
)

func TestMessageIsImmutable(t *testing.T) {
	msg := &fakeMsg{
		subject: "events.created",
		header:  eventHeaders(validTestEvent()),
		data:    []byte("payload"),
		metadata: &jetstream.MsgMetadata{
			Sequence:     jetstream.SequencePair{Stream: 3, Consumer: 2},
			NumDelivered: 2,
			NumPending:   4,
			Timestamp:    time.Unix(100, 0).UTC(),
			Stream:       "EVENTS",
			Consumer:     "events-worker",
		},
	}
	decoded, _, err := decodeMessage(msg, msg.metadata)
	if err != nil {
		t.Fatalf("decodeMessage() error = %v", err)
	}
	msg.data[0] = 'X'
	first := decoded.Payload()
	first[0] = 'Y'
	if got := string(decoded.Payload()); got != "payload" {
		t.Fatalf("decoded payload mutated through alias: %q", got)
	}
	if decoded.MessageID() != "message-1" || decoded.PublicationID() != "publication-1" || decoded.Type() != "created" || decoded.Schema() != "v1" ||
		!decoded.CreatedAt().Equal(validTestEvent().CreatedAt) {
		t.Fatalf("decoded accessors returned inconsistent envelope: message=%q publication=%q type=%q schema=%q created=%v",
			decoded.MessageID(), decoded.PublicationID(), decoded.Type(), decoded.Schema(), decoded.CreatedAt())
	}
}

func TestRemoteTraceMetadataPreservesHandlerContext(t *testing.T) {
	t.Parallel()

	const remoteTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	localSpan := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{2},
	})
	for _, testCase := range []struct {
		name        string
		header      string
		wantTraceID string
		wantRemote  bool
	}{
		{name: "missing", wantTraceID: localSpan.TraceID().String()},
		{name: "invalid", header: "invalid", wantTraceID: localSpan.TraceID().String()},
		{name: "remote", header: "00-" + remoteTraceID + "-00f067aa0ba902b7-01", wantTraceID: remoteTraceID, wantRemote: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			type valueKey struct{}
			root, cancel := context.WithTimeout(context.WithValue(t.Context(), valueKey{}, "retained"), time.Minute)
			defer cancel()
			root = trace.ContextWithSpanContext(root, localSpan)
			source := unitSource(t, 1)
			if testCase.header != "" {
				source.header.Set("traceparent", testCase.header)
			}
			_, remote, err := decodeMessage(source, source.metadata)
			if err != nil {
				t.Fatalf("decodeMessage() error = %v", err)
			}
			if remote.span.IsValid() != testCase.wantRemote {
				t.Fatalf("decoded remote span validity = %t, want %t", remote.span.IsValid(), testCase.wantRemote)
			}
			linked := contextWithRemoteParent(root, remote)
			span := trace.SpanContextFromContext(linked)
			if span.TraceID().String() != testCase.wantTraceID || span.IsRemote() != testCase.wantRemote {
				t.Fatalf("linked span = %v, want trace %s remote=%t", span, testCase.wantTraceID, testCase.wantRemote)
			}
			if got := linked.Value(valueKey{}); got != "retained" {
				t.Fatalf("linked context value = %v, want retained", got)
			}
			wantDeadline, _ := root.Deadline()
			if got, ok := linked.Deadline(); !ok || !got.Equal(wantDeadline) {
				t.Fatalf("linked deadline = %v, %t; want %v", got, ok, wantDeadline)
			}
			cancel()
			if !errors.Is(linked.Err(), context.Canceled) {
				t.Fatalf("linked context error = %v, want cancellation", linked.Err())
			}
		})
	}
}
