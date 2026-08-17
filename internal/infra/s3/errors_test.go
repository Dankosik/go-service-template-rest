package s3

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/smithy-go"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestStableErrorSanitizesProviderFailures(t *testing.T) {
	t.Parallel()
	if got := objectstorage.Kind(stableError(errors.New("provider secret endpoint"))); got != objectstorage.KindInternal {
		t.Fatalf("Kind(stableError(provider error)) = %q, want %q", got, objectstorage.KindInternal)
	}
	if got := objectstorage.Kind(stableError(&bodyTooLargeError{})); got != objectstorage.KindTooLarge {
		t.Fatalf("Kind(stableError(body limit)) = %q, want %q", got, objectstorage.KindTooLarge)
	}
}

func TestErrorMappingAndReadRetryAreConservative(t *testing.T) {
	t.Parallel()
	t.Run("metadata retries only admitted transient responses", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return s3Response(http.StatusServiceUnavailable, nil, `<Error><Code>SlowDown</Code></Error>`), nil
			}
			return s3Response(http.StatusOK, http.Header{
				"Content-Length": {"1"},
				"Last-Modified":  {time.Now().UTC().Format(http.TimeFormat)},
			}, ""), nil
		})
		if _, err := client.Metadata(t.Context(), "object"); err != nil || attempts != 3 {
			t.Fatalf("Metadata() error = %v, attempts = %d, want nil and 3", err, attempts)
		}
	})

	for _, test := range []struct {
		name string
		code int
		body string
		want objectstorage.ErrorKind
	}{
		{name: "denial is terminal", code: http.StatusForbidden, body: `<Error><Code>AccessDenied</Code></Error>`, want: objectstorage.KindDenied},
		{name: "absence is terminal", code: http.StatusNotFound, body: `<Error><Code>NotFound</Code></Error>`, want: objectstorage.KindNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			attempts := 0
			client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
				attempts++
				return s3Response(test.code, nil, test.body), nil
			})
			_, err := client.Metadata(t.Context(), "object")
			if objectstorage.Kind(err) != test.want || attempts != 1 {
				t.Fatalf("Metadata() kind = %q, attempts = %d, want %q and 1", objectstorage.Kind(err), attempts, test.want)
			}
		})
	}

	t.Run("exhaustion is temporary", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			attempts++
			return s3Response(http.StatusInternalServerError, nil, `<Error><Code>InternalError</Code></Error>`), nil
		})
		_, err := client.Metadata(t.Context(), "object")
		if objectstorage.Kind(err) != objectstorage.KindTemporary || attempts != 3 {
			t.Fatalf("Metadata() kind = %q, attempts = %d, want temporary and 3", objectstorage.Kind(err), attempts)
		}
	})

	t.Run("mutation stays one attempt", func(t *testing.T) {
		t.Parallel()
		attempts := 0
		client := scriptedClient(t, func(*http.Request) (*http.Response, error) {
			attempts++
			return s3Response(http.StatusServiceUnavailable, nil, `<Error><Code>SlowDown</Code></Error>`), nil
		})
		if err := client.Delete(t.Context(), "object"); err == nil || attempts != 1 {
			t.Fatalf("Delete() error = %v, attempts = %d, want failure and 1", err, attempts)
		}
	})
}

func TestErrorMappingIsConservativeAndOneAttempt(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		provider Provider
		op       operation
		err      error
		sent     bool
		want     objectstorage.ErrorKind
	}{
		{name: "403 is denied", op: operationMetadata, err: providerTestError{status: http.StatusForbidden, code: "AccessDenied"}, want: objectstorage.KindDenied},
		{name: "401 is denied", op: operationDownload, err: providerTestError{status: http.StatusUnauthorized, code: "InvalidAccessKeyId"}, want: objectstorage.KindDenied},
		{name: "generic metadata 404 is absence", op: operationMetadata, err: providerTestError{status: http.StatusNotFound, code: "NotFound"}, want: objectstorage.KindNotFound},
		{name: "download NoSuchKey is absence", op: operationDownload, err: providerTestError{status: http.StatusNotFound, code: "NoSuchKey"}, want: objectstorage.KindNotFound},
		{name: "generic download 404 is not object absence", op: operationDownload, err: providerTestError{status: http.StatusNotFound, code: "private-code"}, want: objectstorage.KindInternal},
		{name: "missing bucket is not object absence", op: operationDownload, err: providerTestError{status: http.StatusNotFound, code: "NoSuchBucket"}, want: objectstorage.KindInternal},
		{name: "Amazon conditional create only", provider: ProviderAmazonS3, op: operationPutCreateOnly, err: providerTestError{status: http.StatusPreconditionFailed, code: "PreconditionFailed"}, want: objectstorage.KindPreconditionFailed},
		{name: "Amazon concurrent conditional create only is retryable", provider: ProviderAmazonS3, op: operationPutCreateOnly, err: providerTestError{status: http.StatusConflict, code: "ConditionalRequestConflict"}, want: objectstorage.KindTemporary},
		{name: "Amazon unrelated precondition is unknown", provider: ProviderAmazonS3, op: operationPutCreateOnly, err: providerTestError{status: http.StatusPreconditionFailed, code: "private-code"}, sent: true, want: objectstorage.KindOutcomeUnknown},
		{name: "Amazon unrelated conflict is unknown", provider: ProviderAmazonS3, op: operationPutCreateOnly, err: providerTestError{status: http.StatusConflict, code: "OperationAborted"}, sent: true, want: objectstorage.KindOutcomeUnknown},
		{name: "R2 conditional conflict is not Amazon evidence", provider: ProviderCloudflare, op: operationPutCreateOnly, err: providerTestError{status: http.StatusConflict, code: "ConditionalRequestConflict"}, sent: true, want: objectstorage.KindOutcomeUnknown},
		{name: "R2 documented precondition", provider: ProviderCloudflare, op: operationPutCreateOnly, err: providerTestError{status: http.StatusPreconditionFailed, code: "PreconditionFailed"}, sent: true, want: objectstorage.KindPreconditionFailed},
		{name: "metadata throttling is temporary", op: operationMetadata, err: providerTestError{status: http.StatusTooManyRequests, code: "SlowDown"}, want: objectstorage.KindTemporary},
		{name: "delete throttle is not retry permission", op: operationDelete, err: providerTestError{status: http.StatusTooManyRequests, code: "SlowDown"}, want: objectstorage.KindInternal},
		{name: "malformed control body stays internal", op: operationMetadata, err: &bodyTooLargeError{}, want: objectstorage.KindInternal},
		{name: "after write loss is unknown", op: operationDelete, err: context.DeadlineExceeded, sent: true, want: objectstorage.KindOutcomeUnknown},
		{name: "lost multipart create is unknown", op: operationCreateMultipart, err: context.DeadlineExceeded, sent: true, want: objectstorage.KindOutcomeUnknown},
		{name: "before write context is deadline", op: operationDelete, err: context.DeadlineExceeded, want: objectstorage.KindDeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := objectstorage.Kind(operationError(test.provider, test.op, test.err, &sendState{wroteHeaders: test.sent, attempts: 1})); got != test.want {
				t.Fatalf("Kind(operationError()) = %q, want %q", got, test.want)
			}
		})
	}

	t.Run("create-only 412 reaches the shared mapper", func(t *testing.T) {
		t.Parallel()
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			_, _ = decodeAWSChunked(t, request.Body)
			return s3Response(http.StatusPreconditionFailed, nil, `<Error><Code>PreconditionFailed</Code></Error>`), nil
		})
		_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBufferString("data")), objectstorage.UploadOptions{
			ContentLength: 4, Intent: objectstorage.UploadCreateOnly,
		})
		if got := objectstorage.Kind(err); got != objectstorage.KindPreconditionFailed {
			t.Fatalf("Kind(Upload(create-only)) = %q, want %q", got, objectstorage.KindPreconditionFailed)
		}
	})

	t.Run("create-only 409 reaches the shared mapper", func(t *testing.T) {
		t.Parallel()
		client := scriptedClient(t, func(request *http.Request) (*http.Response, error) {
			_, _ = decodeAWSChunked(t, request.Body)
			return s3Response(http.StatusConflict, nil, `<Error><Code>ConditionalRequestConflict</Code></Error>`), nil
		})
		_, err := client.Upload(context.Background(), "object", uploadSource(bytes.NewBufferString("data")), objectstorage.UploadOptions{
			ContentLength: 4, Intent: objectstorage.UploadCreateOnly,
		})
		if got := objectstorage.Kind(err); got != objectstorage.KindTemporary {
			t.Fatalf("Kind(Upload(create-only)) = %q, want %q", got, objectstorage.KindTemporary)
		}
	})
}

func TestProviderDiagnosticClassificationIsFinite(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		status int
		code   string
		want   string
	}{
		{name: "401", status: http.StatusUnauthorized, code: "AccessDenied", want: "credential"},
		{name: "invalid access key", status: http.StatusForbidden, code: "InvalidAccessKeyId", want: "credential"},
		{name: "bad signature", status: http.StatusForbidden, code: "SignatureDoesNotMatch", want: "credential"},
		{name: "expired token", status: http.StatusForbidden, code: "ExpiredToken", want: "credential"},
		{name: "malformed authorization", status: http.StatusBadRequest, code: "AuthorizationHeaderMalformed", want: "provider"},
		{name: "clock skew", status: http.StatusForbidden, code: "RequestTimeTooSkewed", want: "provider"},
		{name: "throttle", status: http.StatusTooManyRequests, code: "SlowDown", want: "throttle"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := failureCategory(providerTestError{status: test.status, code: test.code}, test.status, test.code); got != test.want {
				t.Fatalf("failureCategory() = %q, want %q", got, test.want)
			}
		})
	}
	for _, test := range []struct {
		name   string
		op     operation
		err    error
		status int
		sent   bool
		want   string
	}{
		{name: "response", op: operationMetadata, err: providerTestError{status: http.StatusServiceUnavailable}, status: http.StatusServiceUnavailable, want: "response"},
		{name: "after send", op: operationDelete, err: errors.New("lost response"), sent: true, want: "after_send"},
		{name: "context", op: operationMetadata, err: context.DeadlineExceeded, want: "context"},
		{name: "sign", op: operationPresign, err: errors.New("signing failed"), want: "sign"},
	} {
		t.Run("phase "+test.name, func(t *testing.T) {
			t.Parallel()
			if got := providerFailurePhase(test.op, test.err, test.status, test.sent); got != test.want {
				t.Fatalf("providerFailurePhase() = %q, want %q", got, test.want)
			}
		})
	}
}

type providerTestError struct {
	status    int
	code      string
	requestID string
}

func (err providerTestError) Error() string            { return "private provider error" }
func (err providerTestError) HTTPStatusCode() int      { return err.status }
func (err providerTestError) ServiceRequestID() string { return err.requestID }
func (err providerTestError) Unwrap() error {
	return &smithy.GenericAPIError{Code: err.code, Message: "secret message"}
}
