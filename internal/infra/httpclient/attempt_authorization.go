package httpclient

// profile:outbound-auth-http:start

import (
	"fmt"
	"net/http"
)

// AttemptAuthorizer authorizes one concrete attempt on a request copy.
type AttemptAuthorizer func(*http.Request) error

type attemptAuthorizerKey struct{}

type attemptAuthorizationTransport struct {
	base http.RoundTripper
}

func (t attemptAuthorizationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	authorize, ok := request.Context().Value(attemptAuthorizerKey{}).(AttemptAuthorizer)
	if !ok {
		return t.base.RoundTrip(request) //nolint:wrapcheck // The owned outer client adds operation context.
	}
	attempt := request.Clone(request.Context())
	if attempt.Header == nil {
		attempt.Header = make(http.Header)
	}
	if err := authorize(attempt); err != nil {
		return nil, &attemptAuthorizationError{err: err}
	}
	return t.base.RoundTrip(attempt) //nolint:wrapcheck // The owned outer client adds operation context.
}

type attemptAuthorizationError struct {
	err error
}

func (e *attemptAuthorizationError) Error() string {
	return fmt.Sprintf("authorize outbound HTTP attempt: %v", e.err)
}

func (e *attemptAuthorizationError) Unwrap() error {
	return e.err
}

// profile:outbound-auth-http:end
