package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

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
	metrics.IncConfigStartupOutcome(telemetry.ConfigStartupOutcomeRejected)
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
	if errors.Is(err, errDependencyInit) {
		return fmt.Errorf("startup blocked by network policy: %w", err)
	}
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
