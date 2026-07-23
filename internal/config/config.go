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
	Strict         bool
	LoadBudget     time.Duration
	ValidateBudget time.Duration
}

type LoadReport struct {
	LoadDuration       time.Duration
	ValidateDuration   time.Duration
	UnknownKeyWarnings []string
	FailedStage        string
}

func Load() (Config, error) {
	cfg, _, err := LoadDetailed(LoadOptions{})
	return cfg, err
}

func LoadDetailed(opts LoadOptions) (Config, LoadReport, error) {
	return LoadDetailedWithContext(context.Background(), opts)
}

func LoadDetailedWithContext(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	if err := checkContext(ctx); err != nil {
		return Config{}, LoadReport{}, err
	}

	loadCtx, loadCancel := withContextBudget(ctx, opts.LoadBudget)
	defer loadCancel()

	loadStarted := time.Now()
	k, metadata, err := loadKoanf(loadCtx, opts)
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
	if err := checkContext(loadCtx); err != nil {
		report.FailedStage = StageLoadEnv
		return Config{}, report, err
	}

	cfg, unknownKeys, err := buildSnapshot(k)
	if err != nil {
		report.FailedStage = StageParse
		return Config{}, report, err
	}
	if err := checkContext(loadCtx); err != nil {
		report.FailedStage = StageParse
		return Config{}, report, err
	}

	validateStarted := time.Now()
	validateCtx, validateCancel := withContextBudget(ctx, opts.ValidateBudget)
	defer validateCancel()
	if err := checkValidateContext(validateCtx); err != nil {
		report.ValidateDuration = time.Since(validateStarted)
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	validationResult, err := validateConfig(validateCtx, &cfg, validationOptions{
		Strict:      opts.Strict,
		UnknownKeys: append(unknownKeys, metadata.sectionScalarOverrideKeys...),
	})
	report.ValidateDuration = time.Since(validateStarted)
	report.UnknownKeyWarnings = validationResult.UnknownKeyWarnings
	if err != nil {
		report.FailedStage = StageValidate
		return Config{}, report, err
	}

	return cfg, report, nil
}
