package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// errUnresolvableAuthenticatedRequest reports that the validator handed the
// authentication seam no request to attach the principal to. It cannot happen
// with the validator this repository installs, and it fails closed rather than
// admitting an unattributed request if some future version changes that.
var errUnresolvableAuthenticatedRequest = errors.New("authenticated request is unavailable")

var errInvalidAuthenticatedPrincipal = errors.New("authenticated principal subject is empty")

// PrincipalResolver validates one security requirement and reports who the
// caller is. Returning an error rejects the request exactly as an
// openapi3filter.AuthenticationFunc would.
type PrincipalResolver func(context.Context, *openapi3filter.AuthenticationInput) (reqctx.Principal, error)

// Authenticated adapts a resolver to the AuthenticationFunc that
// RouterConfig.Authenticate takes, and is what makes the resolved caller
// reachable from a handler.
//
// openapi3filter.AuthenticationFunc returns only an error, so a service that
// wires it directly proves the credential and then discards the identity it just
// proved. What is left is re-reading Authorization inside every handler: two
// credential paths where the contract declared one, and the second is the one
// nobody audits.
//
// Publishing mutates the request in place because the validator hands the same
// request to the next handler. The test that asserts a handler observes the
// subject is what fails if that ever stops holding, rather than authorization
// silently going missing.
//
// A nil resolver returns nil, which leaves the seam unwired and every secured
// operation answering 401 — the same fail-closed default as setting
// RouterConfig.Authenticate to nil directly.
func Authenticated(resolve PrincipalResolver) openapi3filter.AuthenticationFunc {
	if resolve == nil {
		return nil
	}

	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		principal, err := resolve(ctx, input)
		if err != nil {
			return err
		}
		if strings.TrimSpace(principal.Subject) == "" {
			return errInvalidAuthenticatedPrincipal
		}

		request := authenticatedRequest(input)
		if request == nil {
			return errUnresolvableAuthenticatedRequest
		}
		setPrincipal(request, principal)
		return nil
	}
}

func setPrincipal(r *http.Request, principal reqctx.Principal) {
	*r = *r.WithContext(reqctx.ContextWithPrincipal(r.Context(), principal))
}

func authenticatedRequest(input *openapi3filter.AuthenticationInput) *http.Request {
	if input == nil || input.RequestValidationInput == nil {
		return nil
	}
	return input.RequestValidationInput.Request
}
