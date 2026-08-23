package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

type messagingRuntime struct {
	client *natsjs.Client
}

func initMessagingRuntime(ctx context.Context, cfg config.MessagingConfig, log *slog.Logger) (messagingRuntime, error) {
	if strings.TrimSpace(cfg.URLs) == "" {
		return messagingRuntime{}, nil
	}
	client, err := natsjs.Connect(
		ctx,
		runtimeopts.Messaging(cfg),
		natsjs.Observability{Logger: log},
	)
	if err != nil {
		return messagingRuntime{}, fmt.Errorf("initialize producer messaging: %w", err)
	}
	return messagingRuntime{client: client}, nil
}

// Publisher is the typed business seam. Routes stay in composition and never
// reach the feature that creates a domainevent.Event.
func (m messagingRuntime) Publisher(routes ...natsjs.Route) (*natsjs.Publisher, error) {
	if m.client == nil {
		return nil, errors.New("messaging is disabled")
	}
	registry, err := natsjs.NewRegistry(routes...)
	if err != nil {
		return nil, fmt.Errorf("build messaging registry: %w", err)
	}
	publisher, err := registry.Publisher(m.client.Producer())
	if err != nil {
		return nil, fmt.Errorf("build messaging publisher: %w", err)
	}
	return publisher, nil
}

// ConnectionRun is the client's connection supervisor loop, or nil when
// messaging is disabled. It exists so the composition root reaches the client
// through this type's accessors like every other caller, rather than testing
// m.client itself and then calling straight through it.
func (m messagingRuntime) ConnectionRun() func(context.Context) error {
	if m.client == nil {
		return nil
	}
	return m.client.Run
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
