package s3

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go"
	"github.com/example/go-service-template-rest/internal/infra/httpclient"
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
	operationMetadata        operation = "metadata"
	operationDownload        operation = "download"
	operationDelete          operation = "delete"
	operationCreateMultipart operation = "create_multipart"
	operationMultipartStage  operation = "multipart_stage"
	operationPut             operation = "put"
	operationPutCreateOnly   operation = "put_create_only"
	operationComplete        operation = "complete"
	operationPresign         operation = "presign"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:/+=-]{1,128}$`)

var diagnosticCodes = map[string]struct{}{
	"AccessDenied": {}, "AuthorizationHeaderMalformed": {}, "ConditionalRequestConflict": {},
	"ExpiredToken": {}, "InternalError": {}, "InvalidAccessKeyId": {}, "InvalidArgument": {},
	"InvalidRequest": {}, "NoSuchBucket": {}, "NoSuchKey": {}, "NotFound": {}, "PreconditionFailed": {},
	"RequestTimeTooSkewed": {}, "RequestTimeout": {}, "RequestTimeoutException": {},
	"ServiceUnavailable": {}, "SignatureDoesNotMatch": {}, "SlowDown": {}, "TooManyRequests": {},
}

type providerFailureError struct {
	result    error
	status    int
	code      string
	requestID string
	category  string
	phase     string
	attempts  int
}

func (err *providerFailureError) Error() string { return err.result.Error() }
func (err *providerFailureError) Unwrap() error { return err.result }

// operationError deliberately exposes only the portable outcome, never the SDK error.
func operationError(provider Provider, op operation, err error, send *sendState) error {
	status, code, requestID := providerError(err)
	wroteHeaders := send != nil && send.wroteHeaders
	result := mappedOperationError(provider, op, err, wroteHeaders, status, code)
	attempts := 0
	if send != nil {
		attempts = send.attempts
	}
	return &providerFailureError{
		result:    result,
		status:    status,
		code:      diagnosticCode(code),
		requestID: diagnosticRequestID(requestID),
		category:  failureCategory(err, status, code),
		phase:     providerFailurePhase(op, err, status, wroteHeaders),
		attempts:  attempts,
	}
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func mappedOperationError(provider Provider, op operation, err error, wroteHeaders bool, status int, code string) error {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return objectstorage.NewError(objectstorage.KindDenied)
	}
	if status == http.StatusNotFound && admittedAbsence(op, code) {
		return objectstorage.NewError(objectstorage.KindNotFound)
	}
	if op == operationPutCreateOnly {
		if kind, ok := createOnlyErrorKind(provider, status, code); ok {
			return objectstorage.NewError(kind)
		}
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
	if readOperation(op) && (status == http.StatusTooManyRequests || status >= http.StatusInternalServerError || readRetryable(err) == aws.TrueTernary) {
		return objectstorage.NewError(objectstorage.KindTemporary)
	}
	return objectstorage.NewError(objectstorage.KindInternal)
}

func admittedAbsence(op operation, code string) bool {
	return op == operationMetadata && (code == "NotFound" || code == "NoSuchKey") ||
		op == operationDownload && code == "NoSuchKey"
}

func createOnlyErrorKind(provider Provider, status int, code string) (objectstorage.ErrorKind, bool) {
	if status == http.StatusPreconditionFailed && code == "PreconditionFailed" &&
		(provider == ProviderAmazonS3 || provider == ProviderCloudflare) {
		return objectstorage.KindPreconditionFailed, true
	}
	if provider == ProviderAmazonS3 && status == http.StatusConflict && code == "ConditionalRequestConflict" {
		return objectstorage.KindTemporary, true
	}
	return "", false
}

func providerError(err error) (int, string, string) {
	var response interface{ HTTPStatusCode() int }
	if !errors.As(err, &response) {
		return 0, "", ""
	}
	requestID := ""
	if request, ok := errors.AsType[interface {
		error
		ServiceRequestID() string
	}](err); ok {
		requestID = request.ServiceRequestID()
	}
	if apiError, ok := errors.AsType[smithy.APIError](err); ok {
		return response.HTTPStatusCode(), apiError.ErrorCode(), requestID
	}
	return response.HTTPStatusCode(), "", requestID
}

func readOperation(op operation) bool {
	return op == operationMetadata || op == operationDownload
}

func mutatingOperation(op operation) bool {
	return op == operationPut || op == operationPutCreateOnly || op == operationCreateMultipart || op == operationComplete || op == operationDelete
}

func providerFailurePhase(op operation, err error, status int, wroteHeaders bool) string {
	if status != 0 {
		return "response"
	}
	if errors.Is(err, httpclient.ErrTargetDenied) {
		return "authority"
	}
	if trustFailure(err) {
		return "tls"
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return "dns"
	}
	if wroteHeaders {
		return "after_send"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "context"
	}
	if op == operationPresign {
		return "sign"
	}
	return "transport"
}

func readRetryable(err error) aws.Ternary {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, httpclient.ErrTargetDenied) || trustFailure(err) {
		return aws.FalseTernary
	}
	status, code, _ := providerError(err)
	if status == http.StatusTooManyRequests || status == http.StatusInternalServerError || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout {
		return aws.TrueTernary
	}
	if code == "RequestTimeout" || code == "RequestTimeoutException" || code == "SlowDown" || code == "TooManyRequests" {
		return aws.TrueTernary
	}
	if status != 0 {
		return aws.FalseTernary
	}
	if _, ok := errors.AsType[*net.DNSError](err); ok {
		return aws.FalseTernary
	}
	if _, ok := errors.AsType[net.Error](err); ok {
		return aws.TrueTernary
	}
	return aws.FalseTernary
}

func trustFailure(err error) bool {
	if _, ok := errors.AsType[*tls.CertificateVerificationError](err); ok {
		return true
	}
	if _, ok := errors.AsType[x509.UnknownAuthorityError](err); ok {
		return true
	}
	if _, ok := errors.AsType[x509.HostnameError](err); ok {
		return true
	}
	_, ok := errors.AsType[x509.CertificateInvalidError](err)
	return ok
}

func diagnosticCode(code string) string {
	if code == "" {
		return ""
	}
	if _, ok := diagnosticCodes[code]; ok {
		return code
	}
	return "other"
}

func diagnosticRequestID(requestID string) string {
	if requestID == "" {
		return ""
	}
	if requestIDPattern.MatchString(requestID) {
		return requestID
	}
	return "invalid"
}

func failureCategory(err error, status int, code string) string {
	if status == http.StatusUnauthorized || code == "InvalidAccessKeyId" || code == "SignatureDoesNotMatch" || code == "ExpiredToken" {
		return "credential"
	}
	if errors.Is(err, httpclient.ErrTargetDenied) || trustFailure(err) {
		return "authority_tls"
	}
	if status == http.StatusTooManyRequests || code == "SlowDown" || code == "TooManyRequests" {
		return "throttle"
	}
	if status == 0 {
		if _, ok := errors.AsType[net.Error](err); ok {
			return "transport"
		}
		return ""
	}
	return "provider"
}
