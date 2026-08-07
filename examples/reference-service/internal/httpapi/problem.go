package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

// ClassifyError maps this feature's error identities onto the problems a client
// sees, once, for every operation.
//
// It lives beside the operations rather than in the composition root so it is
// tested with them: the router tests below drive a real request through a real
// use case and assert the status, which is the only thing that catches a drifted
// mapping. The root installs it, because the transport that consumes it is an
// adapter this package must not import.
//
// The details are this service's own wording. Passing err.Error() through would
// put use-case internals, and eventually a dependency's message, in a response
// body.
func ClassifyError(err error) (problem.Mapped, bool) {
	switch {
	case errors.Is(err, article.ErrNotFound):
		return problem.Mapped{Code: problem.CodeNotFound, Detail: "article was not found"}, true
	case errors.Is(err, article.ErrAlreadyExists):
		return problem.Mapped{Code: problem.CodeConflict, Detail: "an article with this slug already exists"}, true
	case errors.Is(err, article.ErrInvalid):
		return problem.Mapped{Code: problem.CodeBadRequest, Detail: "request is malformed or invalid"}, true
	default:
		// An unclassified error is a fault this service did not anticipate. The
		// transport answers 500 for it, which is the honest status: guessing a
		// friendlier one is how a bug starts reading as a client mistake.
		return problem.Mapped{}, false
	}
}

// newProblem builds one of this example's generated Problem values from the
// shared catalog.
//
// Two things are worth copying. The code, title, and type URI come from
// internal/problem rather than a local table: the one this replaced fell through
// to the 500 type URI for a 409, and no test noticed because the status in the
// body was still 409. And the body carries the request identifier, so a domain
// error is as traceable as the transport rejections the template's own middleware
// writes.
func newProblem(ctx context.Context, status int, detail string) openapi.Problem {
	definition, ok := problem.For(status)
	if !ok {
		// A status the shared catalog does not publish is a contract defect in
		// this package rather than a runtime condition. Answering with the
		// internal-error envelope is the only honest option left: it must not
		// claim the status that was asked for, because nothing describes it.
		definition, _ = problem.For(http.StatusInternalServerError)
	}

	built := openapi.Problem{
		Code:   string(definition.Code),
		Detail: &detail,
		Status: int32(definition.Status), // #nosec G115 -- catalog entries are fixed HTTP status constants.
		Title:  definition.Title,
		Type:   definition.TypeURI,
	}
	if requestID := reqctx.RequestID(ctx); requestID != "" {
		built.RequestId = &requestID
	}
	return built
}
