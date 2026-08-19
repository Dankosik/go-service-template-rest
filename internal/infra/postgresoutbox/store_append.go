package postgresoutbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
)

var (
	ErrConfig          = errors.New("postgres outbox config")
	ErrEventIDConflict = errors.New("domain event ID conflict")
)

const Queue = "outbox"

// Router maps business event identity to the concrete broker address retained
// with the publication job. Business code never receives that address.
type Router func(domainevent.Event) (string, error)

type Appender struct {
	client          *river.Client[pgx.Tx]
	maxPayloadBytes int
	route           Router
}

func NewAppender(maxPayloadBytes int, route Router) (*Appender, error) {
	if maxPayloadBytes <= 0 {
		return nil, fmt.Errorf("%w: max payload bytes must be positive", ErrConfig)
	}
	if route == nil {
		return nil, fmt.Errorf("%w: event router is required", ErrConfig)
	}
	telemetry := otelriver.NewMiddleware(&otelriver.MiddlewareConfig{
		EnableSemanticMetrics:  true,
		EnableTracePropagation: true,
	})
	client, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Plugins: []rivertype.Plugin{telemetry},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize River outbox appender: %w", err)
	}
	return &Appender{client: client, maxPayloadBytes: maxPayloadBytes, route: route}, nil
}

// Append stores one immutable publication job in the transaction owned by the
// caller. It never begins or commits a transaction.
func (a *Appender) Append(ctx context.Context, tx pgx.Tx, event domainevent.Event) error {
	if a == nil || a.client == nil || a.route == nil {
		return fmt.Errorf("%w: appender is required", ErrConfig)
	}
	if tx == nil {
		return fmt.Errorf("%w: transaction is required", ErrConfig)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate domain event: %w", err)
	}
	if len(event.Payload) > a.maxPayloadBytes {
		return fmt.Errorf("validate domain event: payload exceeds %d bytes", a.maxPayloadBytes)
	}
	subject, err := a.route(event)
	if err != nil {
		return fmt.Errorf("route domain event: %w", err)
	}
	if subject == "" {
		return errors.New("route domain event: subject is required")
	}

	args := PublishJob{
		ID: event.ID, Type: event.Type, Version: event.Version,
		OccurredAt: event.OccurredAt, Payload: event.Payload, Subject: subject,
	}
	result, err := a.client.InsertTx(ctx, tx, args, nil)
	if err != nil {
		return fmt.Errorf("insert River outbox job: %w", err)
	}
	if result.UniqueSkippedAsDuplicate && (result.Job == nil || !sameJob(result.Job.EncodedArgs, args)) {
		return fmt.Errorf("%w: event %q already names different bytes", ErrEventIDConflict, event.ID)
	}
	return nil
}

func sameJob(encoded []byte, want PublishJob) bool {
	var got PublishJob
	if json.Unmarshal(encoded, &got) != nil {
		return false
	}
	return got.ID == want.ID &&
		got.Type == want.Type &&
		got.Version == want.Version &&
		got.OccurredAt.Equal(want.OccurredAt) &&
		got.Subject == want.Subject &&
		bytes.Equal(got.Payload, want.Payload)
}
