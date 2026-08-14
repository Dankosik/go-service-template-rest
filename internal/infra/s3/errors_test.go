package s3

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestStableErrorSanitizesProviderFailures(t *testing.T) {
	if got := objectstorage.Kind(stableError(errors.New("provider secret endpoint"))); got != objectstorage.KindInternal {
		t.Fatalf("Kind(stableError(provider error)) = %q, want %q", got, objectstorage.KindInternal)
	}
	if got := objectstorage.Kind(stableError(&bodyTooLargeError{})); got != objectstorage.KindTooLarge {
		t.Fatalf("Kind(stableError(body limit)) = %q, want %q", got, objectstorage.KindTooLarge)
	}
}

func TestErrorMappingIsConservativeAndOneAttempt(t *testing.T) {
	for _, test := range []struct {
		name string
		op   operation
		err  error
		sent bool
		want objectstorage.ErrorKind
	}{
		{name: "403 is denied", op: operationMetadata, err: providerTestError{status: http.StatusForbidden, code: "AccessDenied"}, want: objectstorage.KindDenied},
		{name: "only admitted 404 is absence", op: operationMetadata, err: providerTestError{status: http.StatusNotFound, code: "NoSuchKey"}, want: objectstorage.KindNotFound},
		{name: "generic 404 is not absence", op: operationMetadata, err: providerTestError{status: http.StatusNotFound, code: "NotFound"}, want: objectstorage.KindInternal},
		{name: "arbitrary 404 is internal", op: operationMetadata, err: providerTestError{status: http.StatusNotFound, code: "private-code"}, want: objectstorage.KindInternal},
		{name: "conditional create only", op: operationPutCreateOnly, err: providerTestError{status: http.StatusPreconditionFailed, code: "PreconditionFailed"}, want: objectstorage.KindPreconditionFailed},
		{name: "metadata throttling is temporary", op: operationMetadata, err: providerTestError{status: http.StatusTooManyRequests, code: "SlowDown"}, want: objectstorage.KindTemporary},
		{name: "delete throttle is not retry permission", op: operationDelete, err: providerTestError{status: http.StatusTooManyRequests, code: "SlowDown"}, want: objectstorage.KindInternal},
		{name: "malformed control body stays internal", op: operationMetadata, err: &bodyTooLargeError{}, want: objectstorage.KindInternal},
		{name: "after write loss is unknown", op: operationDelete, err: context.DeadlineExceeded, sent: true, want: objectstorage.KindOutcomeUnknown},
		{name: "before write context is deadline", op: operationDelete, err: context.DeadlineExceeded, want: objectstorage.KindDeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := objectstorage.Kind(operationError(test.op, test.err, test.sent)); got != test.want {
				t.Fatalf("Kind(operationError()) = %q, want %q", got, test.want)
			}
		})
	}

	t.Run("create-only 412 reaches the shared mapper", func(t *testing.T) {
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			_, _ = decodeAWSChunked(t, request.Body)
			return s3Response(http.StatusPreconditionFailed, nil, `<Error><Code>PreconditionFailed</Code></Error>`), nil
		})
		_, err := client.Upload(context.Background(), "object", bytes.NewBufferString("data"), objectstorage.UploadOptions{
			ContentLength: 4, Intent: objectstorage.UploadCreateOnly,
		})
		if got := objectstorage.Kind(err); got != objectstorage.KindPreconditionFailed {
			t.Fatalf("Kind(Upload(create-only)) = %q, want %q", got, objectstorage.KindPreconditionFailed)
		}
	})
}

type providerTestError struct {
	status int
	code   string
}

func (err providerTestError) Error() string       { return "private provider error" }
func (err providerTestError) HTTPStatusCode() int { return err.status }
func (err providerTestError) Unwrap() error {
	return &smithy.GenericAPIError{Code: err.code, Message: "secret message"}
}
