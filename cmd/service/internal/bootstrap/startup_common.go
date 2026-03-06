package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func recordConfigSuccessMetrics(metrics *telemetry.Metrics, report config.LoadReport) {
	if report.LoadDefaultsDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageLoadDefaults, "success", report.LoadDefaultsDuration)
	}
	if report.LoadFileDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageLoadFile, "success", report.LoadFileDuration)
	}
	if report.LoadEnvDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageLoadEnv, "success", report.LoadEnvDuration)
	}
	if report.ParseDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageParse, "success", report.ParseDuration)
	}
	if report.ValidateDuration > 0 {
		metrics.ObserveConfigLoadDuration(config.StageValidate, "success", report.ValidateDuration)
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

func rejectStartupForPolicyViolation(
	ctx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
	dependency string,
	err error,
) error {
	bootstrapSpan.RecordError(err)
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", "policy_violation"),
		attribute.String("failed.stage", "startup.policy."+strings.ToLower(strings.TrimSpace(dependency))),
	)
	metrics.IncConfigValidationFailure("policy_violation")
	metrics.IncConfigStartupOutcome("rejected")
	log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			"startup_probes",
			strings.ToLower(strings.TrimSpace(dependency))+"_policy",
			"error",
			"error.type", "policy_violation",
			"dependency", strings.ToLower(strings.TrimSpace(dependency)),
			"err", err,
		)...,
	)
	recordAdmissionFailure(ctx, deployTelemetry, "policy_violation", strings.ToLower(strings.TrimSpace(dependency)))
	return fmt.Errorf("%w: startup blocked by network policy: %w", config.ErrDependencyInit, err)
}

func rejectStartupForDependencyInit(
	ctx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	deployTelemetry *deployTelemetryRecorder,
	startupLifecycleStartedAt time.Time,
	dependency string,
	stage string,
	err error,
) error {
	dep := strings.ToLower(strings.TrimSpace(dependency))
	if dep == "" {
		dep = "dependency"
	}
	failedStage := strings.TrimSpace(stage)
	if failedStage == "" {
		failedStage = "startup.resolve." + dep
	}

	rejectErr := dependencyInitFailure(dep, err)
	bootstrapSpan.RecordError(rejectErr)
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", "dependency_init"),
		attribute.String("failed.stage", failedStage),
	)
	metrics.IncConfigValidationFailure("dependency_init")
	metrics.IncConfigStartupOutcome("rejected")
	log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			"startup_probes",
			dep+"_config",
			"error",
			"error.type", "dependency_init",
			"dependency", dep,
			"err", rejectErr,
		)...,
	)
	recordAdmissionFailure(ctx, deployTelemetry, "dependency_init", dep)
	return rejectErr
}

func recordAdmissionFailure(
	ctx context.Context,
	deployTelemetry *deployTelemetryRecorder,
	reasonClass string,
	probeType string,
) {
	if deployTelemetry == nil {
		return
	}

	deployTelemetry.RecordAdmission(ctx, "failure", reasonClass, probeType)
}

func startupLogArgs(ctx context.Context, component, operation, outcome string, extra ...any) []any {
	args := make([]any, 0, 6+len(extra))
	args = append(args,
		"component", component,
		"operation", operation,
		"outcome", outcome,
	)

	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		args = append(args,
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		)
	}

	args = append(args, extra...)
	return args
}

func telemetryInitFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "setup_error"
	}
}

func recordConfigStageSpan(tracer trace.Tracer, ctx context.Context, name string, duration time.Duration, result string, errorType string) {
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
