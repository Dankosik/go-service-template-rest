// Package httpapi maps the reference feature to its generated HTTP contract.
package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
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
// The authorization check is why the identity seam exists. The contract's
// security requirement already proved a credential before this ran, and
// reqctx.PrincipalFromContext reports who it proved — so this handler decides
// what that caller may do without reading the Authorization header a second
// time. An absent principal on an operation that declares `security:` is a
// wiring defect rather than an anonymous caller, so it is refused rather than
// challenged.
func (h *handler) CreateArticle(ctx context.Context, request openapi.CreateArticleRequestObject) (openapi.CreateArticleResponseObject, error) {
	principal, authenticated := reqctx.PrincipalFromContext(ctx)
	if !authenticated || !principal.HasScope(ArticleWriteScope) {
		return openapi.CreateArticle403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusForbidden, "credentials do not permit this operation"),
			),
		}, nil
	}

	if request.Body == nil {
		return openapi.CreateArticle400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusBadRequest, "request body is required"),
			),
		}, nil
	}

	created, err := h.articles.Create(ctx, article.Draft{
		Slug:    request.Body.Slug,
		Title:   request.Body.Title,
		Summary: request.Body.Summary,
	})
	switch {
	case errors.Is(err, article.ErrAlreadyExists):
		return openapi.CreateArticle409ApplicationProblemPlusJSONResponse{
			ConflictApplicationProblemPlusJSONResponse: openapi.ConflictApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusConflict, "an article with this slug already exists"),
			),
		}, nil
	case errors.Is(err, article.ErrInvalid):
		// The message is the use case's own wording and contains no caller
		// input, so echoing it cannot reflect attacker-controlled content.
		return openapi.CreateArticle400ApplicationProblemPlusJSONResponse{
			BadRequestApplicationProblemPlusJSONResponse: openapi.BadRequestApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusBadRequest, "request is malformed or invalid"),
			),
		}, nil
	case err != nil:
		//nolint:nilerr // A handled domain failure is a valid typed HTTP response.
		return openapi.CreateArticle500ApplicationProblemPlusJSONResponse{
			InternalServerErrorApplicationProblemPlusJSONResponse: openapi.InternalServerErrorApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusInternalServerError, "request failed"),
			),
		}, nil
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
	if errors.Is(err, article.ErrNotFound) {
		return openapi.GetArticle404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: openapi.NotFoundApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusNotFound, "article was not found"),
			),
		}, nil
	}
	if err != nil {
		// The strict transport contract represents failures as typed responses;
		// returning err would bypass the intended API response.
		//nolint:nilerr // A handled domain failure is a valid typed HTTP response.
		return openapi.GetArticle500ApplicationProblemPlusJSONResponse{
			InternalServerErrorApplicationProblemPlusJSONResponse: openapi.InternalServerErrorApplicationProblemPlusJSONResponse(
				newProblem(ctx, http.StatusInternalServerError, "request failed"),
			),
		}, nil
	}

	return openapi.GetArticle200JSONResponse{
		Slug:    found.Slug,
		Title:   found.Title,
		Summary: found.Summary,
	}, nil
}
