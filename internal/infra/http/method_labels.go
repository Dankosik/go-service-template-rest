package httpx

import (
	"net/http"
	"slices"
	"strings"
)

const otherHTTPMethodLabel = "OTHER"

// boundedHTTPMethods bounds client-visible Allow probing and observability label
// cardinality.
//
// CONNECT and TRACE are deliberately absent. chi cannot route CONNECT from an
// ordinary handler and oapi-codegen never generates a TRACE operation, so
// probing for them costs a router match per 405 and can only ever answer no —
// while advertising TRACE in an Allow header is a finding in most scanners.
// A request using either still gets a bounded OTHER metric label below.
var boundedHTTPMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodPatch,
	http.MethodPost,
	http.MethodPut,
}

func normalizeHTTPMethodLabel(method string) string {
	method = strings.TrimSpace(method)
	if slices.Contains(boundedHTTPMethods, method) {
		return method
	}
	return otherHTTPMethodLabel
}
