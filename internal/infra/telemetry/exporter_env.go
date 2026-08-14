package telemetry

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/samber/lo"
)

// AmbientOTLPExporterEnv returns the sorted names of non-empty
// OTEL_EXPORTER_OTLP_* process variables. The endpoint variables among them are
// honored when this service names no endpoint of its own; the rest are not read.
// Callers subtract whatever supplied the endpoint and report the remainder, so
// an injected setting never looks effective when it is not.
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

// traceExporterEnvConflicts are the standard OpenTelemetry exporter variables
// this service must not ignore when it configures its own exporter.
//
// otlptracehttp applies ambient environment first and explicit options second, so
// WithEndpointURL already makes an injected ENDPOINT or INSECURE harmless.
// Credential and trust material is different: this service sets no client
// certificate and no root CA pool, so these would travel to the collector
// unverified.
//
// This applies only when observability.otel.exporter.otlp_endpoint named the
// destination. When the platform's own variables supplied it, the platform owns
// the whole exporter configuration.
//
// Kept sorted so reported output is stable.
var traceExporterEnvConflicts = []string{
	"OTEL_EXPORTER_OTLP_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_HEADERS",
	"OTEL_EXPORTER_OTLP_TRACES_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_TRACES_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_TRACES_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_TRACES_HEADERS",
}

// metricExporterEnvConflicts are the ambient credential and trust variables a
// configured metrics exporter must not silently honor; see
// traceExporterEnvConflicts. Kept sorted so reported output is stable.
var metricExporterEnvConflicts = []string{
	"OTEL_EXPORTER_OTLP_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_HEADERS",
	"OTEL_EXPORTER_OTLP_METRICS_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_METRICS_CLIENT_CERTIFICATE",
	"OTEL_EXPORTER_OTLP_METRICS_CLIENT_KEY",
	"OTEL_EXPORTER_OTLP_METRICS_HEADERS",
}

// ConflictingTraceExporterEnv returns the non-empty ambient exporter variables
// that a configured exporter cannot safely ignore. See
// traceExporterEnvConflicts for why the endpoint and transport-tuning variables
// are deliberately absent.
func ConflictingTraceExporterEnv() []string {
	return conflictingEnv(traceExporterEnvConflicts)
}

// ConflictingMetricExporterEnv returns the non-empty ambient exporter variables a
// configured metrics exporter cannot safely ignore.
func ConflictingMetricExporterEnv() []string {
	return conflictingEnv(metricExporterEnvConflicts)
}

func rejectConflictingTraceExporterEnv() error {
	return rejectConflictingAmbientEnv(traceExporterEnvConflicts)
}

func rejectConflictingMetricExporterEnv() error {
	return rejectConflictingAmbientEnv(metricExporterEnvConflicts)
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
// count is still each signal's own list.
func rejectConflictingAmbientEnv(names []string) error {
	conflicting := conflictingEnv(names)
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
