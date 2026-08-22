package httpclient

import (
	"maps"
	"net/http"

	"github.com/example/go-service-template-rest/internal/observability/correlationpolicy"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

var reservedPropagationHeader = correlationpolicy.Reserved(reqctx.RequestIDHeader)

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
		return reservedPropagationHeader(name)
	})
}
