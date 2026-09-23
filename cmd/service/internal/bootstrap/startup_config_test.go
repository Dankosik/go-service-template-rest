package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
)

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestBootstrapConfigStageReturnsConfigLoadFailure(t *testing.T) {
	t.Setenv("APP__APP__ENV", "local")

	missingConfig := filepath.Join(t.TempDir(), "missing.yaml")

	_, _, err := bootstrapConfigStage(context.Background(), config.LoadOptions{ConfigPath: missingConfig})
	if err == nil {
		t.Fatal("bootstrapConfigStage() error = nil, want non-nil")
	}
}
