//nolint:dupl // OpenAPI response wrappers intentionally mirror generated status-specific response types.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Dankosik/billing-service/internal/api"
)

type ProblemError struct {
	Status int
	Title  string
	Detail string
}

func NewProblemError(status int, title, detail string) ProblemError {
	return ProblemError{Status: status, Title: title, Detail: detail}
}

func (e ProblemError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.Status, e.Title, e.Detail)
}

func problemFromError(ctx context.Context, err error) (int, api.Problem) {
	var problemErr ProblemError
	if !errors.As(err, &problemErr) {
		problemErr = NewProblemError(http.StatusInternalServerError, "internal server error", "request failed")
	}
	status := problemErr.Status
	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	title := problemErr.Title
	if title == "" {
		title = http.StatusText(status)
	}
	detail := problemErr.Detail
	if detail == "" {
		detail = "request failed"
	}
	problem := api.Problem{
		Detail:    optionalProblemString(detail),
		RequestId: nil,
		Status:    problemHTTPStatus(status),
		Title:     title,
		Type:      "about:blank",
	}
	if ctx != nil {
		problem.RequestId = optionalProblemString(requestIDFromContext(ctx))
	}
	return status, problem
}

func issueMicroleaseProblemResponse(ctx context.Context, err error) api.IssueMicroleaseResponseObject {
	status, problem := problemFromError(ctx, err)
	switch status {
	case http.StatusBadRequest:
		return api.IssueMicrolease400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: api.BadRequestApplicationProblemPlusJSONResponse(problem)}
	case http.StatusUnauthorized:
		return api.IssueMicrolease401ApplicationProblemPlusJSONResponse{UnauthorizedApplicationProblemPlusJSONResponse: api.UnauthorizedApplicationProblemPlusJSONResponse(problem)}
	case http.StatusForbidden:
		return api.IssueMicrolease403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: api.ForbiddenApplicationProblemPlusJSONResponse(problem)}
	case http.StatusConflict:
		return api.IssueMicrolease409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: api.ConflictApplicationProblemPlusJSONResponse(problem)}
	case http.StatusUnprocessableEntity:
		return api.IssueMicrolease422ApplicationProblemPlusJSONResponse{UnprocessableEntityApplicationProblemPlusJSONResponse: api.UnprocessableEntityApplicationProblemPlusJSONResponse(problem)}
	case http.StatusLocked:
		return api.IssueMicrolease423ApplicationProblemPlusJSONResponse{LockedApplicationProblemPlusJSONResponse: api.LockedApplicationProblemPlusJSONResponse(problem)}
	case http.StatusTooManyRequests:
		return api.IssueMicrolease429ApplicationProblemPlusJSONResponse{TooManyRequestsApplicationProblemPlusJSONResponse: api.TooManyRequestsApplicationProblemPlusJSONResponse(problem)}
	case http.StatusServiceUnavailable:
		return api.IssueMicrolease503ApplicationProblemPlusJSONResponse{ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(problem)}
	default:
		return api.IssueMicrolease500ApplicationProblemPlusJSONResponse{InternalServerErrorApplicationProblemPlusJSONResponse: api.InternalServerErrorApplicationProblemPlusJSONResponse(problem)}
	}
}

func readMicroleaseProblemResponse(ctx context.Context, err error) api.ReadMicroleaseResponseObject {
	status, problem := problemFromError(ctx, err)
	switch status {
	case http.StatusBadRequest:
		return api.ReadMicrolease400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: api.BadRequestApplicationProblemPlusJSONResponse(problem)}
	case http.StatusUnauthorized:
		return api.ReadMicrolease401ApplicationProblemPlusJSONResponse{UnauthorizedApplicationProblemPlusJSONResponse: api.UnauthorizedApplicationProblemPlusJSONResponse(problem)}
	case http.StatusForbidden:
		return api.ReadMicrolease403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: api.ForbiddenApplicationProblemPlusJSONResponse(problem)}
	case http.StatusServiceUnavailable:
		return api.ReadMicrolease503ApplicationProblemPlusJSONResponse{ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(problem)}
	default:
		return api.ReadMicrolease500ApplicationProblemPlusJSONResponse{InternalServerErrorApplicationProblemPlusJSONResponse: api.InternalServerErrorApplicationProblemPlusJSONResponse(problem)}
	}
}

func closeMicroleaseProblemResponse(ctx context.Context, err error) api.CloseMicroleaseResponseObject {
	status, problem := problemFromError(ctx, err)
	switch status {
	case http.StatusBadRequest:
		return api.CloseMicrolease400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: api.BadRequestApplicationProblemPlusJSONResponse(problem)}
	case http.StatusUnauthorized:
		return api.CloseMicrolease401ApplicationProblemPlusJSONResponse{UnauthorizedApplicationProblemPlusJSONResponse: api.UnauthorizedApplicationProblemPlusJSONResponse(problem)}
	case http.StatusForbidden:
		return api.CloseMicrolease403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: api.ForbiddenApplicationProblemPlusJSONResponse(problem)}
	case http.StatusConflict:
		return api.CloseMicrolease409ApplicationProblemPlusJSONResponse{ConflictApplicationProblemPlusJSONResponse: api.ConflictApplicationProblemPlusJSONResponse(problem)}
	case http.StatusUnprocessableEntity:
		return api.CloseMicrolease422ApplicationProblemPlusJSONResponse{UnprocessableEntityApplicationProblemPlusJSONResponse: api.UnprocessableEntityApplicationProblemPlusJSONResponse(problem)}
	case http.StatusLocked:
		return api.CloseMicrolease423ApplicationProblemPlusJSONResponse{LockedApplicationProblemPlusJSONResponse: api.LockedApplicationProblemPlusJSONResponse(problem)}
	case http.StatusServiceUnavailable:
		return api.CloseMicrolease503ApplicationProblemPlusJSONResponse{ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(problem)}
	default:
		return api.CloseMicrolease500ApplicationProblemPlusJSONResponse{InternalServerErrorApplicationProblemPlusJSONResponse: api.InternalServerErrorApplicationProblemPlusJSONResponse(problem)}
	}
}

func readBillingOperationProblemResponse(ctx context.Context, err error) api.ReadBillingOperationResponseObject {
	status, problem := problemFromError(ctx, err)
	switch status {
	case http.StatusBadRequest:
		return api.ReadBillingOperation400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse: api.BadRequestApplicationProblemPlusJSONResponse(problem)}
	case http.StatusUnauthorized:
		return api.ReadBillingOperation401ApplicationProblemPlusJSONResponse{UnauthorizedApplicationProblemPlusJSONResponse: api.UnauthorizedApplicationProblemPlusJSONResponse(problem)}
	case http.StatusForbidden:
		return api.ReadBillingOperation403ApplicationProblemPlusJSONResponse{ForbiddenApplicationProblemPlusJSONResponse: api.ForbiddenApplicationProblemPlusJSONResponse(problem)}
	case http.StatusServiceUnavailable:
		return api.ReadBillingOperation503ApplicationProblemPlusJSONResponse{ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(problem)}
	default:
		return api.ReadBillingOperation500ApplicationProblemPlusJSONResponse{InternalServerErrorApplicationProblemPlusJSONResponse: api.InternalServerErrorApplicationProblemPlusJSONResponse(problem)}
	}
}
