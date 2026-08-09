package failure_test

import (
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
)

func TestCodesAreStableAndTransportNeutral(t *testing.T) {
	t.Parallel()

	want := []failure.Code{
		"bad_request",
		"unauthorized",
		"forbidden",
		"not_found",
		"method_not_allowed",
		"already_exists",
		"request_entity_too_large",
		// profile:authn-oidc-jwt:start
		"request_header_fields_too_large",
		// profile:authn-oidc-jwt:end
		"unprocessable_content",
		"too_many_requests",
		"internal_error",
		"service_unavailable",
		"gateway_timeout",
	}
	got := []failure.Code{
		failure.CodeBadRequest,
		failure.CodeUnauthorized,
		failure.CodeForbidden,
		failure.CodeNotFound,
		failure.CodeMethodNotAllowed,
		failure.CodeAlreadyExists,
		failure.CodeRequestEntityTooLarge,
		// profile:authn-oidc-jwt:start
		failure.CodeRequestHeaderFieldsTooLarge,
		// profile:authn-oidc-jwt:end
		failure.CodeUnprocessableContent,
		failure.CodeTooManyRequests,
		failure.CodeInternalError,
		failure.CodeServiceUnavailable,
		failure.CodeGatewayTimeout,
	}
	if len(got) != len(want) {
		t.Fatalf("published codes = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("published codes = %v, want %v", got, want)
		}
		if got[index] == "conflict" {
			t.Fatal("generic conflict became a transport-neutral domain identity")
		}
	}
}

func TestClassifySkipsNilAndUsesFirstMatch(t *testing.T) {
	t.Parallel()

	target := errors.New("target")
	broad := func(error) (failure.Classification, bool) {
		return failure.Classification{Code: failure.CodeBadRequest}, true
	}
	nonMatching := func(error) (failure.Classification, bool) {
		return failure.Classification{}, false
	}
	specific := func(err error) (failure.Classification, bool) {
		if !errors.Is(err, target) {
			return failure.Classification{}, false
		}
		return failure.Classification{Code: failure.CodeAlreadyExists, Detail: "already exists"}, true
	}

	got, ok := failure.Classify(target, []failure.Mapper{nil, nonMatching, specific, broad})
	if !ok {
		t.Fatal("Classify() reported no match")
	}
	want := failure.Classification{Code: failure.CodeAlreadyExists, Detail: "already exists"}
	if got != want {
		t.Fatalf("Classify() = %+v, want %+v", got, want)
	}
}

func TestClassifyRefusesUnknownError(t *testing.T) {
	t.Parallel()

	got, ok := failure.Classify(errors.New("unknown"), []failure.Mapper{nil})
	if ok || got != (failure.Classification{}) {
		t.Fatalf("Classify() = (%+v, %t), want (zero, false)", got, ok)
	}
}
