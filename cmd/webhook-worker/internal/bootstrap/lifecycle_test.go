package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestWebhookWorkerLifecycleUnexpectedExit(t *testing.T) {
	cfg := config.Config{}
	cfg.Observability.Metrics.Addr = "127.0.0.1:0"
	cfg.HTTP.GracePeriod = time.Second
	result := runLifecycle(context.Background(), t.Context(), cfg, telemetry.New(), nil)
	if result.Err == nil || !result.CleanupSafe {
		t.Fatalf("runLifecycle(nil worker) = %+v", result)
	}
}
