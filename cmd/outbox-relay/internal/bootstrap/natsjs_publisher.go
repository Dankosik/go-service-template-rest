package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

func BuildNATSPublisher(ctx context.Context, cfg config.Config, log *slog.Logger) (PublisherRuntime, error) {
	if !cfg.Messaging.Enabled {
		return PublisherRuntime{}, fmt.Errorf("%w: messaging must be enabled for the NATS outbox publisher", postgresoutbox.ErrConfig)
	}
	if cfg.Outbox.PublishConcurrency > cfg.Messaging.MaxPendingPublishes {
		return PublisherRuntime{}, fmt.Errorf(
			"%w: outbox.publish_concurrency must not exceed messaging.max_pending_publishes",
			postgresoutbox.ErrConfig,
		)
	}
	client, err := natsjs.Connect(ctx, runtimeopts.Messaging(cfg.Messaging), natsjs.RoleProducer, natsjs.Observability{Logger: log})
	if err != nil {
		return PublisherRuntime{}, fmt.Errorf("connect NATS outbox publisher: %w", err)
	}
	runtime, err := NewPublisherRuntime(natsjs.NewOutboxPublisher(client.Producer()), client.Run, client.Ready, client.Shutdown)
	if err != nil {
		client.Close()
		return PublisherRuntime{}, err
	}
	return runtime, nil
}
