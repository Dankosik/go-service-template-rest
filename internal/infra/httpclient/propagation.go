package httpclient

import (
	"maps"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
)

type propagationSanitizer struct {
	base http.RoundTripper
}

func (t propagationSanitizer) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (t propagationSanitizer) RoundTrip(request *http.Request) (*http.Response, error) {
	attempt := request.Clone(request.Context())
	attempt.Header = request.Header.Clone()
	attempt.Trailer = request.Trailer.Clone()
	removeReservedHeaders(attempt.Header)
	removeReservedHeaders(attempt.Trailer)
	return t.base.RoundTrip(attempt) //nolint:wrapcheck // The transport error keeps its standard identity.
}

func removeReservedHeaders(header http.Header) {
	maps.DeleteFunc(header, func(name string, _ []string) bool {
		return strings.EqualFold(name, "accept-encoding") ||
			strings.EqualFold(name, "traceparent") ||
			strings.EqualFold(name, "tracestate") ||
			strings.EqualFold(name, "baggage") ||
			strings.EqualFold(name, reqctx.RequestIDHeader)
	})
}
