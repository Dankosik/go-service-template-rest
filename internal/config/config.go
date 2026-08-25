package config

import (
	"context"
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

type LoadReport struct {
	LoadDuration     time.Duration
	ValidateDuration time.Duration
	FailedStage      string
}

// LoadDetailed loads without a caller context. Binaries use
// LoadDetailedWithContext so a startup budget can cancel the load.
func LoadDetailed(opts LoadOptions) (Config, LoadReport, error) {
	return LoadDetailedWithContext(context.Background(), opts)
}

func LoadDetailedWithContext(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	return loadDetailedWithContext(ctx, opts, buildSnapshot, validateConfig)
}

// LoadJobsWorkerDetailedWithContext loads the immutable snapshot required by
// the jobs-worker binary. It validates only the sections that binary consumes.
// profile:jobs-postgres:start
func LoadJobsWorkerDetailedWithContext(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	return loadDetailedWithContext(ctx, opts, buildJobsWorkerSnapshot, validateJobsWorkerConfig)
}

// profile:jobs-postgres:end

func loadDetailedWithContext(
	ctx context.Context,
	opts LoadOptions,
	build func(*koanf.Koanf) (Config, []string, error),
	validate func(*Config, []string) error,
) (Config, LoadReport, error) {
	if err := checkContext(ctx); err != nil {
		return Config{}, LoadReport{}, err
	}

	loadStarted := time.Now()
	k, metadata, err := loadKoanf(ctx, opts)
	report := LoadReport{
		LoadDuration: time.Since(loadStarted),
		FailedStage:  metadata.failedStage,
	}
	if err != nil {
		if report.FailedStage == "" {
			report.FailedStage = StageLoadDefaults
		}
		return Config{}, report, err
	}
	if err := checkContext(ctx); err != nil {
		report.FailedStage = StageLoadEnv
		return Config{}, report, err
	}

	cfg, unknownKeys, err := build(k)
	if err != nil {
		report.FailedStage = StageParse
		return Config{}, report, err
	}
	if err := checkContext(ctx); err != nil {
		report.FailedStage = StageParse
		return Config{}, report, err
	}

	validateStarted := time.Now()
	if err := checkValidateContext(ctx); err != nil {
		report.ValidateDuration = time.Since(validateStarted)
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	unknownKeys = append(unknownKeys, metadata.sectionScalarOverrideKeys...)
	unknownKeys = append(unknownKeys, metadata.malformedEnvironmentKeys...)
	err = validate(&cfg, unknownKeys)
	report.ValidateDuration = time.Since(validateStarted)
	if err != nil {
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	return cfg, report, nil
}
