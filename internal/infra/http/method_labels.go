package httpx

import (
	"net/http"
	"slices"
	"strings"
)

const otherHTTPMethodLabel = "OTHER"

var knownHTTPMethodLabels = []string{
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
	if slices.Contains(knownHTTPMethodLabels, method) {
		return method
	}
	return otherHTTPMethodLabel
}
