// The client-visible detail contract: what a classified domain error carries
// beyond its code and message, and what an unclassified one must not.
//
// The cross-transport case drives the HTTP renderer too, through the one
// exported function that reaches it. That import edge exists because
// retryAfterSeconds is unexported, so a gRPC-side restatement of its rounding
// would prove a copy of the rule rather than the rule.

package grpcx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const testErrorDomain = "grpcx.test.Service"

var errSaturated = errors.New("dependency is saturated")

// saturatedMapper classifies errSaturated with a retry hint, which is the one
// mapper value the two transports render differently.
func saturatedMapper(delay time.Duration) failure.Mapper {
	return func(err error) (failure.Classification, bool) {
		if !errors.Is(err, errSaturated) {
			return failure.Classification{}, false
		}
		return failure.Classification{
			Code:       failure.CodeServiceUnavailable,
			Detail:     "dependency is saturated",
			RetryAfter: delay,
		}, true
	}
}

// Neither transport advertises a delay shorter than the mapper's. gRPC carries
// it exactly; HTTP rounds up, because whole seconds is Retry-After's own
// granularity rather than a defect to mirror.
func TestRetryHintReachesBothTransports(t *testing.T) {
	for _, testCase := range []struct {
		delay          time.Duration
		wantRetryAfter int
	}{
		{delay: 200 * time.Millisecond, wantRetryAfter: 1},
		{delay: time.Second, wantRetryAfter: 1},
		{delay: 1500 * time.Millisecond, wantRetryAfter: 2},
	} {
		t.Run(testCase.delay.String(), func(t *testing.T) {
			mappers := []failure.Mapper{saturatedMapper(testCase.delay)}

			retryInfo := grpcRetryInfoFromServer(t, mappers)
			if retryInfo == nil {
				t.Fatal("gRPC status carried no RetryInfo for a positive mapper delay")
			}
			if got := retryInfo.GetRetryDelay().AsDuration(); got != testCase.delay {
				t.Fatalf("RetryInfo delay = %s, want the mapper's %s", got, testCase.delay)
			}

			header := httpRetryAfter(t, mappers)
			if header != strconv.Itoa(testCase.wantRetryAfter) {
				t.Fatalf("HTTP Retry-After = %q, want %d", header, testCase.wantRetryAfter)
			}
			advertised, err := strconv.Atoi(header)
			if err != nil {
				t.Fatalf("HTTP Retry-After = %q, want whole seconds", header)
			}
			if float64(advertised) < testCase.delay.Seconds() {
				t.Fatalf(
					"HTTP advertised %ds, shorter than the mapper's %s",
					advertised,
					testCase.delay,
				)
			}
		})
	}
}

func TestEveryFailureCodeRendersAConformingReason(t *testing.T) {
	t.Parallel()

	expected := map[failure.Code]codes.Code{
		failure.CodeBadRequest:            codes.InvalidArgument,
		failure.CodeUnauthorized:          codes.Unauthenticated,
		failure.CodeForbidden:             codes.PermissionDenied,
		failure.CodeNotFound:              codes.NotFound,
		failure.CodeMethodNotAllowed:      codes.Unimplemented,
		failure.CodeAlreadyExists:         codes.AlreadyExists,
		failure.CodeRequestEntityTooLarge: codes.ResourceExhausted,
		// profile:authn-bearer:start
		failure.CodeRequestHeaderFieldsTooLarge: codes.ResourceExhausted,
		// profile:authn-bearer:end
		failure.CodeUnprocessableContent: codes.InvalidArgument,
		failure.CodeTooManyRequests:      codes.ResourceExhausted,
		// profile:http-idempotency-postgres:start
		failure.CodeIdempotencyKeyMismatch:    codes.InvalidArgument,
		failure.CodeIdempotencyUnavailable:    codes.Unavailable,
		failure.CodeIdempotencyOutcomeUnknown: codes.Unavailable,
		// profile:http-idempotency-postgres:end
		failure.CodeInternalError:      codes.Internal,
		failure.CodeServiceUnavailable: codes.Unavailable,
		failure.CodeGatewayTimeout:     codes.DeadlineExceeded,
	}
	allCodes := failure.AllCodes()
	if len(expected) != len(allCodes) {
		t.Fatalf("mapping table has %d codes, failure catalog has %d", len(expected), len(allCodes))
	}
	reasonPattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,61}[A-Z0-9]$`)

	for _, failureCode := range allCodes {
		t.Run(string(failureCode), func(t *testing.T) {
			t.Parallel()

			wantCode, ok := expected[failureCode]
			if !ok {
				t.Fatalf("failure code %q has no expected gRPC status", failureCode)
			}
			converted := status.Convert(mappedStatus(failure.Classification{
				Code:   failureCode,
				Detail: "classified failure",
			}, testErrorDomain))
			if converted.Code() != wantCode {
				t.Fatalf("status code = %s, want %s", converted.Code(), wantCode)
			}

			var errorInfo *errdetails.ErrorInfo
			for _, detail := range converted.Details() {
				if typed, ok := detail.(*errdetails.ErrorInfo); ok {
					errorInfo = typed
				}
			}
			if errorInfo == nil {
				t.Fatal("status carried no ErrorInfo")
			}
			wantReason := strings.ToUpper(string(failureCode))
			if errorInfo.GetReason() != wantReason || !reasonPattern.MatchString(errorInfo.GetReason()) {
				t.Fatalf("ErrorInfo reason = %q, want conforming %q", errorInfo.GetReason(), wantReason)
			}
			if errorInfo.GetDomain() != testErrorDomain {
				t.Fatalf("ErrorInfo domain = %q, want %q", errorInfo.GetDomain(), testErrorDomain)
			}
		})
	}
}

func TestHandlerErrorBoundarySnapshotsDomainMappers(t *testing.T) {
	t.Parallel()

	specific := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "specific"}, true
	}
	broad := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeBadRequest, Detail: "broad"}, true
	}
	mappers := []failure.Mapper{specific, broad}
	boundary := handlerErrorBoundary(slog.New(slog.DiscardHandler), mappers)
	mappers[0] = broad

	err := boundary(t.Context(), "/"+testErrorDomain+"/Unary", func(context.Context) error {
		return errSaturated
	})
	assertStatusCode(t, err, codes.AlreadyExists)
}

func TestUnclassifiedErrorCarriesNoDetailsAndNoHandlerText(t *testing.T) {
	const handlerText = "dial tcp 10.0.0.7:5432: connection refused"

	cfg := testServerConfig()
	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, errors.New(handlerText)
			})
	}
	_, connection := startTestServerWithOptions(t, cfg, register, Options{})

	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	assertStatusCode(t, err, codes.Internal)

	converted := status.Convert(err)
	if details := converted.Details(); len(details) != 0 {
		t.Fatalf("unclassified error carried %d details, want none", len(details))
	}
	if converted.Message() == handlerText {
		t.Fatalf("status message disclosed the handler's own text: %q", converted.Message())
	}
}

func TestClassifiedEmptyDetailUsesGRPCOwnedFallback(t *testing.T) {
	t.Parallel()

	err := mappedStatus(failure.Classification{Code: failure.CodeAlreadyExists}, "")
	if code := status.Code(err); code != codes.AlreadyExists {
		t.Fatalf("code = %s, want %s", code, codes.AlreadyExists)
	}
	if detail := status.Convert(err).Message(); detail != "request failed" {
		t.Fatalf("detail = %q, want gRPC-owned fallback", detail)
	}
}

// grpcRetryInfoFromServer drives one classified error through a real server.
func grpcRetryInfoFromServer(
	t *testing.T,
	mappers []failure.Mapper,
) *errdetails.RetryInfo {
	t.Helper()

	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, errSaturated
			})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		DomainErrors: mappers,
	})

	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	if err == nil {
		t.Fatal("classified handler error reached the caller as success")
	}

	for _, detail := range status.Convert(err).Details() {
		if retryInfo, ok := detail.(*errdetails.RetryInfo); ok {
			return retryInfo
		}
	}
	return nil
}

// httpRetryAfter renders the same mapper through the HTTP transport's own
// response path and returns the header a caller receives.
func httpRetryAfter(t *testing.T, mappers []failure.Mapper) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe", nil)
	httpx.RejectResponse(slog.New(slog.DiscardHandler), mappers...)(recorder, request, errSaturated)

	return recorder.Header().Get("Retry-After")
}
