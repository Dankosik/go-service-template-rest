package bootstrap

import (
	"context"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type configStageDuration struct {
	stage    string
	duration time.Duration
}

const (
	startupConfigCompatibilityStage  = telemetry.ConfigLoadStageStartupCompatibility
	startupConfigCompatibilityReason = "startup_compatibility"
)

func configLoadStageDurations(report config.LoadReport) []configStageDuration {
	return []configStageDuration{
		{stage: telemetry.ConfigLoadStageLoadDefaults, duration: report.LoadDefaultsDuration},
		{stage: telemetry.ConfigLoadStageLoadFile, duration: report.LoadFileDuration},
		{stage: telemetry.ConfigLoadStageLoadEnv, duration: report.LoadEnvDuration},
		{stage: telemetry.ConfigLoadStageParse, duration: report.ParseDuration},
		{stage: telemetry.ConfigLoadStageValidate, duration: report.ValidateDuration},
	}
}

func recordConfigSuccessMetrics(metrics *telemetry.Metrics, report config.LoadReport) {
	for _, stage := range configLoadStageDurations(report) {
		if stage.duration > 0 {
			metrics.ObserveConfigLoadDuration(configLoadStageMetricLabel(stage.stage), telemetry.ConfigLoadResultSuccess, stage.duration)
		}
	}
}

func configLoadStageMetricLabel(stage string) string {
	switch strings.TrimSpace(stage) {
	case config.StageLoadDefaults:
		return telemetry.ConfigLoadStageLoadDefaults
	case config.StageLoadFile:
		return telemetry.ConfigLoadStageLoadFile
	case config.StageLoadEnv:
		return telemetry.ConfigLoadStageLoadEnv
	case config.StageParse:
		return telemetry.ConfigLoadStageParse
	case config.StageValidate:
		return telemetry.ConfigLoadStageValidate
	case telemetry.ConfigLoadStageStartupCompatibility:
		return telemetry.ConfigLoadStageStartupCompatibility
	default:
		return stage
	}
}

func startupRejectionReasonForConfigErrorType(errorType string) string {
	switch strings.TrimSpace(strings.ToLower(errorType)) {
	case config.ErrorTypeLoad:
		return telemetry.StartupRejectionReasonConfigLoad
	case config.ErrorTypeParse:
		return telemetry.StartupRejectionReasonConfigParse
	case config.ErrorTypeValidate:
		return telemetry.StartupRejectionReasonConfigValidate
	case config.ErrorTypeStrictUnknownKey:
		return telemetry.StartupRejectionReasonConfigStrictUnknownKey
	case config.ErrorTypeSecretPolicy:
		return telemetry.StartupRejectionReasonConfigSecretPolicy
	default:
		return telemetry.StartupRejectionReasonOther
	}
}

func failedStageDetails(report config.LoadReport) (string, time.Duration) {
	stage := strings.TrimSpace(report.FailedStage)
	if stage == "" {
		stage = config.StageLoadDefaults
	}
	duration := report.FailedStageDuration
	if duration <= 0 {
		duration = report.LoadDuration
	}
	if duration <= 0 {
		duration = time.Millisecond
	}
	return stage, duration
}

func recordConfigStageSpan(ctx context.Context, tracer trace.Tracer, name string, duration time.Duration, result string, errorType string) {
	if duration <= 0 {
		return
	}
	_, span := tracer.Start(ctx, name)
	attrs := []attribute.KeyValue{
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.String("result", result),
	}
	if strings.TrimSpace(errorType) != "" {
		attrs = append(attrs, attribute.String("error.type", errorType))
	}
	span.SetAttributes(attrs...)
	span.End()
}
