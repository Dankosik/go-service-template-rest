package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
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
	client, err := natsjs.Connect(
		ctx,
		runtimeopts.Messaging(cfg),
		natsjs.RoleProducer,
		natsjs.Observability{Logger: log},
	)
	if err != nil {
		return messagingRuntime{}, fmt.Errorf("initialize producer messaging: %w", err)
	}
	return messagingRuntime{client: client}, nil
}

// Producer is the seam a feature takes to publish from the API process. The
// template wires no publisher, so nothing here calls it yet — the same gap
// postgresoutbox states for its own append side, where cmd/service builds no
// Store at all. Deleting it as unused would remove the one accessor a feature
// needs; the worked wiring is in docs/durable-messaging.md.
func (m messagingRuntime) Producer() *natsjs.Producer {
	if m.client == nil {
		return nil
	}
	return m.client.Producer()
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
