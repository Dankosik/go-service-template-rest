package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

type messagingRuntime struct {
	client *natsjs.Client
}

func initMessagingRuntime(ctx context.Context, cfg config.MessagingConfig, log *slog.Logger) (messagingRuntime, error) {
	if !cfg.Enabled {
		return messagingRuntime{}, nil
	}
	client, err := natsjs.Connect(ctx, natsjs.Config{
		URLs:                 splitMessagingURLs(cfg.URLs),
		CredentialsFile:      cfg.CredentialsFile,
		RootCAFile:           cfg.RootCAFile,
		AllowPlaintext:       cfg.AllowPlaintext,
		AllowUnauthenticated: cfg.AllowUnauthenticated,
		Stream:               cfg.Stream,
		MaxPayloadBytes:      cfg.MaxPayloadBytes,
		MaxPendingPublishes:  cfg.MaxPendingPublishes,
	}, natsjs.RoleProducer, natsjs.Observability{Logger: log})
	if err != nil {
		return messagingRuntime{}, fmt.Errorf("initialize producer messaging: %w", err)
	}
	return messagingRuntime{client: client}, nil
}

func splitMessagingURLs(value string) []string {
	values := make([]string, 0)
	for item := range strings.SplitSeq(value, ",") {
		values = append(values, strings.TrimSpace(item))
	}
	return values
}

func (m messagingRuntime) Producer() *natsjs.Producer {
	if m.client == nil {
		return nil
	}
	return m.client.Producer()
}

func (m messagingRuntime) Ready() bool {
	return m.client == nil || m.client.Ready()
}

func (m messagingRuntime) ReadinessProbes() []health.Probe {
	if m.client == nil {
		return nil
	}
	return []health.Probe{m.client}
}

func (m messagingRuntime) StartDrain() {
	if m.client != nil {
		m.client.StopPublish()
	}
}

func (m messagingRuntime) Shutdown(ctx context.Context) error {
	if m.client == nil {
		return nil
	}
	if err := m.client.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown producer messaging: %w", err)
	}
	return nil
}

func (m messagingRuntime) Close() {
	if m.client != nil {
		m.client.Close()
	}
}
