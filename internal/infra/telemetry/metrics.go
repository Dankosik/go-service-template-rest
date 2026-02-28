package telemetry

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry      *prometheus.Registry
	requestsTotal *prometheus.CounterVec
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

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		requestsTotal,
	)

	return &Metrics{
		registry:      registry,
		requestsTotal: requestsTotal,
	}
}

func (m *Metrics) ObserveHTTPRequest(method, route string, statusCode int) {
	if m == nil {
		return
	}
	m.requestsTotal.WithLabelValues(method, route, strconv.Itoa(statusCode)).Inc()
}

func (m *Metrics) Handler() http.Handler {
	if m == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
