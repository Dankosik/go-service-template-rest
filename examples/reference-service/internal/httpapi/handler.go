// Package httpapi maps the reference feature to its generated HTTP contract.
package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

// ArticleWriteScope is the permission a credential must carry to create an
// article. The composition root decides which credentials are granted it; this
// package decides which operation requires it.
const ArticleWriteScope = "articles:write"

var _ openapi.StrictServerInterface = (*handler)(nil)

type handler struct {
	articles *article.Service
}

// CreateArticle authorizes the caller, maps the generated request body onto the
// feature-owned draft, then maps each domain error identity onto its documented
// status. The use case never sees an HTTP type and this transport never invents a
// business rule.
//
// The authorization check is why the identity seam exists: the contract's
// security requirement already proved a credential, and
// reqctx.PrincipalFromContext reports who it proved, so this handler decides what
// that caller may do without reading the Authorization header a second time. An
// absent principal on an operation declaring `security:` is a wiring defect
// rather than an anonymous caller, so it is refused rather than challenged.
func (h *handler) CreateArticle(ctx context.Context, request openapi.CreateArticleRequestObject) (openapi.CreateArticleResponseObject, error) {
	principal, authenticated := reqctx.PrincipalFromContext(ctx)
	if !authenticated {
		return nil, errors.New("authenticated principal missing")
	}
	if !principal.HasScope(ArticleWriteScope) {
		return openapi.CreateArticle403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(
				newProblem(ctx, problem.CodeForbidden, "credentials do not permit this operation"),
			),
		}, nil
	}

	if request.Body == nil {
		return openapi.CreateArticle400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				newProblem(ctx, problem.CodeBadRequest, "request body is required"),
			),
		}, nil
	}

	created, err := h.articles.Create(ctx, article.Draft{
		Slug:    request.Body.Slug,
		Title:   request.Body.Title,
		Summary: request.Body.Summary,
	})
	if err != nil {
		// Returned rather than mapped here. The composition root installs
		// article.ClassifyError once for the whole feature, so the switch that used to
		// live in every operation exists in one place. Wrapping adds the
		// operation while preserving the sentinel identity errors.Is matches on;
		// an error the table does not recognize stays a 500, which is the honest
		// answer for a fault this service did not anticipate.
		return nil, fmt.Errorf("create article: %w", err)
	}

	return openapi.CreateArticle201JSONResponse{
		Headers: openapi.CreateArticle201ResponseHeaders{
			Location: "/api/v1/articles/" + created.Slug,
		},
		Body: openapi.Article{
			Slug:    created.Slug,
			Title:   created.Title,
			Summary: created.Summary,
		},
	}, nil
}

func (h *handler) GetArticle(ctx context.Context, request openapi.GetArticleRequestObject) (openapi.GetArticleResponseObject, error) {
	found, err := h.articles.Get(ctx, request.Slug)
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}

	return openapi.GetArticle200JSONResponse{
		Slug:    found.Slug,
		Title:   found.Title,
		Summary: found.Summary,
	}, nil
}
