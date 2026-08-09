package httpx

import (
	"net/http"
	"slices"
)

// healthProbeRoutePaths are the platform probe routes owned by this template's
// OpenAPI contract. Orchestrators poll them every few seconds for the life of
// the pod, which is why the access log and the admission middlewares each treat
// them differently from contract traffic; each states its own reason.
var healthProbeRoutePaths = []string{"/health/live", "/health/ready"}

// isHealthProbeRequest matches probe routes by raw path rather than by routed
// template, because the admission middlewares run before the generated router
// has matched anything. AccessLog matches the same set on the routed template
// instead; see skipHealthProbeLog.
func isHealthProbeRequest(r *http.Request) bool {
	if r == nil || r.Method != http.MethodGet {
		return false
	}
	return slices.Contains(healthProbeRoutePaths, r.URL.Path)
}
