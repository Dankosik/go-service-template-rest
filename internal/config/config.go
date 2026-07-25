package config

import (
	"context"
	"time"
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

	cfg, unknownKeys, err := buildSnapshot(k)
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

	err = validateConfig(
		&cfg,
		append(unknownKeys, metadata.sectionScalarOverrideKeys...),
	)
	report.ValidateDuration = time.Since(validateStarted)
	if err != nil {
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	return cfg, report, nil
}
