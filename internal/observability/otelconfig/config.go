// Package otelconfig owns the trace sampler names this service accepts.
//
// It is a leaf on purpose. Both internal/config (which must reject an
// unsupported sampler at startup, before any exporter exists) and
// internal/infra/telemetry (which builds the sampler) need the same answer, and
// neither may import the other.
package otelconfig

import (
	"fmt"
	"math"
	"strings"
)

const (
	// SamplerAlwaysOn is the OTel always_on trace sampler name.
	SamplerAlwaysOn = "always_on"

	// SamplerAlwaysOff is the OTel always_off trace sampler name.
	SamplerAlwaysOff = "always_off"

	// SamplerTraceIDRatio is the OTel traceidratio sampler name.
	SamplerTraceIDRatio = "traceidratio"

	// SamplerParentBasedTraceIDRatio is the OTel parentbased_traceidratio sampler name.
	SamplerParentBasedTraceIDRatio = "parentbased_traceidratio"

	// DefaultTracesSampler is the repository default trace sampler name.
	DefaultTracesSampler = SamplerParentBasedTraceIDRatio

	// DefaultTracesSamplerArg is the repository default trace sampler ratio.
	DefaultTracesSamplerArg float64 = 0.10
)

// TraceSamplerOrDefault normalizes name, substituting the repository default
// when nothing was configured.
func TraceSamplerOrDefault(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return DefaultTracesSampler
	}
	return normalized
}

// ValidateTraceSampler reports whether a sampler name and its ratio argument are
// usable together. Both call sites previously repeated three separate predicates
// in the same order; one entry point keeps them from drifting.
//
// Messages name the settings rather than the Go parameters, so a caller can
// prefix its own section and still produce an operator-actionable key.
func ValidateTraceSampler(name string, arg float64) error {
	switch TraceSamplerOrDefault(name) {
	case SamplerAlwaysOn, SamplerAlwaysOff, SamplerTraceIDRatio, SamplerParentBasedTraceIDRatio:
	default:
		return fmt.Errorf("traces_sampler is unsupported")
	}
	if math.IsNaN(arg) || math.IsInf(arg, 0) {
		return fmt.Errorf("traces_sampler_arg must be finite")
	}
	if arg < 0 || arg > 1 {
		return fmt.Errorf("traces_sampler_arg must be in range [0,1]")
	}
	return nil
}
