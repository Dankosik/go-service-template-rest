package article

import (
	"errors"

	"github.com/example/go-service-template-rest/internal/failure"
)

var (
	ErrNotFound      = errors.New("article not found")
	ErrAlreadyExists = errors.New("article already exists")
	ErrInvalid       = errors.New("article is invalid")
)

// ClassifyError maps this feature's error identities once for every transport.
func ClassifyError(err error) (failure.Classification, bool) {
	switch {
	case errors.Is(err, ErrNotFound):
		return failure.Classification{Code: failure.CodeNotFound, Detail: "article was not found"}, true
	case errors.Is(err, ErrAlreadyExists):
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "an article with this slug already exists"}, true
	case errors.Is(err, ErrInvalid):
		return failure.Classification{Code: failure.CodeBadRequest, Detail: "request is malformed or invalid"}, true
	default:
		return failure.Classification{}, false
	}
}
