package otelconfig

import (
	"math"
	"testing"
)

func TestTraceSamplerVocabulary(t *testing.T) {
	t.Parallel()

	if got := TraceSamplerOrDefault("  "); got != DefaultTracesSampler {
		t.Fatalf("TraceSamplerOrDefault(empty) = %q, want %q", got, DefaultTracesSampler)
	}
	if got := TraceSamplerOrDefault(" ALWAYS_ON "); got != SamplerAlwaysOn {
		t.Fatalf("TraceSamplerOrDefault() = %q, want %q", got, SamplerAlwaysOn)
	}
	if !TraceSamplerSupported(" TRACEIDRATIO ") {
		t.Fatal("TraceSamplerSupported(TRACEIDRATIO) = false, want true")
	}
	if TraceSamplerSupported("sometimes") {
		t.Fatal("TraceSamplerSupported(sometimes) = true, want false")
	}
}

func TestTraceSamplerArgValidation(t *testing.T) {
	t.Parallel()

	for _, arg := range []float64{0, DefaultTracesSamplerArg, 1} {
		if !TraceSamplerArgFinite(arg) {
			t.Fatalf("TraceSamplerArgFinite(%v) = false, want true", arg)
		}
		if !TraceSamplerArgInRange(arg) {
			t.Fatalf("TraceSamplerArgInRange(%v) = false, want true", arg)
		}
	}

	for _, arg := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if TraceSamplerArgFinite(arg) {
			t.Fatalf("TraceSamplerArgFinite(%v) = true, want false", arg)
		}
		if TraceSamplerArgInRange(arg) {
			t.Fatalf("TraceSamplerArgInRange(%v) = true, want false", arg)
		}
	}

	for _, arg := range []float64{-0.1, 1.1} {
		if !TraceSamplerArgFinite(arg) {
			t.Fatalf("TraceSamplerArgFinite(%v) = false, want true", arg)
		}
		if TraceSamplerArgInRange(arg) {
			t.Fatalf("TraceSamplerArgInRange(%v) = true, want false", arg)
		}
	}
}
