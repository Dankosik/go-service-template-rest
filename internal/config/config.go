package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
)

const (
	StageLoadDefaults = "config.load.defaults"
	StageLoadFile     = "config.load.file"
	StageLoadEnv      = "config.load.env"
	StageParse        = "config.parse"
	StageValidate     = "config.validate"
)

type LoadOptions struct {
	ConfigPath     string
	ConfigOverlays []string
}

// LoadReport describes a load attempt. When the load returns an error,
// FailedStage always names the stage that failed.
type LoadReport struct {
	LoadDuration     time.Duration
	ValidateDuration time.Duration
	FailedStage      string
}

// Load loads and validates the immutable service snapshot. ctx is the startup
// budget, so it can cancel the load between stages.
func Load(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	return load(ctx, opts, buildSnapshot, validateConfig)
}

// profile:jobs-postgres:start
// LoadJobsWorker loads the immutable snapshot required by the jobs-worker
// binary. It validates only the sections that binary consumes.
func LoadJobsWorker(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	return load(ctx, opts, buildJobsWorkerSnapshot, validateJobsWorkerConfig)
}

// profile:jobs-postgres:end

func load(
	ctx context.Context,
	opts LoadOptions,
	build func(*koanf.Koanf) (Config, []string, error),
	validate func(*Config) error,
) (Config, LoadReport, error) {
	if err := checkLoadContext(ctx); err != nil {
		return Config{}, LoadReport{FailedStage: StageLoadDefaults}, err
	}

	loadStarted := time.Now()
	k, metadata, err := loadKoanf(ctx, opts)
	report := LoadReport{LoadDuration: time.Since(loadStarted)}
	if err != nil {
		report.FailedStage = metadata.failedStage
		return Config{}, report, err
	}

	cfg, unknownKeys, err := build(k)
	if err != nil {
		report.FailedStage = StageParse
		return Config{}, report, err
	}
	if err := checkLoadContext(ctx); err != nil {
		report.FailedStage = StageParse
		return Config{}, report, err
	}

	validateStarted := time.Now()
	if err := checkValidateContext(ctx); err != nil {
		report.ValidateDuration = time.Since(validateStarted)
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	// Unknown keys are rejected before any section rule runs.
	unknownKeys = append(unknownKeys, metadata.sectionScalarOverrideKeys...)
	unknownKeys = append(unknownKeys, metadata.malformedEnvironmentKeys...)
	if unknown := normalizeUnknownKeys(unknownKeys); len(unknown) > 0 {
		err = fmt.Errorf("%w: unknown keys: %s", ErrUnknownKey, strings.Join(unknown, ", "))
	} else {
		err = validate(&cfg)
	}
	report.ValidateDuration = time.Since(validateStarted)
	if err != nil {
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	return cfg, report, nil
}
