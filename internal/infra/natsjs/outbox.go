package natsjs

import (
	"fmt"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

// NewOutboxAppender builds the service-owned event routing table once. Domain
// code appends events without ever seeing the selected NATS subject.
func NewOutboxAppender(maxPayloadBytes int, routes ...Route) (*postgresoutbox.Appender, error) {
	subjects, err := buildRoutes(routes)
	if err != nil {
		return nil, err
	}
	appender, err := postgresoutbox.NewAppender(maxPayloadBytes, func(event domainevent.Event) (string, error) {
		if err := validateRequiredValue("message ID", event.ID); err != nil {
			return "", err
		}
		return subjects.subject(event.Type, event.Version)
	})
	if err != nil {
		return nil, fmt.Errorf("initialize PostgreSQL outbox appender: %w", err)
	}
	return appender, nil
}
