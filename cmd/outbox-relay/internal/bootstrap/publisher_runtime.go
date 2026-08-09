package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

// PublisherRuntime is one publisher and the lifecycle of the client that owns
// it. Its fields stay private so only the validating constructor can create a
// value admitted by the relay process. It remains available to outbox-only
// services after a selected transport builder is removed.
type PublisherRuntime struct {
	publisher postgresoutbox.Publisher
	run       func(context.Context) error
	ready     func() bool
	shutdown  func(context.Context) error
}

func NewPublisherRuntime(
	publisher postgresoutbox.Publisher,
	run func(context.Context) error,
	ready func() bool,
	shutdown func(context.Context) error,
) (PublisherRuntime, error) {
	runtime := PublisherRuntime{publisher: publisher, run: run, ready: ready, shutdown: shutdown}
	if err := validatePublisherRuntime(runtime); err != nil {
		return PublisherRuntime{}, err
	}
	return runtime, nil
}

func validatePublisherRuntime(runtime PublisherRuntime) error {
	if err := postgresoutbox.ValidatePublisher(runtime.publisher); err != nil {
		return fmt.Errorf("validate publisher: %w", err)
	}
	if runtime.run == nil || runtime.ready == nil || runtime.shutdown == nil {
		return fmt.Errorf("%w: publisher runtime lifecycle is incomplete", postgresoutbox.ErrConfig)
	}
	return nil
}

type PublisherBuilder func(context.Context, config.Config, *slog.Logger) (PublisherRuntime, error)
