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

type Registry struct {
	routes   map[routeKey]string
	handlers map[routeKey]func(context.Context, domainevent.Event) error
}

func NewRegistry(routes ...Route) (*Registry, error) {
	subjects, err := buildRoutes(routes)
	if err != nil {
		return nil, err
	}
	return &Registry{routes: subjects, handlers: make(map[routeKey]func(context.Context, domainevent.Event) error)}, nil
}

// Handle registers a typed handler without exposing subjects, headers,
// delivery attempts, or acknowledgements to business code.
func Handle[T any](registry *Registry, kind domainevent.Kind[T], handler func(context.Context, domainevent.Typed[T]) error) error {
	if handler == nil {
		return fmt.Errorf("%w: event handler is required", ErrRejected)
	}
	err := registry.Register(kind.Type, kind.Version, func(ctx context.Context, event domainevent.Event) error {
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

func (r *Registry) Register(eventType string, version uint16, handler func(context.Context, domainevent.Event) error) error {
	if r == nil || r.routes == nil {
		return fmt.Errorf("%w: event registry is required", ErrRejected)
	}
	key := routeKey{typeName: eventType, version: version}
	if _, ok := r.routes[key]; !ok {
		return fmt.Errorf("%w: no route for %s v%d", ErrRejected, eventType, version)
	}
	if handler == nil {
		return fmt.Errorf("%w: handler is required for %s v%d", ErrRejected, eventType, version)
	}
	if _, exists := r.handlers[key]; exists {
		return fmt.Errorf("%w: duplicate handler for %s v%d", ErrRejected, eventType, version)
	}
	r.handlers[key] = handler
	return nil
}

func (r *Registry) Handler() (Handler, error) {
	if r == nil || len(r.handlers) == 0 {
		return nil, fmt.Errorf("%w: no typed event handlers are registered", ErrRejected)
	}
	return func(ctx context.Context, message Message) error {
		version, err := schemaVersion(message.Schema())
		if err != nil {
			return Permanent(err)
		}
		key := routeKey{typeName: message.Type(), version: version}
		handler, ok := r.handlers[key]
		if !ok {
			return Permanent(fmt.Errorf("no handler for %s v%d", message.Type(), version))
		}
		if subject := r.routes[key]; message.Subject() != subject {
			return domainevent.Permanent(fmt.Errorf(
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
	return &Publisher{producer: producer, routes: r.routes}, nil
}

type Publisher struct {
	producer *Producer
	routes   map[routeKey]string
}

func (p *Publisher) Publish(ctx context.Context, event domainevent.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate domain event: %w", err)
	}
	subject, ok := p.routes[routeKey{typeName: event.Type, version: event.Version}]
	if !ok {
		return fmt.Errorf("%w: no route for %s v%d", ErrRejected, event.Type, event.Version)
	}
	_, err := p.producer.Publish(ctx, Event{
		Subject: subject, MessageID: event.ID, PublicationID: event.ID,
		Type: event.Type, Schema: "v" + strconv.FormatUint(uint64(event.Version), 10),
		CreatedAt: event.OccurredAt, Payload: event.Payload,
	})
	return err
}

func buildRoutes(routes []Route) (map[routeKey]string, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("%w: at least one event route is required", ErrRejected)
	}
	subjects := make(map[routeKey]string, len(routes))
	for _, route := range routes {
		if err := validateRequiredValue("event type", route.Type); err != nil {
			return nil, err
		}
		if route.Version == 0 {
			return nil, fmt.Errorf("%w: event version must be positive", ErrRejected)
		}
		if !validSubject(route.Subject, false) {
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
	if err != nil || version == 0 || schema != "v"+strconv.FormatUint(version, 10) {
		return 0, fmt.Errorf("invalid event schema %q", schema)
	}
	return uint16(version), nil
}
