package config

import (
	"context"
	"strings"
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
	LoadDuration         time.Duration
	LoadDefaultsDuration time.Duration
	LoadFileDuration     time.Duration
	LoadEnvDuration      time.Duration
	ParseDuration        time.Duration
	ValidateDuration     time.Duration
	UnknownKeyWarnings   []string
	FailedStage          string
	FailedStageDuration  time.Duration
}

func Load() (Config, error) {
	cfg, _, err := LoadDetailed(LoadOptions{})
	return cfg, err
}

func LoadWithOptions(opts LoadOptions) (Config, error) {
	cfg, _, err := LoadDetailed(opts)
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
		LoadDuration:         time.Since(loadStarted),
		LoadDefaultsDuration: metadata.loadDefaultsDuration,
		LoadFileDuration:     metadata.loadFileDuration,
		LoadEnvDuration:      metadata.loadEnvDuration,
		FailedStage:          metadata.failedStage,
		FailedStageDuration:  metadata.failedStageDuration,
	}
	if err != nil {
		report.markFailedStage(report.FailedStage, report.FailedStageDuration)
		return Config{}, report, err
	}
	if err := checkContext(loadCtx); err != nil {
		report.markFailedStage(StageLoadEnv, report.LoadEnvDuration)
		return Config{}, report, err
	}

	parseStarted := time.Now()
	cfg, err := buildSnapshot(k)
	report.ParseDuration = time.Since(parseStarted)
	if err != nil {
		report.FailedStage = StageParse
		report.FailedStageDuration = report.ParseDuration
		return Config{}, report, err
	}
	if err := checkContext(loadCtx); err != nil {
		report.markFailedStage(StageParse, report.ParseDuration)
		return Config{}, report, err
	}

	validateStarted := time.Now()
	validateCtx, validateCancel := withContextBudget(ctx, opts.ValidateBudget)
	defer validateCancel()
	if err := checkValidateContext(validateCtx); err != nil {
		report.ValidateDuration = time.Since(validateStarted)
		report.markFailedStage(StageValidate, report.ValidateDuration)
		return Config{}, report, err
	}

	validationResult, err := validateConfig(validateCtx, k, &cfg, validationOptions{
		Strict:                opts.Strict,
		AdditionalUnknownKeys: metadata.sectionScalarOverrideKeys,
	})
	report.ValidateDuration = time.Since(validateStarted)
	report.UnknownKeyWarnings = validationResult.UnknownKeyWarnings
	if err != nil {
		report.FailedStage = StageValidate
		report.FailedStageDuration = report.ValidateDuration
		return Config{}, report, err
	}

	return cfg, report, nil
}

func (report *LoadReport) markFailedStage(stage string, duration time.Duration) {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = StageLoadDefaults
	}
	if duration <= 0 {
		duration = report.durationForStage(stage)
	}
	if duration <= 0 {
		duration = report.LoadDuration
	}
	if duration <= 0 {
		duration = time.Millisecond
	}
	report.FailedStage = stage
	report.FailedStageDuration = duration
}

func (report LoadReport) durationForStage(stage string) time.Duration {
	switch stage {
	case StageLoadDefaults:
		return report.LoadDefaultsDuration
	case StageLoadFile:
		return report.LoadFileDuration
	case StageLoadEnv:
		return report.LoadEnvDuration
	case StageParse:
		return report.ParseDuration
	case StageValidate:
		return report.ValidateDuration
	default:
		return 0
	}
}
