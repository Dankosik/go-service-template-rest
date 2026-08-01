package natsjs

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	headerMessageID             = "Message-Id"
	headerPublicationID         = "Publication-Id"
	headerEventType             = "Event-Type"
	headerEventSchema           = "Event-Schema"
	headerOrderingKey           = "Ordering-Key"
	headerCreatedAt             = "Created-At"
	headerCorrelationID         = "Correlation-Id"
	headerOriginalSubject       = "Original-Subject"
	headerOriginalStream        = "Original-Stream"
	headerOriginalConsumer      = "Original-Consumer"
	headerOriginalStreamSeq     = "Original-Stream-Sequence"
	headerOriginalConsumerSeq   = "Original-Consumer-Sequence"
	headerOriginalNumDelivered  = "Original-Num-Delivered"
	headerOriginalStoredAt      = "Original-Stored-At"
	headerOriginalPublicationID = "Original-Publication-Id"
	headerDeadLetterReason      = "Dead-Letter-Reason"
)

type Event struct {
	Subject       string
	MessageID     string
	PublicationID string
	Type          string
	Schema        string
	OrderingKey   string
	CreatedAt     time.Time
	Payload       []byte
}

type PublishResult struct {
	Stream    string
	Sequence  uint64
	Duplicate bool
}

type DeliveryMetadata struct {
	Stream, Consumer                 string
	StreamSequence, ConsumerSequence uint64
	NumDelivered, NumPending         uint64
	StoredAt                         time.Time
	SourceSubject                    string
	SourcePublicationID              string
}

type Message struct {
	subject       string
	messageID     string
	publicationID string
	eventType     string
	schema        string
	orderingKey   string
	correlationID string
	createdAt     time.Time
	payload       []byte
	metadata      DeliveryMetadata
}

func (m Message) Subject() string            { return m.subject }
func (m Message) MessageID() string          { return m.messageID }
func (m Message) PublicationID() string      { return m.publicationID }
func (m Message) Type() string               { return m.eventType }
func (m Message) Schema() string             { return m.schema }
func (m Message) OrderingKey() string        { return m.orderingKey }
func (m Message) CorrelationID() string      { return m.correlationID }
func (m Message) CreatedAt() time.Time       { return m.createdAt }
func (m Message) Payload() []byte            { return slices.Clone(m.payload) }
func (m Message) Metadata() DeliveryMetadata { return m.metadata }

type Handler func(context.Context, Message) error

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

func NewID() string { return rand.Text() }

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
	if err := validateOptionalValue("ordering key", event.OrderingKey); err != nil {
		return err
	}
	if event.CreatedAt.IsZero() || event.CreatedAt.Location() != time.UTC {
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

func validateOptionalValue(name, value string) error {
	if len(value) > 256 {
		return fmt.Errorf("%w: %s exceeds 256 bytes", ErrRejected, name)
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("%w: %s contains a control character", ErrRejected, name)
		}
	}
	return nil
}

func buildNATSMessage(ctx context.Context, event Event, maxPayloadBytes int) (*nats.Msg, error) {
	if err := validateEvent(event, maxPayloadBytes); err != nil {
		return nil, err
	}
	header := make(nats.Header)
	header.Set(headerMessageID, event.MessageID)
	header.Set(headerPublicationID, event.PublicationID)
	header.Set(headerEventType, event.Type)
	header.Set(headerEventSchema, event.Schema)
	header.Set(headerCreatedAt, event.CreatedAt.Format(time.RFC3339Nano))
	if event.OrderingKey != "" {
		header.Set(headerOrderingKey, event.OrderingKey)
	}
	if correlationID := reqctx.RequestID(ctx); reqctx.ValidRequestID(correlationID) {
		header.Set(headerCorrelationID, correlationID)
	}
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

func decodeMessage(msg jetstream.Msg, metadata *jetstream.MsgMetadata) (context.Context, Message, error) {
	header := msg.Headers()
	createdAt, err := time.Parse(time.RFC3339Nano, header.Get(headerCreatedAt))
	if err != nil || createdAt.IsZero() {
		return context.Background(), Message{}, fmt.Errorf("invalid creation time")
	}
	values := []struct {
		name  string
		value string
	}{
		{headerMessageID, header.Get(headerMessageID)},
		{headerPublicationID, header.Get(headerPublicationID)},
		{headerEventType, header.Get(headerEventType)},
		{headerEventSchema, header.Get(headerEventSchema)},
	}
	for _, field := range values {
		if err := validateRequiredValue(field.name, field.value); err != nil {
			return context.Background(), Message{}, err
		}
	}
	if err := validateOptionalValue(headerOrderingKey, header.Get(headerOrderingKey)); err != nil {
		return context.Background(), Message{}, err
	}
	correlationID := header.Get(headerCorrelationID)
	if correlationID != "" && !reqctx.ValidRequestID(correlationID) {
		return context.Background(), Message{}, fmt.Errorf("invalid correlation ID")
	}
	ctx := propagation.TraceContext{}.Extract(context.Background(), headerCarrier(header))
	if correlationID != "" {
		ctx = reqctx.ContextWithRequestID(ctx, correlationID)
	}
	return ctx, Message{
		subject:       msg.Subject(),
		messageID:     header.Get(headerMessageID),
		publicationID: header.Get(headerPublicationID),
		eventType:     header.Get(headerEventType),
		schema:        header.Get(headerEventSchema),
		orderingKey:   header.Get(headerOrderingKey),
		correlationID: correlationID,
		createdAt:     createdAt.UTC(),
		payload:       slices.Clone(msg.Data()),
		metadata: DeliveryMetadata{
			Stream:              metadata.Stream,
			Consumer:            metadata.Consumer,
			StreamSequence:      metadata.Sequence.Stream,
			ConsumerSequence:    metadata.Sequence.Consumer,
			NumDelivered:        metadata.NumDelivered,
			NumPending:          metadata.NumPending,
			StoredAt:            metadata.Timestamp.UTC(),
			SourceSubject:       msg.Subject(),
			SourcePublicationID: header.Get(headerPublicationID),
		},
	}, nil
}

func contextWithRemoteParent(base, extracted context.Context) context.Context {
	spanContext := trace.SpanContextFromContext(extracted)
	if spanContext.IsValid() {
		base = trace.ContextWithRemoteSpanContext(base, spanContext)
	}
	if correlationID := reqctx.RequestID(extracted); correlationID != "" {
		base = reqctx.ContextWithRequestID(base, correlationID)
	}
	return base
}

func deadLetterMessage(source jetstream.Msg, metadata *jetstream.MsgMetadata, decoded Message, reason string) (*nats.Msg, string) {
	header := make(nats.Header)
	for _, name := range []string{
		headerMessageID, headerPublicationID, headerEventType, headerEventSchema,
		headerOrderingKey, headerCreatedAt, headerCorrelationID,
		"traceparent", "tracestate",
	} {
		if value := source.Headers().Get(name); value != "" {
			header.Set(name, value)
		}
	}
	header.Set(headerOriginalSubject, source.Subject())
	header.Set(headerOriginalStream, metadata.Stream)
	header.Set(headerOriginalConsumer, metadata.Consumer)
	header.Set(headerOriginalStreamSeq, strconv.FormatUint(metadata.Sequence.Stream, 10))
	header.Set(headerOriginalConsumerSeq, strconv.FormatUint(metadata.Sequence.Consumer, 10))
	header.Set(headerOriginalNumDelivered, strconv.FormatUint(metadata.NumDelivered, 10))
	header.Set(headerOriginalStoredAt, metadata.Timestamp.UTC().Format(time.RFC3339Nano))
	header.Set(headerOriginalPublicationID, source.Headers().Get(headerPublicationID))
	header.Set(headerDeadLetterReason, reason)
	identity := strings.Join([]string{
		metadata.Stream,
		strconv.FormatUint(metadata.Sequence.Stream, 10),
		metadata.Timestamp.UTC().Format(time.RFC3339Nano),
		source.Headers().Get(headerPublicationID),
	}, "\x00")
	digest := sha256.Sum256([]byte(identity))
	transferID := "dlq-" + hex.EncodeToString(digest[:])
	header.Set(headerPublicationID, transferID)
	if header.Get(headerMessageID) == "" {
		header.Set(headerMessageID, transferID)
	}
	if decoded.messageID != "" {
		header.Set(headerMessageID, decoded.messageID)
	}
	return &nats.Msg{Header: header, Data: slices.Clone(source.Data())}, transferID
}

func isPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

type headerCarrier nats.Header

func (h headerCarrier) Get(key string) string { return nats.Header(h).Get(key) }
func (h headerCarrier) Set(key, value string) { nats.Header(h).Set(key, value) }
func (h headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys
}
