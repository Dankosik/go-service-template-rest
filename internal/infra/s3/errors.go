package s3

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/smithy-go"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

type bodyTooLargeError struct{}

func (*bodyTooLargeError) Error() string {
	return "S3 response body exceeds its configured limit"
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func stableError(err error) error {
	if _, ok := errors.AsType[*bodyTooLargeError](err); ok {
		return objectstorage.NewError(objectstorage.KindTooLarge)
	}
	return objectstorage.NewError(objectstorage.KindInternal)
}

type operation string

const (
	operationMetadata      operation = "metadata"
	operationDelete        operation = "delete"
	operationPut           operation = "put"
	operationPutCreateOnly operation = "put_create_only"
	operationComplete      operation = "complete"
	operationPresign       operation = "presign"
)

// operationError deliberately exposes only the portable outcome, never the SDK error.
//
//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func operationError(op operation, err error, wroteHeaders bool) error {
	status, code := providerError(err)
	if status == http.StatusForbidden {
		return objectstorage.NewError(objectstorage.KindDenied)
	}
	if op == operationMetadata && status == http.StatusNotFound && admittedAbsenceCode(code) {
		return objectstorage.NewError(objectstorage.KindNotFound)
	}
	if op == operationPutCreateOnly && status == http.StatusPreconditionFailed {
		return objectstorage.NewError(objectstorage.KindPreconditionFailed)
	}
	if wroteHeaders && mutatingOperation(op) {
		return objectstorage.NewError(objectstorage.KindOutcomeUnknown)
	}
	if errors.Is(err, context.Canceled) {
		return objectstorage.NewError(objectstorage.KindCancelled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return objectstorage.NewError(objectstorage.KindDeadlineExceeded)
	}
	if op == operationMetadata && (status == http.StatusTooManyRequests || status >= http.StatusInternalServerError) {
		return objectstorage.NewError(objectstorage.KindTemporary)
	}
	return objectstorage.NewError(objectstorage.KindInternal)
}

func providerError(err error) (int, string) {
	var response interface{ HTTPStatusCode() int }
	if !errors.As(err, &response) {
		return 0, ""
	}
	if apiError, ok := errors.AsType[smithy.APIError](err); ok {
		return response.HTTPStatusCode(), apiError.ErrorCode()
	}
	return response.HTTPStatusCode(), ""
}

func admittedAbsenceCode(code string) bool {
	return code == "NoSuchKey"
}

func mutatingOperation(op operation) bool {
	return op == operationPut || op == operationPutCreateOnly || op == operationComplete || op == operationDelete
}
