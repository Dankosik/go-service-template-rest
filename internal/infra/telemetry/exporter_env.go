package telemetry

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/samber/lo"
)

// AmbientOTLPExporterEnv returns the sorted names of non-empty
// OTEL_EXPORTER_OTLP_* process variables. Endpoint variables among them may
// supply a destination when typed configuration does not; non-secret tuning
// remains under the official exporter's documented environment behavior.
// Values are never returned because they may contain credentials.
func AmbientOTLPExporterEnv() []string {
	var names []string
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OTEL_EXPORTER_OTLP_") && strings.TrimSpace(value) != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// sharedExporterEnvConflicts are the signal-agnostic standard OpenTelemetry
// exporter variables this service must not ignore when it configures its own
// exporter.
//
// The official exporters apply ambient environment first and explicit options
// second, so WithEndpointURL already makes an injected ENDPOINT or INSECURE
// harmless. Credential and trust material is different: this service sets no
// client certificate and no root CA pool, so these would travel to the
// collector unverified.
//
// This applies only when this service's own configuration named the
// destination. When the platform's own variables supplied it, the platform owns
// the whole exporter configuration.
var sharedExporterEnvConflicts = []string{
	"OTEL_EXPORTER_OTLP_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_HEADERS",
}

// traceExporterEnvConflicts adds the traces-specific credential and trust
// variables to sharedExporterEnvConflicts.
var traceExporterEnvConflicts = append(slices.Clone(sharedExporterEnvConflicts),
	"OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_TRACES_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_TRACES_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_TRACES_HEADERS",
)

// metricExporterEnvConflicts adds the metrics-specific credential and trust
// variables to sharedExporterEnvConflicts.
var metricExporterEnvConflicts = append(slices.Clone(sharedExporterEnvConflicts),
	"OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
)

// RejectedAmbientEnv returns the non-empty ambient exporter variables that
// exporter setup refuses for this endpoint rather than ignores, sorted. See
// sharedExporterEnvConflicts for why the endpoint and transport-tuning variables
// are deliberately absent.
//
// Only an endpoint this service configured rejects any: when the platform named
// the destination, the platform owns the credentials that come with it.
func (e ExporterEndpoint) RejectedAmbientEnv() []string {
	if !e.ConfiguredByService {
		return nil
	}
	switch e.signalPath {
	case otlpTracesPath:
		return conflictingEnv(traceExporterEnvConflicts)
	case otlpMetricsPath:
		return conflictingEnv(metricExporterEnvConflicts)
	default:
		return nil
	}
}

// conflictingEnv returns the non-empty variables among names, sorted so reported
// output is stable.
func conflictingEnv(names []string) []string {
	conflicting := lo.Filter(names, func(name string, _ int) bool { return nonEmptyEnv(name) })
	slices.Sort(conflicting)
	return conflicting
}

// rejectConflictingAmbientEnv refuses a configured exporter that would otherwise
// silently honor injected credentials or trust material.
//
// Both signals reject on the same terms, and the wording is the part worth
// owning here: an operator matching on this message should not have to discover
// that traces and metrics phrase the same refusal differently. Which variables
// count is the endpoint's own list.
func rejectConflictingAmbientEnv(endpoint ExporterEndpoint) error {
	conflicting := endpoint.RejectedAmbientEnv()
	if len(conflicting) == 0 {
		return nil
	}
	return fmt.Errorf(
		"unsupported ambient otel exporter environment (%s): injected credentials and trust material are not verifiable here; configure observability.otel.exporter.* instead",
		strings.Join(conflicting, ", "),
	)
}

func nonEmptyEnv(name string) bool {
	return strings.TrimSpace(os.Getenv(name)) != ""
}
