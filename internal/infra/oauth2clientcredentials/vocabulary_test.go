package oauth2clientcredentials

import (
	"slices"
	"testing"
)

func TestFailureVocabularyAndPrecedence(t *testing.T) {
	t.Parallel()
	if maxConfiguredDependencies != 1 || maxDependencyNameBytes != 64 {
		t.Fatalf("dependency bounds = %d, %d", maxConfiguredDependencies, maxDependencyNameBytes)
	}
	literals := map[string]struct {
		got  string
		want string
	}{
		"meter":                {meterName, "service.outbound_auth"},
		"token resolutions":    {tokenResolutionsInstrument, "outbound.auth.token.resolutions"},
		"provider attempts":    {providerAttemptsInstrument, "outbound.auth.provider.attempts"},
		"provider duration":    {providerDurationInstrument, "outbound.auth.provider.attempt.duration"},
		"resource rejections":  {resourceRejectionsInstrument, "outbound.auth.resource.rejections"},
		"dependency attribute": {attributeDependency, "outbound.auth.dependency"},
		"source attribute":     {attributeSource, "outbound.auth.source"},
		"result attribute":     {attributeResult, "outbound.auth.result"},
		"failure attribute":    {attributeFailureClass, "outbound.auth.failure_class"},
		"transport attribute":  {attributeTransport, "outbound.auth.transport"},
		"metrics warning":      {eventMetricsDegraded, "outbound_auth_metrics_degraded"},
		"configured event":     {eventConfigured, "outbound_auth_configured"},
		"component field":      {fieldComponent, "component"},
		"dependency field":     {fieldDependency, "dependency"},
		"reason field":         {fieldReason, "reason"},
		"component value":      {componentOutboundAuth, "outbound_auth"},
		"degradation reason":   {reasonInstrumentFailed, "instrument_unavailable"},
	}
	for name, literal := range literals {
		if literal.got != literal.want {
			t.Fatalf("%s = %q, want %q", name, literal.got, literal.want)
		}
	}
	for name, values := range map[string]struct {
		got  []string
		want []string
	}{
		"source":    {[]string{sourceCache, sourceAcquisition}, []string{"cache", "acquisition"}},
		"result":    {[]string{resultSuccess, resultFailure}, []string{"success", "failure"}},
		"transport": {[]string{transportHTTP, transportGRPC}, []string{"http", "grpc"}},
		"rejection": {[]string{resultUnauthenticated, resultForbidden}, []string{"unauthenticated", "forbidden"}},
	} {
		if !slices.Equal(values.got, values.want) {
			t.Fatalf("%s values = %q, want %q", name, values.got, values.want)
		}
	}

	want := []FailureClass{
		FailureInvalidConfiguration,
		FailureEndpointTrust,
		FailureCallerCanceled,
		FailureProviderTimeout,
		FailureProviderUnavailable,
		FailureClientRejected,
		FailureGrantRejected,
		FailureUnsupportedResponse,
		FailureTokenUnusable,
		FailureDownstreamUnauthenticated,
		FailureDownstreamForbidden,
	}
	wantSpelling := []string{
		"invalid_configuration",
		"endpoint_trust",
		"caller_canceled",
		"provider_timeout",
		"provider_unavailable",
		"client_rejected",
		"grant_rejected",
		"unsupported_response",
		"token_unusable",
		"downstream_unauthenticated",
		"downstream_forbidden",
	}
	if len(failureMessages) != len(want) {
		t.Fatalf("failure class count = %d, want %d", len(failureMessages), len(want))
	}
	for index, class := range want {
		if string(class) != wantSpelling[index] || !validFailureClass(class) || failureMessages[class] == "" {
			t.Fatalf("failure class %d = %q, valid=%t, message=%q", index, class, validFailureClass(class), failureMessages[class])
		}
	}
	if validFailureClass("unknown") {
		t.Fatal("unknown failure class is valid")
	}
}
