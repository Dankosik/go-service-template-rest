package natsjs

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// This file is where an envelope becomes NATS bytes and back: buildNATSMessage
// writes a publication and decodeMessage reads a delivery.
//
// The headers below are the whole wire contract, including the Original-* group
// only the dead-letter transfer writes. They are declared together because they
// are one published contract — renaming any of them is a consumer-visible
// change — while message_deadletter.go owns the transfer that writes and reads
// that second group.
const (
	headerMessageID        = "Message-Id"
	headerEventType        = "Event-Type"
	headerEventSchema      = "Event-Schema"
	headerCreatedAt        = "Created-At"
	headerOriginalSubject  = "Original-Subject"
	headerDeadLetterReason = "Dead-Letter-Reason"
)

func validateEvent(event Event, maxPayloadBytes int) error {
	if !validSubject(event.Subject, false) {
		return fmt.Errorf("%w: invalid event subject", ErrRejected)
	}
	if err := validateRequiredValue("message ID", event.MessageID); err != nil {
		return err
	}
	if err := validateRequiredValue("publication ID", event.PublicationID); err != nil {
		return err
	}
	if err := validateRequiredValue("event type", event.Type); err != nil {
		return err
	}
	if err := validateRequiredValue("event schema", event.Schema); err != nil {
		return err
	}
	_, offset := event.CreatedAt.Zone()
	if event.CreatedAt.IsZero() || offset != 0 {
		return fmt.Errorf("%w: creation time must be non-zero UTC", ErrRejected)
	}
	if len(event.Payload) > maxPayloadBytes {
		return fmt.Errorf("%w: payload exceeds configured maximum", ErrRejected)
	}
	return nil
}

func validateRequiredValue(name, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s is required", ErrRejected, name)
	}
	return validateOptionalValue(name, value)
}

// validateOptionalValue rejects what must not travel in a NATS header.
//
// UTF-8 validity is checked before the scan below rather than left to it.
// Ranging a string yields U+FFFD for each invalid byte, and U+FFFD is not a
// control character — so a value that is not text at all would pass the scan and
// reach the broker, where the consumer that decodes it is the one that fails.
// The durable event and inbox validators reject it at their own boundary for
// the same reason.
//
// Control characters are matched with [unicode.IsControl] rather than an ASCII
// test, so C1 (U+0080-U+009F) is refused alongside C0. Those bytes are as
// unreadable in a subscriber's log line as the ASCII ones, and a header is read
// far more often than it is parsed.
func validateOptionalValue(name, value string) error {
	if len(value) > maxHeaderValueBytes {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrRejected, name, maxHeaderValueBytes)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s must be valid UTF-8", ErrRejected, name)
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return fmt.Errorf("%w: %s contains a control character", ErrRejected, name)
	}
	return nil
}

func buildNATSMessage(ctx context.Context, event Event, maxPayloadBytes int) (*nats.Msg, error) {
	if err := validateEvent(event, maxPayloadBytes); err != nil {
		return nil, err
	}
	header := make(nats.Header)
	header.Set(headerMessageID, event.MessageID)
	header.Set(jetstream.MsgIDHeader, event.PublicationID)
	header.Set(headerEventType, event.Type)
	header.Set(headerEventSchema, event.Schema)
	header.Set(headerCreatedAt, event.CreatedAt.Format(time.RFC3339Nano))
	propagation.TraceContext{}.Inject(ctx, headerCarrier(header))
	msg := &nats.Msg{Subject: event.Subject, Header: header, Data: slices.Clone(event.Payload)}
	if err := validateEncodedMessage(msg, maxPayloadBytes); err != nil {
		return nil, err
	}
	return msg, nil
}

func validateEncodedMessage(msg *nats.Msg, maxPayloadBytes int) error {
	if len(msg.Data) > maxPayloadBytes {
		return fmt.Errorf("%w: payload exceeds configured maximum", ErrRejected)
	}
	headerBytes := encodedHeaderBytes(msg.Header)
	if headerBytes > HeaderLimitBytes {
		return fmt.Errorf("%w: encoded headers exceed %d bytes", ErrRejected, HeaderLimitBytes)
	}
	if msg.Size() > maxPayloadBytes+HeaderLimitBytes {
		return fmt.Errorf("%w: encoded message exceeds configured maximum", ErrRejected)
	}
	return nil
}

func encodedHeaderBytes(header nats.Header) int {
	if len(header) == 0 {
		return 0
	}
	size := len("NATS/1.0\r\n\r\n")
	for key, values := range header {
		for _, value := range values {
			size += len(key) + 2 + len(value) + 2
		}
	}
	return size
}

// remoteContext is the W3C parent extracted but not yet attached to a local context.
type remoteContext struct {
	span trace.SpanContext
}

func decodeMessage(msg jetstream.Msg, metadata *jetstream.MsgMetadata) (Message, remoteContext, error) {
	header := msg.Headers()
	createdAt, err := time.Parse(time.RFC3339Nano, header.Get(headerCreatedAt))
	if err != nil || createdAt.IsZero() {
		return Message{}, remoteContext{}, fmt.Errorf("%w: invalid creation time", ErrRejected)
	}
	// The validated values are what the Message below is built from, rather than
	// a second round of header.Get: a rule added here that rewrites a value —
	// trimming or canonicalizing rather than only rejecting — must reach the
	// delivered message, and re-reading the header would silently skip it.
	messageID := header.Get(headerMessageID)
	publicationID := header.Get(jetstream.MsgIDHeader)
	eventType := header.Get(headerEventType)
	schema := header.Get(headerEventSchema)
	for _, field := range []struct {
		name  string
		value string
	}{
		{headerMessageID, messageID},
		{jetstream.MsgIDHeader, publicationID},
		{headerEventType, eventType},
		{headerEventSchema, schema},
	} {
		if err := validateRequiredValue(field.name, field.value); err != nil {
			return Message{}, remoteContext{}, err
		}
	}
	extracted := propagation.TraceContext{}.Extract(context.Background(), headerCarrier(header))
	remote := remoteContext{span: trace.SpanContextFromContext(extracted)}
	return Message{
		subject:       msg.Subject(),
		messageID:     messageID,
		publicationID: publicationID,
		eventType:     eventType,
		schema:        schema,
		createdAt:     createdAt.UTC(),
		payload:       slices.Clone(msg.Data()),
		metadata: DeliveryMetadata{
			Stream:           metadata.Stream,
			Consumer:         metadata.Consumer,
			StreamSequence:   metadata.Sequence.Stream,
			ConsumerSequence: metadata.Sequence.Consumer,
			NumDelivered:     metadata.NumDelivered,
			NumPending:       metadata.NumPending,
			StoredAt:         metadata.Timestamp.UTC(),
		},
	}, remote, nil
}

// contextWithRemoteParent attaches the message's origin to the handler context.
func contextWithRemoteParent(base context.Context, remote remoteContext) context.Context {
	if remote.span.IsValid() {
		base = trace.ContextWithRemoteSpanContext(base, remote.span)
	}
	return base
}

// wireSize is what one message costs on the wire, which is the bound
// WorkerConfig.MaxDeliveryBytes and the stream's MaxMsgSize are both stated in.
func wireSize(msg jetstream.Msg) int {
	return (&nats.Msg{Subject: msg.Subject(), Header: msg.Headers(), Data: msg.Data()}).Size()
}

type headerCarrier nats.Header

func (h headerCarrier) Get(key string) string { return nats.Header(h).Get(key) }
func (h headerCarrier) Set(key, value string) { nats.Header(h).Set(key, value) }
func (h headerCarrier) Keys() []string {
	return slices.AppendSeq(make([]string, 0, len(h)), maps.Keys(h))
}
