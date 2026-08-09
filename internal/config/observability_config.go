package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/example/go-service-template-rest/internal/observability/otelconfig"
)

func validateObservabilityConfig(cfg *ObservabilityConfig) error {
	cfg.Metrics.Addr = strings.TrimSpace(cfg.Metrics.Addr)
	if cfg.Metrics.Addr != "" {
		_, rawPort, err := net.SplitHostPort(cfg.Metrics.Addr)
		if err != nil {
			return fmt.Errorf("%w: observability.metrics.addr must be host:port", ErrValidate)
		}
		port, err := strconv.ParseUint(rawPort, 10, 16)
		if err != nil || port == 0 {
			return fmt.Errorf("%w: observability.metrics.addr port must be in range [1,65535]", ErrValidate)
		}
	}
	cfg.OTel.ServiceName = strings.TrimSpace(cfg.OTel.ServiceName)
	cfg.OTel.TracesSampler = strings.TrimSpace(cfg.OTel.TracesSampler)
	cfg.OTel.Exporter.OTLPEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPEndpoint)
	cfg.OTel.Exporter.OTLPMetricsEndpoint = strings.TrimSpace(cfg.OTel.Exporter.OTLPMetricsEndpoint)
	if cfg.OTel.ServiceName == "" {
		return fmt.Errorf("%w: observability.otel.service_name cannot be empty", ErrValidate)
	}
	return validateObservabilitySampler(cfg.OTel.TracesSampler, cfg.OTel.TracesSamplerArg)
}

// validateObservabilitySampler prefixes the shared sampler rules with the
// section that owns them, so a rejection names the setting an operator can edit.
func validateObservabilitySampler(sampler string, samplerArg float64) error {
	if err := otelconfig.ValidateTraceSampler(sampler, samplerArg); err != nil {
		return fmt.Errorf("%w: observability.otel.%w", ErrValidate, err)
	}
	return nil
}
