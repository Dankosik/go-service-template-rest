package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
)

func TestReadinessTransitionsUpdateGRPCHealth(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		const refreshInterval = 10 * time.Millisecond
		probe := &switchingProbe{}
		log := slog.New(slog.DiscardHandler)
		supervisor := newSupervisedBackground(t.Context(), log)
		service := newReadinessService([]health.Probe{probe}, supervisor)
		grpcServer := newFakeGRPCRuntimeServer()
		superviseReadiness(config.Config{
			HTTP: config.HTTPConfig{ReadinessTimeout: 100 * time.Millisecond},
			Health: config.HealthConfig{
				RefreshInterval:  refreshInterval,
				FailureThreshold: 1,
			},
		}, log, service, supervisor, grpcServer)
		synctest.Wait()
		if err := service.Cached(); err != nil {
			t.Fatalf("initial readiness error = %v", err)
		}

		probe.failed.Store(true)
		synctest.Sleep(refreshInterval)
		synctest.Wait()
		if ready := <-grpcServer.serving; ready {
			t.Fatal("gRPC health remained serving after readiness failure")
		}

		probe.failed.Store(false)
		synctest.Sleep(refreshInterval)
		synctest.Wait()
		if ready := <-grpcServer.serving; !ready {
			t.Fatal("gRPC health remained not serving after readiness recovery")
		}

		service.StartDrain()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := supervisor.Shutdown(ctx); err != nil {
			t.Fatalf("shutdown readiness supervisor: %v", err)
		}
	})
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
