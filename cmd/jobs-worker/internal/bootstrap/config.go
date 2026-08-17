package bootstrap

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresjobs"
	"github.com/example/go-service-template-rest/internal/jobs"
)

const (
	startupTimeout   = 15 * time.Second
	diagnosticsClose = 2 * time.Second
	telemetryClose   = 5 * time.Second
)

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	return config.ParseLoadOptions("jobs-worker", args, nil)
}

func engineConfig(cfg config.JobsConfig, instanceID string) (postgresjobs.EngineConfig, error) {
	suffix := rand.Text()
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" || !utf8.ValidString(instanceID) || len(instanceID) >= jobs.MaxIdentityBytes-len(suffix) {
		return postgresjobs.EngineConfig{}, fmt.Errorf("%w: jobs worker instance identity is invalid", postgresjobs.ErrConfig)
	}
	for _, character := range instanceID {
		if unicode.IsControl(character) {
			return postgresjobs.EngineConfig{}, fmt.Errorf("%w: jobs worker instance identity contains a control character", postgresjobs.ErrConfig)
		}
	}
	workerID := instanceID + "/" + suffix
	observationMaxAge := cfg.ObservationInterval
	for _, interval := range []time.Duration{cfg.PollInterval, cfg.StoreOperationTimeout} {
		if interval > time.Duration(math.MaxInt64)-observationMaxAge {
			return postgresjobs.EngineConfig{}, fmt.Errorf("%w: jobs observation freshness envelope overflows", postgresjobs.ErrConfig)
		}
		observationMaxAge += interval
	}
	return postgresjobs.EngineConfig{
		WorkerID: workerID, MaxConcurrency: cfg.MaxConcurrency, LeaseDuration: cfg.LeaseDuration,
		ObservationInterval: cfg.ObservationInterval, ObservationMaxAge: observationMaxAge, DrainTimeout: cfg.DrainTimeout,
	}, nil
}

func validateRuntimeConfig(cfg config.Config) error {
	if !cfg.Jobs.Enabled || !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: jobs and postgres must be enabled for jobs-worker", postgresjobs.ErrConfig)
	}
	if strings.TrimSpace(cfg.Observability.Metrics.Addr) == "" {
		return fmt.Errorf("%w: jobs worker diagnostics address is required", postgresjobs.ErrConfig)
	}
	return runtimeopts.ValidateGracePeriod(cfg.HTTP.GracePeriod, "jobs.drain_timeout", cfg.Jobs.DrainTimeout, diagnosticsClose+telemetryClose)
}

func validateTerminationEnvelope(gracePeriod, required time.Duration) error {
	if required > gracePeriod {
		return fmt.Errorf("%w: jobs termination envelope %s exceeds http.grace_period %s", postgresjobs.ErrConfig, required, gracePeriod)
	}
	return nil
}
