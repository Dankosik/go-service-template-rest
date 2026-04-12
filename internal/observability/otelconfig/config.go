package otelconfig

import (
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

	// OTLPProtocolHTTPProtobuf is the supported OTLP HTTP protobuf protocol value.
	OTLPProtocolHTTPProtobuf = "http/protobuf"

	// DefaultOTLPProtocol is the repository default OTLP exporter protocol.
	DefaultOTLPProtocol = OTLPProtocolHTTPProtobuf
)

func NormalizeTraceSampler(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func TraceSamplerOrDefault(name string) string {
	normalized := NormalizeTraceSampler(name)
	if normalized == "" {
		return DefaultTracesSampler
	}
	return normalized
}

func TraceSamplerSupported(name string) bool {
	switch NormalizeTraceSampler(name) {
	case SamplerAlwaysOn, SamplerAlwaysOff, SamplerTraceIDRatio, SamplerParentBasedTraceIDRatio:
		return true
	default:
		return false
	}
}

func TraceSamplerArgFinite(arg float64) bool {
	return !math.IsNaN(arg) && !math.IsInf(arg, 0)
}

func TraceSamplerArgInRange(arg float64) bool {
	return TraceSamplerArgFinite(arg) && arg >= 0 && arg <= 1
}

func NormalizeOTLPProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func OTLPProtocolOrDefault(protocol string) string {
	normalized := NormalizeOTLPProtocol(protocol)
	if normalized == "" {
		return DefaultOTLPProtocol
	}
	return normalized
}

func OTLPProtocolSupported(protocol string) bool {
	return NormalizeOTLPProtocol(protocol) == OTLPProtocolHTTPProtobuf
}
