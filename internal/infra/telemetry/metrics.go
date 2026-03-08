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

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
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
