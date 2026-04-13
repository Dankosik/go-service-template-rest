package httpx

import (
	"net/http"
	"slices"
	"strings"
)

const otherHTTPMethodLabel = "OTHER"

// routePolicyHTTPMethods bounds path probing for client-visible Allow behavior.
var routePolicyHTTPMethods = []string{
	http.MethodConnect,
	http.MethodGet,
	http.MethodHead,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodPatch,
	http.MethodPost,
	http.MethodPut,
	http.MethodTrace,
}

// metricHTTPMethodLabels bounds observability label cardinality.
var metricHTTPMethodLabels = []string{
	http.MethodConnect,
	http.MethodGet,
	http.MethodHead,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodPatch,
	http.MethodPost,
	http.MethodPut,
	http.MethodTrace,
}

func normalizeHTTPMethodLabel(method string) string {
	method = strings.TrimSpace(method)
	if slices.Contains(metricHTTPMethodLabels, method) {
		return method
	}
	return otherHTTPMethodLabel
}
