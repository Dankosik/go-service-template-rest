package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type rolloutCorrelation struct {
	RolloutID    string
	DeploymentID string
	CIRunID      string
	CommitSHA    string
}

func rolloutCorrelationFromEnv() rolloutCorrelation {
	return rolloutCorrelation{
		RolloutID:    firstNonEmpty(os.Getenv("ROLLOUT_ID"), os.Getenv("RAILWAY_DEPLOYMENT_ID")),
		DeploymentID: firstNonEmpty(os.Getenv("DEPLOYMENT_ID"), os.Getenv("RAILWAY_DEPLOYMENT_ID")),
		CIRunID:      firstNonEmpty(os.Getenv("CI_RUN_ID"), os.Getenv("GITHUB_RUN_ID")),
		CommitSHA:    firstNonEmpty(os.Getenv("COMMIT_SHA"), os.Getenv("GITHUB_SHA")),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type deployTelemetryRecorder struct {
	log               *slog.Logger
	metrics           *telemetry.Metrics
	environment       string
	correlation       rolloutCorrelation
	admissionStarted  time.Time
	admissionRecorded bool
}

func newDeployTelemetryRecorder(log *slog.Logger, metrics *telemetry.Metrics, environment string) *deployTelemetryRecorder {
	if log == nil {
		log = slog.Default()
	}

	return &deployTelemetryRecorder{
		log:              log,
		metrics:          metrics,
		environment:      strings.TrimSpace(environment),
		correlation:      rolloutCorrelationFromEnv(),
		admissionStarted: time.Now(),
	}
}

func (r *deployTelemetryRecorder) SetLogger(log *slog.Logger) {
	if r == nil || log == nil {
		return
	}
	r.log = log
}

func (r *deployTelemetryRecorder) SetEnvironment(environment string) {
	if r == nil {
		return
	}
	r.environment = strings.TrimSpace(environment)
}

func (r *deployTelemetryRecorder) RecordAdmission(ctx context.Context, result, reasonClass, probeType string) {
	if r == nil || r.admissionRecorded {
		return
	}

	duration := time.Since(r.admissionStarted)
	r.metrics.ObserveDeployHealthAdmission(r.environment, result, reasonClass, duration)
	if strings.ToLower(strings.TrimSpace(result)) != "success" && strings.TrimSpace(probeType) != "" {
		r.metrics.IncDeployHealthProbeFailure(r.environment, probeType)
	}

	tracer := otel.Tracer("service.deploy")
	_, span := tracer.Start(ctx, "deploy.health.admission")
	r.setCorrelationSpanAttributes(span)
	span.SetAttributes(
		attribute.String("environment", safeEnvironment(r.environment)),
		attribute.String("result", strings.ToLower(strings.TrimSpace(result))),
		attribute.String("reason_class", strings.ToLower(strings.TrimSpace(reasonClass))),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)
	span.End()

	args := []any{
		"environment", safeEnvironment(r.environment),
		"result", strings.ToLower(strings.TrimSpace(result)),
		"reason_class", strings.ToLower(strings.TrimSpace(reasonClass)),
		"duration_ms", duration.Milliseconds(),
	}
	args = r.appendCorrelationLogArgs(args)

	if strings.EqualFold(strings.TrimSpace(result), "success") {
		r.log.Info("deploy_health_check", args...)
	} else {
		r.log.Error("deploy_health_check", args...)
	}

	r.admissionRecorded = true
}

func (r *deployTelemetryRecorder) RecordRollback(ctx context.Context, trigger, result, previousRevision string, duration time.Duration) {
	if r == nil {
		return
	}

	r.metrics.ObserveRollbackExecution(r.environment, trigger, result, duration)

	tracer := otel.Tracer("service.deploy")
	_, span := tracer.Start(ctx, "deploy.rollback.execute")
	r.setCorrelationSpanAttributes(span)
	span.SetAttributes(
		attribute.String("environment", safeEnvironment(r.environment)),
		attribute.String("trigger", strings.ToLower(strings.TrimSpace(trigger))),
		attribute.String("result", strings.ToLower(strings.TrimSpace(result))),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)
	if strings.TrimSpace(previousRevision) != "" {
		span.SetAttributes(attribute.String("previous_revision", strings.TrimSpace(previousRevision)))
	}
	span.End()

	args := []any{
		"environment", safeEnvironment(r.environment),
		"trigger", strings.ToLower(strings.TrimSpace(trigger)),
		"result", strings.ToLower(strings.TrimSpace(result)),
		"duration_ms", duration.Milliseconds(),
	}
	if strings.TrimSpace(previousRevision) != "" {
		args = append(args, "previous_revision", strings.TrimSpace(previousRevision))
	}
	args = r.appendCorrelationLogArgs(args)

	if strings.EqualFold(strings.TrimSpace(result), "success") {
		r.log.Info("rollback_execution", args...)
	} else {
		r.log.Error("rollback_execution", args...)
	}
}

func (r *deployTelemetryRecorder) RecordRollbackPostcheck(endpoint, result string) {
	if r == nil {
		return
	}
	r.metrics.IncRollbackPostcheck(r.environment, endpoint, result)
}

func (r *deployTelemetryRecorder) RecordConfigDriftDetected(ctx context.Context, source, driftID, configRevision string) {
	if r == nil {
		return
	}

	r.metrics.IncConfigDriftDetected(r.environment, source)
	r.metrics.SetConfigDriftOpen(r.environment, true)

	tracer := otel.Tracer("service.deploy")
	_, span := tracer.Start(ctx, "deploy.config_drift.check")
	r.setCorrelationSpanAttributes(span)
	span.SetAttributes(
		attribute.String("environment", safeEnvironment(r.environment)),
		attribute.String("state", "detected"),
		attribute.String("source", strings.ToLower(strings.TrimSpace(source))),
	)
	if strings.TrimSpace(driftID) != "" {
		span.SetAttributes(attribute.String("drift_id", strings.TrimSpace(driftID)))
	}
	if strings.TrimSpace(configRevision) != "" {
		span.SetAttributes(attribute.String("config_revision", strings.TrimSpace(configRevision)))
	}
	span.End()

	args := []any{
		"environment", safeEnvironment(r.environment),
		"source", strings.ToLower(strings.TrimSpace(source)),
	}
	if strings.TrimSpace(driftID) != "" {
		args = append(args, "drift_id", strings.TrimSpace(driftID))
	}
	if strings.TrimSpace(configRevision) != "" {
		args = append(args, "config_revision", strings.TrimSpace(configRevision))
	}
	args = r.appendCorrelationLogArgs(args)
	r.log.Warn("config_drift_detected", args...)
}

func (r *deployTelemetryRecorder) RecordConfigDriftReconciled(ctx context.Context, result, driftID, configRevision string, duration time.Duration) {
	if r == nil {
		return
	}

	r.metrics.ObserveConfigDriftReconcile(r.environment, result, duration)
	r.metrics.SetConfigDriftOpen(r.environment, false)

	tracer := otel.Tracer("service.deploy")
	_, span := tracer.Start(ctx, "deploy.config_drift.check")
	r.setCorrelationSpanAttributes(span)
	span.SetAttributes(
		attribute.String("environment", safeEnvironment(r.environment)),
		attribute.String("state", "reconciled"),
		attribute.String("result", strings.ToLower(strings.TrimSpace(result))),
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)
	if strings.TrimSpace(driftID) != "" {
		span.SetAttributes(attribute.String("drift_id", strings.TrimSpace(driftID)))
	}
	if strings.TrimSpace(configRevision) != "" {
		span.SetAttributes(attribute.String("config_revision", strings.TrimSpace(configRevision)))
	}
	span.End()

	args := []any{
		"environment", safeEnvironment(r.environment),
		"result", strings.ToLower(strings.TrimSpace(result)),
		"duration_ms", duration.Milliseconds(),
	}
	if strings.TrimSpace(driftID) != "" {
		args = append(args, "drift_id", strings.TrimSpace(driftID))
	}
	if strings.TrimSpace(configRevision) != "" {
		args = append(args, "config_revision", strings.TrimSpace(configRevision))
	}
	args = r.appendCorrelationLogArgs(args)

	if strings.EqualFold(strings.TrimSpace(result), "success") {
		r.log.Info("config_drift_reconciled", args...)
	} else {
		r.log.Error("config_drift_reconciled", args...)
	}
}

func (r *deployTelemetryRecorder) RecordNetworkIngressPolicyViolation(ctx context.Context, reasonClass, decision string) {
	if r == nil {
		return
	}
	r.recordNetworkPolicyViolation(ctx, "network_ingress_policy_violation", "ingress", reasonClass, decision)
}

func (r *deployTelemetryRecorder) RecordNetworkEgressPolicyViolation(ctx context.Context, reasonClass, decision string) {
	if r == nil {
		return
	}
	r.recordNetworkPolicyViolation(ctx, "network_egress_policy_violation", "egress", reasonClass, decision)
}

func (r *deployTelemetryRecorder) RecordNetworkExceptionStateChange(ctx context.Context, policyClass, state, decision, exceptionID string) {
	if r == nil {
		return
	}

	policy := strings.ToLower(strings.TrimSpace(policyClass))
	exceptionState := strings.ToLower(strings.TrimSpace(state))
	decisionValue := strings.ToLower(strings.TrimSpace(decision))
	active := exceptionState == "active"
	r.metrics.SetNetworkExceptionActive(r.environment, policy, active)

	tracer := otel.Tracer("service.deploy")
	_, span := tracer.Start(ctx, "security.network.policy")
	r.setCorrelationSpanAttributes(span)
	span.SetAttributes(
		attribute.String("environment", safeEnvironment(r.environment)),
		attribute.String("policy_class", policy),
		attribute.String("state", exceptionState),
		attribute.String("decision", decisionValue),
	)
	if strings.TrimSpace(exceptionID) != "" {
		span.SetAttributes(attribute.String("exception_id", strings.TrimSpace(exceptionID)))
	}
	span.End()

	args := []any{
		"environment", safeEnvironment(r.environment),
		"policy_class", policy,
		"state", exceptionState,
		"decision", decisionValue,
	}
	if strings.TrimSpace(exceptionID) != "" {
		args = append(args, "exception_id", strings.TrimSpace(exceptionID))
	}
	args = r.appendCorrelationLogArgs(args)
	r.log.Info("network_exception_state_change", args...)
}

func (r *deployTelemetryRecorder) recordNetworkPolicyViolation(ctx context.Context, eventName, policyClass, reasonClass, decision string) {
	policy := strings.ToLower(strings.TrimSpace(policyClass))
	reason := strings.ToLower(strings.TrimSpace(reasonClass))
	decisionValue := strings.ToLower(strings.TrimSpace(decision))

	r.metrics.IncNetworkPolicyViolation(r.environment, policy, reason)

	tracer := otel.Tracer("service.deploy")
	_, span := tracer.Start(ctx, "security.network.policy")
	r.setCorrelationSpanAttributes(span)
	span.SetAttributes(
		attribute.String("environment", safeEnvironment(r.environment)),
		attribute.String("policy_class", policy),
		attribute.String("reason_class", reason),
		attribute.String("decision", decisionValue),
	)
	span.End()

	args := []any{
		"environment", safeEnvironment(r.environment),
		"policy_class", policy,
		"reason_class", reason,
		"decision", decisionValue,
	}
	args = r.appendCorrelationLogArgs(args)
	r.log.Error(eventName, args...)
}

func (r *deployTelemetryRecorder) appendCorrelationLogArgs(args []any) []any {
	if r == nil {
		return args
	}
	if r.correlation.RolloutID != "" {
		args = append(args, "rollout_id", r.correlation.RolloutID)
	}
	if r.correlation.DeploymentID != "" {
		args = append(args, "deployment_id", r.correlation.DeploymentID)
	}
	if r.correlation.CIRunID != "" {
		args = append(args, "ci_run_id", r.correlation.CIRunID)
	}
	if r.correlation.CommitSHA != "" {
		args = append(args, "commit_sha", r.correlation.CommitSHA)
	}
	return args
}

func (r *deployTelemetryRecorder) setCorrelationSpanAttributes(span trace.Span) {
	if r == nil || span == nil {
		return
	}
	if r.correlation.RolloutID != "" {
		span.SetAttributes(attribute.String("rollout_id", r.correlation.RolloutID))
	}
	if r.correlation.DeploymentID != "" {
		span.SetAttributes(attribute.String("deployment_id", r.correlation.DeploymentID))
	}
	if r.correlation.CIRunID != "" {
		span.SetAttributes(attribute.String("ci_run_id", r.correlation.CIRunID))
	}
	if r.correlation.CommitSHA != "" {
		span.SetAttributes(attribute.String("commit_sha", r.correlation.CommitSHA))
	}
}

func safeEnvironment(environment string) string {
	trimmed := strings.TrimSpace(environment)
	if trimmed == "" {
		return "unknown"
	}
	return strings.ToLower(trimmed)
}
