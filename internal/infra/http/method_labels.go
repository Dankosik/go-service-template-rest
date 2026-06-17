package httpx

import (
	"net/http"
	"slices"
	"strings"
)

const otherHTTPMethodLabel = "OTHER"

// boundedHTTPMethods bounds client-visible Allow probing and observability label cardinality.
var boundedHTTPMethods = []string{
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
	if slices.Contains(boundedHTTPMethods, method) {
		return method
	}
	return otherHTTPMethodLabel
}
