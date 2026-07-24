package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func recordStartupRejection(bootstrapSpan trace.Span, errorType, failedStage string, err error) {
	if err != nil {
		bootstrapSpan.RecordError(err)
	}
	bootstrapSpan.SetAttributes(
		attribute.String("result", "error"),
		attribute.String("error.type", errorType),
		attribute.String("failed.stage", failedStage),
	)
}

func rejectStartupForPolicyViolation(
	ctx context.Context,
	bootstrapSpan trace.Span,
	log *slog.Logger,
	dependency string,
	err error,
	extra ...any,
) error {
	dep := strings.ToLower(strings.TrimSpace(dependency))
	recordStartupRejection(bootstrapSpan, "policy_violation", "startup.policy."+dep, err)
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

func rejectPostgresStartupForDependencyInit(
	ctx context.Context,
	bootstrapSpan trace.Span,
	log *slog.Logger,
	err error,
) error {
	rejectErr := postgresDependencyInitFailure(err)
	recordStartupRejection(bootstrapSpan, "dependency_init", startupPostgresResolveStage, rejectErr)
	log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			"postgres_config",
			"error",
			"error.type", "dependency_init",
			"dependency", startupDependencyPostgres,
			"err", rejectErr,
		)...,
	)
	return rejectErr
}

func recordDependencyProbeRejection(ctx context.Context, runtime postgresStartupRuntime, err error) {
	recordStartupRejection(runtime.bootstrapSpan, "dependency_init", startupPostgresProbeStage, err)
	runtime.log.Error(
		"startup_blocked",
		startupLogArgs(
			ctx,
			startupLogComponentStartupProbes,
			startupPostgresProbeOperation,
			"error",
			"error.type", "dependency_init",
			"dependency", startupDependencyPostgres,
			"err", err,
		)...,
	)
}
