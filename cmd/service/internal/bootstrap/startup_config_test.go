package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

func TestFailedConfigStage(t *testing.T) {
	t.Parallel()

	if got := failedConfigStage(config.LoadReport{}); got != config.StageLoadDefaults {
		t.Fatalf("failedConfigStage() = %q, want %q", got, config.StageLoadDefaults)
	}
	if got := failedConfigStage(config.LoadReport{FailedStage: config.StageValidate}); got != config.StageValidate {
		t.Fatalf("failedConfigStage() = %q, want %q", got, config.StageValidate)
	}
}

func TestBootstrapConfigStageReturnsConfigLoadFailure(t *testing.T) {
	t.Setenv("APP__APP__ENV", "local")

	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{ConfigPath: missingConfig})
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want non-nil")
	}
}
