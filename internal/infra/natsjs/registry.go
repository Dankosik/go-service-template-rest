package natsjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/example/go-service-template-rest/internal/domainevent"
)

// Route is composition-owned broker routing for one typed event kind.
type Route struct {
	Type    string
	Version uint16
	Subject string
}

type routeKey struct {
	typeName string
	version  uint16
}

// routeTable maps each routed event kind to the subject it is published on.
type routeTable map[routeKey]string

// subject reports where eventType at version is published. An unrouted kind is
// an [ErrRejected] refusal.
func (t routeTable) subject(eventType string, version uint16) (string, error) {
	subject, ok := t[routeKey{typeName: eventType, version: version}]
	if !ok {
		return "", fmt.Errorf("%w: no route for %s v%d", ErrRejected, eventType, version)
	}
	return subject, nil
}

type Registry struct {
	subjects      routeTable
	eventHandlers map[routeKey]func(context.Context, domainevent.Event) error
}

func NewRegistry(routes ...Route) (*Registry, error) {
	subjects, err := buildRoutes(routes)
	if err != nil {
		return nil, err
	}
	return &Registry{subjects: subjects, eventHandlers: make(map[routeKey]func(context.Context, domainevent.Event) error)}, nil
}

// Handle registers a typed handler without exposing subjects, headers,
// delivery attempts, or acknowledgements to business code.
func (r *Registry) Handle[T any](kind domainevent.Kind[T], handler func(context.Context, domainevent.Typed[T]) error) error {
	if handler == nil {
		return fmt.Errorf("%w: event handler is required", ErrRejected)
	}
	err := r.register(kind.Type, kind.Version, func(ctx context.Context, event domainevent.Event) error {
		var payload T
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return Permanent(fmt.Errorf("decode %s v%d: %w", event.Type, event.Version, err))
		}
		return handler(ctx, domainevent.Typed[T]{ID: event.ID, OccurredAt: event.OccurredAt, Payload: payload})
	})
	if err != nil {
		return fmt.Errorf("register typed event handler: %w", err)
	}
	return nil
}

// register admits one decoded-event handler for a routed kind; [Registry.Handle]
// is the entry point that builds it from a typed handler.
func (r *Registry) register(eventType string, version uint16, handler func(context.Context, domainevent.Event) error) error {
	if r == nil || r.subjects == nil {
		return fmt.Errorf("%w: event registry is required", ErrRejected)
	}
	if _, err := r.subjects.subject(eventType, version); err != nil {
		return err
	}
	key := routeKey{typeName: eventType, version: version}
	if handler == nil {
		return fmt.Errorf("%w: handler is required for %s v%d", ErrRejected, eventType, version)
	}
	if _, exists := r.eventHandlers[key]; exists {
		return fmt.Errorf("%w: duplicate handler for %s v%d", ErrRejected, eventType, version)
	}
	r.eventHandlers[key] = handler
	return nil
}

// Handler reads the live registry. Complete registration before using the
// returned handler for concurrent deliveries.
func (r *Registry) Handler() (Handler, error) {
	if r == nil || len(r.eventHandlers) == 0 {
		return nil, fmt.Errorf("%w: no typed event handlers are registered", ErrRejected)
	}
	return func(ctx context.Context, message Message) error {
		version, err := schemaVersion(message.Schema())
		if err != nil {
			return Permanent(err)
		}
		key := routeKey{typeName: message.Type(), version: version}
		handler, ok := r.eventHandlers[key]
		if !ok {
			return Permanent(fmt.Errorf("no handler for %s v%d", message.Type(), version))
		}
		if subject := r.subjects[key]; message.Subject() != subject {
			return Permanent(fmt.Errorf(
				"unexpected subject %q for %s v%d, want %q",
				message.Subject(), message.Type(), version, subject,
			))
		}
		return handler(ctx, domainevent.Event{
			ID: message.MessageID(), Type: message.Type(), Version: version,
			OccurredAt: message.CreatedAt(), Payload: message.Payload(),
		})
	}, nil
}

// Publisher returns the typed business publisher backed by this registry's
// composition-owned routes.
func (r *Registry) Publisher(producer *Producer) (*Publisher, error) {
	if producer == nil {
		return nil, fmt.Errorf("%w: producer is required", ErrRejected)
	}
	return &Publisher{producer: producer, subjects: r.subjects}, nil
}

type Publisher struct {
	producer *Producer
	subjects routeTable
}

// SchemaForVersion returns the NATS event-schema spelling for version. It
// formats every uint16, including zero; registry admission and parsing own
// whether a particular version is valid.
func SchemaForVersion(version uint16) string {
	return "v" + strconv.FormatUint(uint64(version), 10)
}

func (p *Publisher) Publish(ctx context.Context, event domainevent.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate domain event: %w", err)
	}
	subject, err := p.subjects.subject(event.Type, event.Version)
	if err != nil {
		return err
	}
	_, err = p.producer.Publish(ctx, EventFromDomain(subject, event))
	return err
}

// EventFromDomain maps a domain occurrence to the NATS publication envelope.
// The caller supplies the routed or durably stored subject and owns admission.
func EventFromDomain(subject string, event domainevent.Event) Event {
	return Event{
		Subject: subject, MessageID: event.ID, PublicationID: event.ID,
		Type: event.Type, Schema: SchemaForVersion(event.Version),
		CreatedAt: event.OccurredAt, Payload: event.Payload,
	}
}

func buildRoutes(routes []Route) (routeTable, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("%w: at least one event route is required", ErrRejected)
	}
	subjects := make(routeTable, len(routes))
	for _, route := range routes {
		if err := validateRequiredValue("event type", route.Type); err != nil {
			return nil, err
		}
		if route.Version == 0 {
			return nil, fmt.Errorf("%w: event version must be positive", ErrRejected)
		}
		if !validPublishSubject(route.Subject) {
			return nil, fmt.Errorf("%w: invalid event subject", ErrRejected)
		}
		key := routeKey{typeName: route.Type, version: route.Version}
		if _, exists := subjects[key]; exists {
			return nil, fmt.Errorf("%w: duplicate route for %s v%d", ErrRejected, route.Type, route.Version)
		}
		subjects[key] = route.Subject
	}
	return subjects, nil
}

func schemaVersion(schema string) (uint16, error) {
	if len(schema) < 2 || schema[0] != 'v' {
		return 0, fmt.Errorf("invalid event schema %q", schema)
	}
	version, err := strconv.ParseUint(schema[1:], 10, 16)
	if err != nil || version == 0 || schema != SchemaForVersion(uint16(version)) {
		return 0, fmt.Errorf("invalid event schema %q", schema)
	}
	return uint16(version), nil
}
