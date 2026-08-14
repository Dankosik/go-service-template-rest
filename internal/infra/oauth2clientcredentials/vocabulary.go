package oauth2clientcredentials

// FailureClass is the closed, sanitized outbound-auth failure taxonomy.
type FailureClass string

const (
	maxConfiguredDependencies = 1
	maxDependencyNameBytes    = 64

	meterName                    = "service.outbound_auth"
	tokenResolutionsInstrument   = "outbound.auth.token.resolutions"
	providerAttemptsInstrument   = "outbound.auth.provider.attempts"
	providerDurationInstrument   = "outbound.auth.provider.attempt.duration"
	resourceRejectionsInstrument = "outbound.auth.resource.rejections"

	attributeDependency   = "outbound.auth.dependency"
	attributeSource       = "outbound.auth.source"
	attributeResult       = "outbound.auth.result"
	attributeFailureClass = "outbound.auth.failure_class"
	attributeTransport    = "outbound.auth.transport"

	sourceCache            = "cache"
	sourceAcquisition      = "acquisition"
	resultSuccess          = "success"
	resultFailure          = "failure"
	transportHTTP          = "http"
	transportGRPC          = "grpc"
	resultUnauthenticated  = "unauthenticated"
	resultForbidden        = "forbidden"
	eventMetricsDegraded   = "outbound_auth_metrics_degraded"
	eventConfigured        = "outbound_auth_configured"
	fieldComponent         = "component"
	fieldDependency        = "dependency"
	fieldReason            = "reason"
	componentOutboundAuth  = "outbound_auth"
	reasonInstrumentFailed = "instrument_unavailable"

	FailureInvalidConfiguration      FailureClass = "invalid_configuration"
	FailureEndpointTrust             FailureClass = "endpoint_trust"
	FailureCallerCanceled            FailureClass = "caller_canceled"
	FailureProviderTimeout           FailureClass = "provider_timeout"
	FailureProviderUnavailable       FailureClass = "provider_unavailable"
	FailureClientRejected            FailureClass = "client_rejected"
	FailureGrantRejected             FailureClass = "grant_rejected"
	FailureUnsupportedResponse       FailureClass = "unsupported_response"
	FailureTokenUnusable             FailureClass = "token_unusable"
	FailureDownstreamUnauthenticated FailureClass = "downstream_unauthenticated"
	FailureDownstreamForbidden       FailureClass = "downstream_forbidden"
)

var failureMessages = map[FailureClass]string{
	FailureInvalidConfiguration:      "outbound authentication configuration is invalid",
	FailureEndpointTrust:             "outbound authentication endpoint is not trusted",
	FailureCallerCanceled:            "outbound authentication caller canceled",
	FailureProviderTimeout:           "outbound authentication provider timed out",
	FailureProviderUnavailable:       "outbound authentication provider is unavailable",
	FailureClientRejected:            "outbound authentication client was rejected",
	FailureGrantRejected:             "outbound authentication grant was rejected",
	FailureUnsupportedResponse:       "outbound authentication provider response is unsupported",
	FailureTokenUnusable:             "outbound authentication token is unusable",
	FailureDownstreamUnauthenticated: "outbound dependency rejected authentication",
	FailureDownstreamForbidden:       "outbound dependency denied the operation",
}

func validFailureClass(class FailureClass) bool {
	_, ok := failureMessages[class]
	return ok
}
