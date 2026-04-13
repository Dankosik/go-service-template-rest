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
	configFailures           *prometheus.CounterVec
	startupRejections        *prometheus.CounterVec
	telemetryInitFailures    *prometheus.CounterVec
	configUnknownKeyWarnings prometheus.Counter
	configStartupOutcome     *prometheus.CounterVec
	startupDependencyStatus  *prometheus.GaugeVec
}

const (
	// ConfigLoadResultSuccess is the bounded label for successful config load stages.
	ConfigLoadResultSuccess = "success"

	// ConfigLoadResultError is the bounded label for failed config load stages.
	ConfigLoadResultError = "error"

	// ConfigLoadResultOther is the bounded fallback label for unknown config load results.
	ConfigLoadResultOther = "other"
)

const (
	// ConfigLoadStageLoadDefaults is the bounded label for config default loading.
	ConfigLoadStageLoadDefaults = "config.load.defaults"

	// ConfigLoadStageLoadFile is the bounded label for config file loading.
	ConfigLoadStageLoadFile = "config.load.file"

	// ConfigLoadStageLoadEnv is the bounded label for config environment loading.
	ConfigLoadStageLoadEnv = "config.load.env"

	// ConfigLoadStageParse is the bounded label for config parsing.
	ConfigLoadStageParse = "config.parse"

	// ConfigLoadStageValidate is the bounded label for config validation.
	ConfigLoadStageValidate = "config.validate"

	// ConfigLoadStageStartupCompatibility is the bounded label for startup compatibility validation.
	ConfigLoadStageStartupCompatibility = "startup.config.compatibility"

	// ConfigLoadStageOther is the bounded fallback label for unknown config load stages.
	ConfigLoadStageOther = "other"
)

const (
	// ConfigFailureReasonLoad is the bounded label for config load failures.
	ConfigFailureReasonLoad = "load"

	// ConfigFailureReasonParse is the bounded label for config parse failures.
	ConfigFailureReasonParse = "parse"

	// ConfigFailureReasonValidate is the bounded label for config validation failures.
	ConfigFailureReasonValidate = "validate"

	// ConfigFailureReasonStrictUnknownKey is the bounded label for strict unknown-key failures.
	ConfigFailureReasonStrictUnknownKey = "strict_unknown_key"

	// ConfigFailureReasonSecretPolicy is the bounded label for secret policy failures.
	ConfigFailureReasonSecretPolicy = "secret_policy"

	// ConfigFailureReasonOther is the bounded fallback label for unknown config failures.
	ConfigFailureReasonOther = "other"
)

const (
	// ConfigStartupOutcomeReady is the bounded label for startup admission-ready outcomes.
	ConfigStartupOutcomeReady = "ready"

	// ConfigStartupOutcomeRejected is the bounded label for rejected startup outcomes.
	ConfigStartupOutcomeRejected = "rejected"

	// ConfigStartupOutcomeOther is the bounded fallback label for unknown startup outcomes.
	ConfigStartupOutcomeOther = "other"
)

const (
	// TelemetryFailureReasonSetupError is the bounded label for generic tracing setup failures.
	TelemetryFailureReasonSetupError = "setup_error"

	// TelemetryFailureReasonDeadlineExceeded is the bounded label for tracing setup deadline failures.
	TelemetryFailureReasonDeadlineExceeded = "deadline_exceeded"

	// TelemetryFailureReasonCanceled is the bounded label for tracing setup cancellation.
	TelemetryFailureReasonCanceled = "canceled"

	// TelemetryFailureReasonOther is the bounded fallback label for unknown tracing setup failures.
	TelemetryFailureReasonOther = "other"
)

const (
	// StartupRejectionReasonConfigLoad is the bounded startup rejection label for config load failures.
	StartupRejectionReasonConfigLoad = "config_load"

	// StartupRejectionReasonConfigParse is the bounded startup rejection label for config parse failures.
	StartupRejectionReasonConfigParse = "config_parse"

	// StartupRejectionReasonConfigValidate is the bounded startup rejection label for config validation failures.
	StartupRejectionReasonConfigValidate = "config_validate"

	// StartupRejectionReasonConfigStartupCompatibility is the bounded startup rejection label for bootstrap config compatibility failures.
	StartupRejectionReasonConfigStartupCompatibility = "config_startup_compatibility"

	// StartupRejectionReasonConfigStrictUnknownKey is the bounded startup rejection label for strict unknown key failures.
	StartupRejectionReasonConfigStrictUnknownKey = "config_strict_unknown_key"

	// StartupRejectionReasonConfigSecretPolicy is the bounded startup rejection label for secret policy failures.
	StartupRejectionReasonConfigSecretPolicy = "config_secret_policy"

	// StartupRejectionReasonPolicyViolation is the bounded startup rejection label for startup policy failures.
	StartupRejectionReasonPolicyViolation = "policy_violation"

	// StartupRejectionReasonDependencyInit is the bounded startup rejection label for dependency initialization failures.
	StartupRejectionReasonDependencyInit = "dependency_init"

	// StartupRejectionReasonStartupError is the bounded startup rejection label for HTTP startup failures.
	StartupRejectionReasonStartupError = "startup_error"

	// StartupRejectionReasonOther is the bounded fallback startup rejection label.
	StartupRejectionReasonOther = "other"
)

const (
	// StartupDependencyPostgres is the bounded dependency label for Postgres.
	StartupDependencyPostgres = "postgres"

	// StartupDependencyRedis is the bounded dependency label for Redis.
	StartupDependencyRedis = "redis"

	// StartupDependencyMongo is the bounded dependency label for MongoDB.
	StartupDependencyMongo = "mongo"

	// StartupDependencyTelemetry is the bounded dependency label for telemetry.
	StartupDependencyTelemetry = "telemetry"

	// StartupDependencyNetworkPolicy is the bounded dependency label for network policy.
	StartupDependencyNetworkPolicy = "network_policy"

	// StartupDependencyIngressPolicy is the bounded dependency label for ingress policy.
	StartupDependencyIngressPolicy = "ingress_policy"

	// StartupDependencyMetricsExposure is the bounded dependency label for metrics exposure policy.
	StartupDependencyMetricsExposure = "metrics_exposure"

	// StartupDependencyEgressException is the bounded dependency label for egress exception policy.
	StartupDependencyEgressException = "egress_exception"

	// StartupDependencyOther is the bounded fallback label for unknown startup dependencies.
	StartupDependencyOther = "other"
)

const (
	// StartupDependencyModeDisabled is the bounded mode label for disabled dependencies.
	StartupDependencyModeDisabled = "disabled"

	// StartupDependencyModeCriticalFailClosed is the bounded mode label for critical fail-closed dependencies.
	StartupDependencyModeCriticalFailClosed = "critical_fail_closed"

	// StartupDependencyModeCriticalFailDegraded is the bounded mode label for critical degraded dependencies.
	StartupDependencyModeCriticalFailDegraded = "critical_fail_degraded"

	// StartupDependencyModeOptionalFailOpen is the bounded mode label for optional fail-open dependencies.
	StartupDependencyModeOptionalFailOpen = "optional_fail_open"

	// StartupDependencyModeFeatureOff is the bounded mode label for admitted feature-off dependencies.
	StartupDependencyModeFeatureOff = "feature_off"

	// StartupDependencyModeDegradedReadOnlyOrStale is the bounded mode label for degraded read-only or stale dependencies.
	StartupDependencyModeDegradedReadOnlyOrStale = "degraded_read_only_or_stale"

	// StartupDependencyModeOther is the bounded fallback label for unknown dependency modes.
	StartupDependencyModeOther = "other"
)

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

	configFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_failures_total",
			Help: "Total number of config failures by bounded reason.",
		},
		[]string{"reason"},
	)

	startupRejections := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "startup_rejections_total",
			Help: "Total number of startup rejections by bounded reason.",
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

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		requestsTotal,
		requestDuration,
		configLoadDuration,
		configFailures,
		startupRejections,
		telemetryInitFailures,
		configUnknownKeyWarnings,
		configStartupOutcome,
		startupDependencyStatus,
	)

	return &Metrics{
		registry:                 registry,
		requestsTotal:            requestsTotal,
		requestDuration:          requestDuration,
		configLoadDuration:       configLoadDuration,
		configFailures:           configFailures,
		startupRejections:        startupRejections,
		telemetryInitFailures:    telemetryInitFailures,
		configUnknownKeyWarnings: configUnknownKeyWarnings,
		configStartupOutcome:     configStartupOutcome,
		startupDependencyStatus:  startupDependencyStatus,
	}
}

func (m *Metrics) ObserveHTTPRequest(method, route string, statusCode int) {
	if m == nil || m.requestsTotal == nil {
		return
	}
	m.requestsTotal.WithLabelValues(method, route, strconv.Itoa(statusCode)).Inc()
}

func (m *Metrics) ObserveHTTPRequestDuration(method, route string, statusCode int, duration time.Duration) {
	if m == nil || m.requestDuration == nil {
		return
	}
	m.requestDuration.WithLabelValues(method, route, strconv.Itoa(statusCode)).Observe(duration.Seconds())
}

func (m *Metrics) ObserveConfigLoadDuration(stage, result string, duration time.Duration) {
	if m == nil || m.configLoadDuration == nil {
		return
	}
	m.configLoadDuration.WithLabelValues(normalizeConfigLoadStage(stage), normalizeConfigLoadResult(result)).Observe(duration.Seconds())
}

func (m *Metrics) IncConfigFailure(reason string) {
	if m == nil || m.configFailures == nil {
		return
	}
	m.configFailures.WithLabelValues(normalizeConfigFailureReason(reason)).Inc()
}

func (m *Metrics) IncStartupRejection(reason string) {
	if m == nil || m.startupRejections == nil {
		return
	}
	m.startupRejections.WithLabelValues(normalizeStartupRejectionReason(reason)).Inc()
}

func (m *Metrics) AddConfigUnknownKeyWarnings(count int) {
	if m == nil || m.configUnknownKeyWarnings == nil || count <= 0 {
		return
	}
	m.configUnknownKeyWarnings.Add(float64(count))
}

func (m *Metrics) IncTelemetryInitFailure(reason string) {
	if m == nil || m.telemetryInitFailures == nil {
		return
	}
	m.telemetryInitFailures.WithLabelValues(normalizeTelemetryFailureReason(reason)).Inc()
}

func (m *Metrics) IncConfigStartupOutcome(outcome string) {
	if m == nil || m.configStartupOutcome == nil {
		return
	}
	m.configStartupOutcome.WithLabelValues(normalizeConfigStartupOutcome(outcome)).Inc()
}

func (m *Metrics) MarkStartupDependencyReady(dep, mode string) {
	m.setStartupDependencyStatus(dep, mode, 1)
}

func (m *Metrics) MarkStartupDependencyBlocked(dep, mode string) {
	m.setStartupDependencyStatus(dep, mode, 0)
}

func (m *Metrics) setStartupDependencyStatus(dep, mode string, value float64) {
	if m == nil || m.startupDependencyStatus == nil {
		return
	}
	m.startupDependencyStatus.WithLabelValues(normalizeStartupDependency(dep), normalizeStartupDependencyMode(mode)).Set(value)
}

func (m *Metrics) Handler() http.Handler {
	if m == nil || m.registry == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func normalizeTelemetryFailureReason(reason string) string {
	normalized := strings.TrimSpace(strings.ToLower(reason))
	switch normalized {
	case TelemetryFailureReasonSetupError, TelemetryFailureReasonDeadlineExceeded, TelemetryFailureReasonCanceled:
		return normalized
	default:
		return TelemetryFailureReasonOther
	}
}

func normalizeConfigLoadResult(result string) string {
	normalized := strings.TrimSpace(strings.ToLower(result))
	switch normalized {
	case ConfigLoadResultSuccess:
		return ConfigLoadResultSuccess
	case ConfigLoadResultError:
		return ConfigLoadResultError
	default:
		return ConfigLoadResultOther
	}
}

func normalizeConfigLoadStage(stage string) string {
	normalized := strings.TrimSpace(strings.ToLower(stage))
	switch normalized {
	case ConfigLoadStageLoadDefaults,
		ConfigLoadStageLoadFile,
		ConfigLoadStageLoadEnv,
		ConfigLoadStageParse,
		ConfigLoadStageValidate,
		ConfigLoadStageStartupCompatibility:
		return normalized
	default:
		return ConfigLoadStageOther
	}
}

func normalizeConfigFailureReason(reason string) string {
	normalized := strings.TrimSpace(strings.ToLower(reason))
	switch normalized {
	case ConfigFailureReasonLoad:
		return ConfigFailureReasonLoad
	case ConfigFailureReasonParse:
		return ConfigFailureReasonParse
	case ConfigFailureReasonValidate:
		return ConfigFailureReasonValidate
	case ConfigFailureReasonStrictUnknownKey:
		return ConfigFailureReasonStrictUnknownKey
	case ConfigFailureReasonSecretPolicy:
		return ConfigFailureReasonSecretPolicy
	default:
		return ConfigFailureReasonOther
	}
}

func normalizeConfigStartupOutcome(outcome string) string {
	normalized := strings.TrimSpace(strings.ToLower(outcome))
	switch normalized {
	case ConfigStartupOutcomeReady:
		return ConfigStartupOutcomeReady
	case ConfigStartupOutcomeRejected:
		return ConfigStartupOutcomeRejected
	default:
		return ConfigStartupOutcomeOther
	}
}

func normalizeStartupRejectionReason(reason string) string {
	normalized := strings.TrimSpace(strings.ToLower(reason))
	switch normalized {
	case StartupRejectionReasonConfigLoad:
		return StartupRejectionReasonConfigLoad
	case StartupRejectionReasonConfigParse:
		return StartupRejectionReasonConfigParse
	case StartupRejectionReasonConfigValidate:
		return StartupRejectionReasonConfigValidate
	case StartupRejectionReasonConfigStartupCompatibility:
		return StartupRejectionReasonConfigStartupCompatibility
	case StartupRejectionReasonConfigStrictUnknownKey:
		return StartupRejectionReasonConfigStrictUnknownKey
	case StartupRejectionReasonConfigSecretPolicy:
		return StartupRejectionReasonConfigSecretPolicy
	case StartupRejectionReasonPolicyViolation:
		return StartupRejectionReasonPolicyViolation
	case StartupRejectionReasonDependencyInit:
		return StartupRejectionReasonDependencyInit
	case StartupRejectionReasonStartupError:
		return StartupRejectionReasonStartupError
	default:
		return StartupRejectionReasonOther
	}
}

func normalizeStartupDependency(dep string) string {
	normalized := strings.TrimSpace(strings.ToLower(dep))
	switch normalized {
	case StartupDependencyPostgres,
		StartupDependencyRedis,
		StartupDependencyMongo,
		StartupDependencyTelemetry,
		StartupDependencyNetworkPolicy,
		StartupDependencyIngressPolicy,
		StartupDependencyMetricsExposure,
		StartupDependencyEgressException:
		return normalized
	default:
		return StartupDependencyOther
	}
}

func normalizeStartupDependencyMode(mode string) string {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	switch normalized {
	case StartupDependencyModeDisabled,
		StartupDependencyModeCriticalFailClosed,
		StartupDependencyModeCriticalFailDegraded,
		StartupDependencyModeOptionalFailOpen,
		StartupDependencyModeFeatureOff,
		StartupDependencyModeDegradedReadOnlyOrStale:
		return normalized
	default:
		return StartupDependencyModeOther
	}
}
