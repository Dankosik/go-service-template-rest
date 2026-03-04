package telemetry

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry                 *prometheus.Registry
	requestsTotal            *prometheus.CounterVec
	requestDuration          *prometheus.HistogramVec
	configLoadDuration       *prometheus.HistogramVec
	configValidationFailures *prometheus.CounterVec
	telemetryInitFailures    *prometheus.CounterVec
	configUnknownKeyWarnings prometheus.Counter
	configStartupOutcome     *prometheus.CounterVec
	startupDependencyStatus  *prometheus.GaugeVec
	deployHealthAdmission    *prometheus.CounterVec
	deployHealthDuration     *prometheus.HistogramVec
	deployHealthProbeFailure *prometheus.CounterVec
	rollbackExecution        *prometheus.CounterVec
	rollbackRecoveryDuration *prometheus.HistogramVec
	rollbackPostcheck        *prometheus.CounterVec
	configDriftDetected      *prometheus.CounterVec
	configDriftOpen          *prometheus.GaugeVec
	configDriftReconcile     *prometheus.HistogramVec
	networkPolicyViolation   *prometheus.CounterVec
	networkExceptionActive   *prometheus.GaugeVec
}

func New() *Metrics {
	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "route", "status_code"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10},
		},
		[]string{"method", "route", "status_code"},
	)

	configLoadDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "config_load_duration_seconds",
			Help:    "Configuration lifecycle stage duration in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"stage", "result"},
	)

	configValidationFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_validation_failures_total",
			Help: "Total number of config validation failures by reason.",
		},
		[]string{"reason"},
	)

	telemetryInitFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telemetry_init_failure_total",
			Help: "Total number of telemetry initialization failures by reason.",
		},
		[]string{"reason"},
	)

	configUnknownKeyWarnings := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "config_unknown_key_warnings_total",
			Help: "Total number of unknown key warnings when strict mode is disabled.",
		},
	)

	configStartupOutcome := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_startup_outcome_total",
			Help: "Total startup outcomes for config bootstrap lifecycle.",
		},
		[]string{"outcome"},
	)

	startupDependencyStatus := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "startup_dependency_status",
			Help: "Status of startup dependency initialization by dependency and mode (1=ready, 0=blocked).",
		},
		[]string{"dep", "mode"},
	)

	deployHealthAdmission := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deploy_health_admission_total",
			Help: "Deploy health admission checks by environment, result, and reason class.",
		},
		[]string{"environment", "result", "reason_class"},
	)

	deployHealthDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "deploy_health_admission_duration_seconds",
			Help:    "Duration of deploy health admission checks in seconds.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120, 180, 300},
		},
		[]string{"environment", "result"},
	)

	deployHealthProbeFailure := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deploy_health_probe_failures_total",
			Help: "Deploy health probe failures by environment and probe type.",
		},
		[]string{"environment", "probe_type"},
	)

	rollbackExecution := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rollback_execution_total",
			Help: "Rollback executions by environment, trigger, and result.",
		},
		[]string{"environment", "trigger", "result"},
	)

	rollbackRecoveryDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rollback_recovery_duration_seconds",
			Help:    "Rollback recovery duration in seconds.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"environment", "result"},
	)

	rollbackPostcheck := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rollback_postcheck_total",
			Help: "Rollback post-check results by environment and endpoint.",
		},
		[]string{"environment", "endpoint", "result"},
	)

	configDriftDetected := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_drift_detected_total",
			Help: "Config drift detections by environment and source.",
		},
		[]string{"environment", "source"},
	)

	configDriftOpen := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "config_drift_open",
			Help: "Open config drift flag by environment (1=open, 0=closed).",
		},
		[]string{"environment"},
	)

	configDriftReconcile := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "config_drift_reconcile_duration_seconds",
			Help:    "Config drift reconciliation duration in seconds.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 300, 1800, 3600, 14400, 86400},
		},
		[]string{"environment", "result"},
	)

	networkPolicyViolation := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "network_policy_violation_total",
			Help: "Network policy violations by environment, policy class, and reason class.",
		},
		[]string{"environment", "policy_class", "reason_class"},
	)

	networkExceptionActive := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_exception_active",
			Help: "Active network exception state by environment and policy class (1=active, 0=inactive).",
		},
		[]string{"environment", "policy_class"},
	)

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		requestsTotal,
		requestDuration,
		configLoadDuration,
		configValidationFailures,
		telemetryInitFailures,
		configUnknownKeyWarnings,
		configStartupOutcome,
		startupDependencyStatus,
		deployHealthAdmission,
		deployHealthDuration,
		deployHealthProbeFailure,
		rollbackExecution,
		rollbackRecoveryDuration,
		rollbackPostcheck,
		configDriftDetected,
		configDriftOpen,
		configDriftReconcile,
		networkPolicyViolation,
		networkExceptionActive,
	)

	return &Metrics{
		registry:                 registry,
		requestsTotal:            requestsTotal,
		requestDuration:          requestDuration,
		configLoadDuration:       configLoadDuration,
		configValidationFailures: configValidationFailures,
		telemetryInitFailures:    telemetryInitFailures,
		configUnknownKeyWarnings: configUnknownKeyWarnings,
		configStartupOutcome:     configStartupOutcome,
		startupDependencyStatus:  startupDependencyStatus,
		deployHealthAdmission:    deployHealthAdmission,
		deployHealthDuration:     deployHealthDuration,
		deployHealthProbeFailure: deployHealthProbeFailure,
		rollbackExecution:        rollbackExecution,
		rollbackRecoveryDuration: rollbackRecoveryDuration,
		rollbackPostcheck:        rollbackPostcheck,
		configDriftDetected:      configDriftDetected,
		configDriftOpen:          configDriftOpen,
		configDriftReconcile:     configDriftReconcile,
		networkPolicyViolation:   networkPolicyViolation,
		networkExceptionActive:   networkExceptionActive,
	}
}

func (m *Metrics) ObserveHTTPRequest(method, route string, statusCode int) {
	if m == nil {
		return
	}
	m.requestsTotal.WithLabelValues(method, route, strconv.Itoa(statusCode)).Inc()
}

func (m *Metrics) ObserveHTTPRequestDuration(method, route string, statusCode int, duration time.Duration) {
	if m == nil {
		return
	}
	m.requestDuration.WithLabelValues(method, route, strconv.Itoa(statusCode)).Observe(duration.Seconds())
}

func (m *Metrics) ObserveConfigLoadDuration(stage, result string, duration time.Duration) {
	if m == nil {
		return
	}
	m.configLoadDuration.WithLabelValues(stage, result).Observe(duration.Seconds())
}

func (m *Metrics) IncConfigValidationFailure(reason string) {
	if m == nil {
		return
	}
	m.configValidationFailures.WithLabelValues(reason).Inc()
}

func (m *Metrics) AddConfigUnknownKeyWarnings(count int) {
	if m == nil || count <= 0 {
		return
	}
	m.configUnknownKeyWarnings.Add(float64(count))
}

func (m *Metrics) IncTelemetryInitFailure(reason string) {
	if m == nil {
		return
	}
	m.telemetryInitFailures.WithLabelValues(normalizeTelemetryFailureReason(reason)).Inc()
}

func (m *Metrics) IncConfigStartupOutcome(outcome string) {
	if m == nil {
		return
	}
	m.configStartupOutcome.WithLabelValues(outcome).Inc()
}

func (m *Metrics) SetStartupDependencyStatus(dep, mode string, ready bool) {
	if m == nil {
		return
	}
	value := 0.0
	if ready {
		value = 1.0
	}
	m.startupDependencyStatus.WithLabelValues(dep, mode).Set(value)
}

func (m *Metrics) ObserveDeployHealthAdmission(environment, result, reasonClass string, duration time.Duration) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	res := normalizeResultLabel(result)
	reason := normalizeReasonClassLabel(reasonClass)
	m.deployHealthAdmission.WithLabelValues(env, res, reason).Inc()
	m.deployHealthDuration.WithLabelValues(env, res).Observe(duration.Seconds())
}

func (m *Metrics) IncDeployHealthProbeFailure(environment, probeType string) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	probe := normalizeProbeTypeLabel(probeType)
	m.deployHealthProbeFailure.WithLabelValues(env, probe).Inc()
}

func (m *Metrics) ObserveRollbackExecution(environment, trigger, result string, duration time.Duration) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	t := normalizeTriggerLabel(trigger)
	res := normalizeResultLabel(result)
	m.rollbackExecution.WithLabelValues(env, t, res).Inc()
	m.rollbackRecoveryDuration.WithLabelValues(env, res).Observe(duration.Seconds())
}

func (m *Metrics) IncRollbackPostcheck(environment, endpoint, result string) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	ep := normalizeEndpointLabel(endpoint)
	res := normalizeResultLabel(result)
	m.rollbackPostcheck.WithLabelValues(env, ep, res).Inc()
}

func (m *Metrics) IncConfigDriftDetected(environment, source string) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	src := normalizeDriftSourceLabel(source)
	m.configDriftDetected.WithLabelValues(env, src).Inc()
}

func (m *Metrics) SetConfigDriftOpen(environment string, open bool) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	value := 0.0
	if open {
		value = 1.0
	}
	m.configDriftOpen.WithLabelValues(env).Set(value)
}

func (m *Metrics) ObserveConfigDriftReconcile(environment, result string, duration time.Duration) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	res := normalizeResultLabel(result)
	m.configDriftReconcile.WithLabelValues(env, res).Observe(duration.Seconds())
}

func (m *Metrics) IncNetworkPolicyViolation(environment, policyClass, reasonClass string) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	policy := normalizePolicyClassLabel(policyClass)
	reason := normalizeNetworkReasonClassLabel(reasonClass)
	m.networkPolicyViolation.WithLabelValues(env, policy, reason).Inc()
}

func (m *Metrics) SetNetworkExceptionActive(environment, policyClass string, active bool) {
	if m == nil {
		return
	}
	env := normalizeEnvironmentLabel(environment)
	policy := normalizePolicyClassLabel(policyClass)
	value := 0.0
	if active {
		value = 1.0
	}
	m.networkExceptionActive.WithLabelValues(env, policy).Set(value)
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func normalizeFieldGroupLabel(fieldGroup string) string {
	normalized := strings.ToLower(strings.TrimSpace(fieldGroup))
	if normalized == "" {
		return "other"
	}
	if separator := strings.Index(normalized, "."); separator > 0 {
		normalized = normalized[:separator]
	}

	switch normalized {
	case "app", "http", "log", "observability", "postgres", "redis", "mongo", "feature_flags":
		return normalized
	default:
		return "other"
	}
}

func normalizeTelemetryFailureReason(reason string) string {
	normalized := strings.TrimSpace(strings.ToLower(reason))
	switch normalized {
	case "setup_error", "deadline_exceeded", "canceled":
		return normalized
	default:
		return "other"
	}
}

func normalizeEnvironmentLabel(environment string) string {
	normalized := strings.ToLower(strings.TrimSpace(environment))
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func normalizeResultLabel(result string) string {
	normalized := strings.ToLower(strings.TrimSpace(result))
	switch normalized {
	case "success", "failure", "skipped":
		return normalized
	default:
		return "other"
	}
}

func normalizeReasonClassLabel(reasonClass string) string {
	normalized := strings.ToLower(strings.TrimSpace(reasonClass))
	switch normalized {
	case "ready", "readiness", "timeout", "probe_failure", "ci_failed", "ci_unknown", "dependency_init", "startup_error", "policy_violation":
		return normalized
	default:
		return "other"
	}
}

func normalizeProbeTypeLabel(probeType string) string {
	normalized := strings.ToLower(strings.TrimSpace(probeType))
	switch normalized {
	case "readiness", "postgres", "redis", "mongo", "startup":
		return normalized
	default:
		return "other"
	}
}

func normalizeTriggerLabel(trigger string) string {
	normalized := strings.ToLower(strings.TrimSpace(trigger))
	switch normalized {
	case "admission_failed", "runtime_error", "shutdown_error", "manual":
		return normalized
	default:
		return "other"
	}
}

func normalizeEndpointLabel(endpoint string) string {
	normalized := strings.ToLower(strings.TrimSpace(endpoint))
	switch normalized {
	case "/health/live", "/health/ready":
		return normalized
	default:
		return "other"
	}
}

func normalizeDriftSourceLabel(source string) string {
	normalized := strings.ToLower(strings.TrimSpace(source))
	switch normalized {
	case "ci", "runtime":
		return normalized
	default:
		return "other"
	}
}

func normalizePolicyClassLabel(policyClass string) string {
	normalized := strings.ToLower(strings.TrimSpace(policyClass))
	switch normalized {
	case "ingress", "egress":
		return normalized
	default:
		return "other"
	}
}

func normalizeNetworkReasonClassLabel(reasonClass string) string {
	normalized := strings.ToLower(strings.TrimSpace(reasonClass))
	switch normalized {
	case "missing_exception", "missing_metadata", "expired_exception", "public_target_denied", "scheme_denied", "invalid_configuration":
		return normalized
	default:
		return "other"
	}
}
