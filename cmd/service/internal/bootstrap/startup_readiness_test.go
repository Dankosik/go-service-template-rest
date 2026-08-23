package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/waittest"
)

func TestReadinessTransitionsUpdateGRPCHealth(t *testing.T) {
	probe := &switchingProbe{}
	log := slog.New(slog.DiscardHandler)
	supervisor := newSupervisedBackground(context.Background(), log)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = supervisor.Shutdown(ctx)
	})
	service := newReadinessService([]health.Probe{probe}, supervisor)
	grpcServer := newFakeGRPCRuntimeServer()
	superviseReadiness(config.Config{
		HTTP: config.HTTPConfig{ReadinessTimeout: 100 * time.Millisecond},
		Health: config.HealthConfig{
			RefreshInterval:  10 * time.Millisecond,
			FailureThreshold: 1,
		},
	}, log, service, supervisor, grpcServer)
	waittest.UntilFunc(t, time.Second, func(context.Context) bool {
		return service.Cached() == nil
	}, func() string { return "initial readiness evaluation" })

	probe.failed.Store(true)
	if ready := waittest.Receive(t, grpcServer.serving, time.Second, "gRPC health to follow readiness failure"); ready {
		t.Fatal("gRPC health remained serving after readiness failure")
	}
	probe.failed.Store(false)
	if ready := waittest.Receive(t, grpcServer.serving, time.Second, "gRPC health to follow readiness recovery"); !ready {
		t.Fatal("gRPC health remained not serving after readiness recovery")
	}
}

type switchingProbe struct {
	failed atomic.Bool
}

func (*switchingProbe) Name() string { return "switching" }

func (p *switchingProbe) Check(context.Context) error {
	if p.failed.Load() {
		return errors.New("not ready")
	}
	return nil
}
