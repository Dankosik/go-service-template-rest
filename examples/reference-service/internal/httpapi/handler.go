// Package httpapi maps the reference feature to its generated HTTP contract.
package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
)

var _ openapi.StrictServerInterface = (*handler)(nil)

type handler struct {
	articles *article.Service
}

func (h *handler) GetArticle(ctx context.Context, request openapi.GetArticleRequestObject) (openapi.GetArticleResponseObject, error) {
	found, err := h.articles.Get(ctx, request.Slug)
	if errors.Is(err, article.ErrNotFound) {
		return openapi.GetArticle404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: openapi.NotFoundApplicationProblemPlusJSONResponse(
				problem("not_found", "not found", http.StatusNotFound, "article was not found"),
			),
		}, nil
	}
	if err != nil {
		// The strict transport contract represents failures as typed responses;
		// returning err would bypass the intended API response.
		//nolint:nilerr // A handled domain failure is a valid typed HTTP response.
		return openapi.GetArticle500ApplicationProblemPlusJSONResponse{
			InternalServerErrorApplicationProblemPlusJSONResponse: openapi.InternalServerErrorApplicationProblemPlusJSONResponse(
				problem("internal_error", "internal server error", http.StatusInternalServerError, "request failed"),
			),
		}, nil
	}

	return openapi.GetArticle200JSONResponse{
		Slug:    found.Slug,
		Title:   found.Title,
		Summary: found.Summary,
	}, nil
}
