package httpapi

import (
	"context"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

// newProblem builds one of this example's generated Problem values from the
// shared catalog.
//
// Two things are worth copying. The code, title, and type URI come from
// internal/problem rather than a local table: the one this replaced fell through
// to the 500 type URI for a 409, and no test noticed because the status in the
// body was still 409. And the body carries the request identifier, so a domain
// error is as traceable as the transport rejections the template's own middleware
// writes.
func newProblem(ctx context.Context, code problem.Code, detail string) openapi.Problem {
	definition := problem.ForCodeOrInternal(code)

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
