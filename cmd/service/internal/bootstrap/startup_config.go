package bootstrap

import (
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
)

const (
	startupConfigCompatibilityStage  = "startup.config.compatibility"
	startupConfigCompatibilityReason = "startup_compatibility"
)

func failedConfigStage(report config.LoadReport) string {
	stage := strings.TrimSpace(report.FailedStage)
	if stage == "" {
		return config.StageLoadDefaults
	}
	return stage
}
