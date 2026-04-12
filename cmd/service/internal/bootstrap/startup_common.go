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

func recordStartupRejection(bootstrapSpan trace.Span, metrics *telemetry.Metrics, metricReason, errorType, failedStage string, err error) {
	if err != nil {
		bootstrapSpan.RecordError(err)
	}
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", errorType),
		attribute.String("failed.stage", failedStage),
	)
	metrics.IncStartupRejection(metricReason)
	metrics.IncConfigStartupOutcome("rejected")
}

func rejectStartupForPolicyViolation(
	ctx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
	dependency string,
	err error,
	extra ...any,
) error {
	dep := strings.ToLower(strings.TrimSpace(dependency))
	recordStartupRejection(bootstrapSpan, metrics, telemetry.StartupRejectionReasonPolicyViolation, "policy_violation", "startup.policy."+dep, err)
	args := startupLogArgs(
		ctx,
		startupLogComponentStartupProbes,
		dep+"_policy",
		"error",
		"error.type", "policy_violation",
		"dependency", dep,
		"err", err,
	)
	args = append(args, extra...)
	log.Error("startup_blocked", args...)
	return fmt.Errorf("%w: startup blocked by network policy: %w", errDependencyInit, err)
}

func rejectStartupForDependencyInit(
	ctx context.Context,
	bootstrapSpan trace.Span,
	metrics *telemetry.Metrics,
	log *slog.Logger,
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
	recordStartupRejection(bootstrapSpan, metrics, telemetry.StartupRejectionReasonDependencyInit, "dependency_init", failedStage, rejectErr)
	log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			dep+"_config",
			"error",
			"error.type", "dependency_init",
			"dependency", dep,
			"err", rejectErr,
		)...,
	)
	return rejectErr
}

func recordDependencyProbeRejection(
	ctx context.Context,
	runtime dependencyProbeRuntime,
	labels startupDependencyProbeLabels,
	mode string,
	err error,
) {
	dep := strings.ToLower(strings.TrimSpace(labels.dependency))
	if dep == "" {
		dep = "dependency"
	}
	operation := strings.TrimSpace(labels.operation)
	if operation == "" {
		operation = dep + "_probe"
	}
	stage := strings.TrimSpace(labels.probeStage)
	if stage == "" {
		stage = "startup.probe." + dep
	}

	recordStartupRejection(runtime.bootstrapSpan, runtime.metrics, telemetry.StartupRejectionReasonDependencyInit, "dependency_init", stage, err)

	args := startupLogArgs(
		ctx,
		startupLogComponentStartupProbes,
		operation,
		"error",
		"error.type", "dependency_init",
		"dependency", dep,
		"err", err,
	)
	if strings.TrimSpace(mode) != "" {
		args = append(args, "mode", strings.TrimSpace(mode))
	}
	runtime.log.Error("startup_blocked", args...)
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
		return telemetry.TelemetryFailureReasonDeadlineExceeded
	case errors.Is(err, context.Canceled):
		return telemetry.TelemetryFailureReasonCanceled
	default:
		return telemetry.TelemetryFailureReasonSetupError
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
