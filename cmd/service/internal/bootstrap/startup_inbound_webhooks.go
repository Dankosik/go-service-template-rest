// profile:inbound-webhooks-standard:start
package bootstrap

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgresinboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

func initInboundWebhookReceiver(
	cfg config.Config,
	pool *pgxpool.Pool,
	metrics *telemetry.Metrics,
	log *slog.Logger,
) (inboundwebhook.Receiver, error) {
	if cfg.InboundWebhooks.Endpoints == "" && cfg.InboundWebhooks.StaticSecrets == "" && pool == nil {
		return inboundwebhook.NoopReceiver{}, nil
	}
	if pool == nil {
		return nil, errors.New("initialize inbound webhooks: postgres is required")
	}
	endpoints, err := postgresinboundwebhook.ParseEndpointManifest(cfg.InboundWebhooks.Endpoints)
	if err != nil {
		return nil, fmt.Errorf("initialize inbound webhooks: %w", err)
	}
	secrets, err := postgresinboundwebhook.ParseSecretManifest(cfg.InboundWebhooks.StaticSecrets)
	if err != nil {
		return nil, fmt.Errorf("initialize inbound webhooks: %w", err)
	}
	trust, err := postgresinboundwebhook.BindSecrets(endpoints, secrets)
	if err != nil {
		return nil, fmt.Errorf("initialize inbound webhooks: %w", err)
	}
	receiver, err := postgresinboundwebhook.NewReceiver(
		pool,
		trust,
		postgresinboundwebhook.WithMeter(metrics.MeterProvider(), log),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize inbound webhooks: %w", err)
	}
	return receiver, nil
}

// profile:inbound-webhooks-standard:end
