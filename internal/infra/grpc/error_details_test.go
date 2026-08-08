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
	"net/http"
	"net/http/httptest"
	"strconv"
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

const testErrorDomain = "orders.example.com"

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
		delay           time.Duration
		wantRetryAfter  int
		wantGRPCSeconds float64
	}{
		{delay: 200 * time.Millisecond, wantRetryAfter: 1, wantGRPCSeconds: 0.2},
		{delay: time.Second, wantRetryAfter: 1, wantGRPCSeconds: 1},
		{delay: 1500 * time.Millisecond, wantRetryAfter: 2, wantGRPCSeconds: 1.5},
	} {
		t.Run(testCase.delay.String(), func(t *testing.T) {
			mappers := []failure.Mapper{saturatedMapper(testCase.delay)}

			retryInfo, _ := classifiedDetailsFromServer(t, mappers, testErrorDomain)
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
			if float64(advertised) < testCase.wantGRPCSeconds {
				t.Fatalf(
					"HTTP advertised %ds, shorter than the mapper's %s",
					advertised,
					testCase.delay,
				)
			}
		})
	}
}

// The gRPC code space is coarser than failure.Code: InvalidArgument answers for
// both CodeBadRequest and CodeUnprocessableContent. The reason is what lets a
// caller tell them apart.
func TestErrorInfoDistinguishesCodesSharingOneGRPCCode(t *testing.T) {
	reasons := make(map[failure.Code]string, 2)
	for _, code := range []failure.Code{failure.CodeBadRequest, failure.CodeUnprocessableContent} {
		mappers := []failure.Mapper{func(err error) (failure.Classification, bool) {
			if !errors.Is(err, errSaturated) {
				return failure.Classification{}, false
			}
			return failure.Classification{Code: code, Detail: "rejected"}, true
		}}

		_, errorInfo := classifiedDetailsFromServer(t, mappers, testErrorDomain)
		if errorInfo == nil {
			t.Fatalf("%s carried no ErrorInfo", code)
		}
		if errorInfo.GetDomain() != testErrorDomain {
			t.Fatalf("ErrorInfo domain = %q, want %q", errorInfo.GetDomain(), testErrorDomain)
		}
		reasons[code] = errorInfo.GetReason()
	}

	if reasons[failure.CodeBadRequest] == reasons[failure.CodeUnprocessableContent] {
		t.Fatalf(
			"both codes rendered reason %q, so a caller cannot tell them apart",
			reasons[failure.CodeBadRequest],
		)
	}
}

// google.rpc.ErrorInfo documents Reason as at most 63 characters matching
// [A-Z][A-Z0-9_]+[A-Z0-9]. A code added later that does not render to one is a
// defect in that code, caught here over the whole catalog rather than per call
// site.
func TestEveryFailureCodeRendersAConformingReason(t *testing.T) {
	for code := range failureCodeConstantNames(t) {
		reason := reasonFor(t, code)
		if len(reason) < 3 || len(reason) > 63 {
			t.Errorf("%s renders reason %q, whose length is outside [3,63]", code, reason)
			continue
		}
		for index, char := range reason {
			isUpper := char >= 'A' && char <= 'Z'
			isDigit := char >= '0' && char <= '9'
			edge := index == 0 || index == len(reason)-1
			if isUpper || (isDigit && !edge) || (char == '_' && !edge) {
				continue
			}
			t.Errorf("%s renders reason %q, which is not UPPER_SNAKE_CASE", code, reason)
			break
		}
	}
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
	_, connection := startTestServerWithOptions(t, cfg, register, Options{
		ErrorDomain: testErrorDomain,
	})

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

func TestErrorInfoIsOmittedWithoutADomainAndRetryInfoIsNot(t *testing.T) {
	mappers := []failure.Mapper{saturatedMapper(time.Second)}

	retryInfo, errorInfo := classifiedDetailsFromServer(t, mappers, "")
	if errorInfo != nil {
		t.Fatalf("ErrorInfo published without a domain: %v", errorInfo)
	}
	if retryInfo == nil {
		t.Fatal("an absent domain also dropped the retry hint")
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

// classifiedDetailsFromServer drives one classified error through a real server
// and returns whichever of the two details the caller received.
func classifiedDetailsFromServer(
	t *testing.T,
	mappers []failure.Mapper,
	domain string,
) (*errdetails.RetryInfo, *errdetails.ErrorInfo) {
	t.Helper()

	register := func(registrar grpc.ServiceRegistrar) {
		registerUnaryTestService(registrar, testUnaryFullMethod,
			func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return nil, errSaturated
			})
	}
	_, connection := startTestServerWithOptions(t, testServerConfig(), register, Options{
		DomainErrors: mappers,
		ErrorDomain:  domain,
	})

	err := connection.Invoke(t.Context(), testUnaryFullMethod, &emptypb.Empty{}, &emptypb.Empty{})
	if err == nil {
		t.Fatal("classified handler error reached the caller as success")
	}

	var (
		retryInfo *errdetails.RetryInfo
		errorInfo *errdetails.ErrorInfo
	)
	for _, detail := range status.Convert(err).Details() {
		switch typed := detail.(type) {
		case *errdetails.RetryInfo:
			retryInfo = typed
		case *errdetails.ErrorInfo:
			errorInfo = typed
		}
	}
	return retryInfo, errorInfo
}

func reasonFor(t *testing.T, code failure.Code) string {
	t.Helper()

	for _, detail := range status.Convert(
		mappedStatus(failure.Classification{Code: code}, testErrorDomain),
	).Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			return info.GetReason()
		}
	}
	t.Fatalf("%s rendered no ErrorInfo", code)
	return ""
}

// httpRetryAfter renders the same mapper through the HTTP transport's own
// response path and returns the header a caller receives.
func httpRetryAfter(t *testing.T, mappers []failure.Mapper) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe", nil)
	httpx.RejectResponse(mappers...)(recorder, request, errSaturated)

	return recorder.Header().Get("Retry-After")
}
