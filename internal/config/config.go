package config

import (
	"context"
	"time"

	"github.com/knadh/koanf/v2"
)

// The load-pipeline stage names, and one transport scheme that has to be
// declared with them. secureTransportTLS is shared by grpc_config.go and
// messaging_config.go, each of which a build profile removes on its own, so it
// can live in neither; the minimal profile removes both, and there `unused`
// reports it unless it stays contiguous with a constant something still reads.
// Keep it in this block, with no blank line separating it.
const (
	StageLoadDefaults  = "config.load.defaults"
	StageLoadFile      = "config.load.file"
	StageLoadEnv       = "config.load.env"
	StageParse         = "config.parse"
	StageValidate      = "config.validate"
	secureTransportTLS = "tls"
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
func LoadJobsWorkerDetailedWithContext(ctx context.Context, opts LoadOptions) (Config, LoadReport, error) {
	return loadDetailedWithContext(ctx, opts, buildJobsWorkerSnapshot, validateJobsWorkerConfig)
}

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

	err = validate(
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
