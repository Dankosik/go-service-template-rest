package oauth2clientcredentials

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type canaryContextError struct {
	cause error
}

func (e *canaryContextError) Error() string { return forbiddenCanary }
func (e *canaryContextError) Unwrap() error { return e.cause }

func TestAuthErrorsAreSanitized(t *testing.T) {
	classes := []FailureClass{
		FailureInvalidConfiguration,
		FailureEndpointTrust,
		FailureCallerCanceled,
		FailureProviderTimeout,
		FailureProviderUnavailable,
		FailureClientRejected,
		FailureGrantRejected,
		FailureUnsupportedResponse,
		FailureTokenUnusable,
		FailureDownstreamUnauthenticated,
		FailureDownstreamForbidden,
	}
	canaryCause := fmt.Errorf("raw provider cause: %s", forbiddenCanary)
	for _, class := range classes {
		t.Run(string(class), func(t *testing.T) {
			cause := canaryCause
			if class == FailureCallerCanceled {
				cause = context.Canceled
			}
			err := &authError{class: class, cause: cause}
			assertFailureClass(t, err, class)
			if err.Error() != failureMessages[class] {
				t.Fatalf("Error() = %q, want %q", err, failureMessages[class])
			}
			if class != FailureCallerCanceled && errors.Is(err, canaryCause) {
				t.Fatal("auth error retained a raw provider cause")
			}
		})
	}

	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		err := callerFailure(cause)
		assertFailureClass(t, err, FailureCallerCanceled)
		if !errors.Is(err, cause) {
			t.Fatalf("callerFailure(%v) did not preserve safe context identity", cause)
		}
	}
	wrappedContext := &canaryContextError{cause: context.Canceled}
	normalized := callerFailure(wrappedContext)
	assertFailureClass(t, normalized, FailureCallerCanceled)
	if !errors.Is(normalized, context.Canceled) || errors.Is(normalized, wrappedContext) {
		t.Fatal("callerFailure() did not reduce a wrapped cause to the safe context identity")
	}
	if _, leaked := errors.AsType[*canaryContextError](normalized); leaked {
		t.Fatal("callerFailure() retained the wrapped caller cause")
	}

	invalidCause := fmt.Errorf("%s: not a context outcome", forbiddenCanary)
	invalid := callerFailure(invalidCause)
	assertFailureClass(t, invalid, FailureCallerCanceled)
	if errors.Is(invalid, invalidCause) {
		t.Fatal("callerFailure() retained a non-context cause")
	}
	crafted := &authError{class: FailureCallerCanceled, cause: invalidCause}
	if errors.Is(crafted, invalidCause) {
		t.Fatal("authError.Unwrap() exposed a non-context caller cause")
	}

	unknown := &authError{class: "unknown", cause: canaryCause}
	if class, ok := FailureClassOf(unknown); ok || class != "" {
		t.Fatalf("FailureClassOf(unknown) = %q, %t", class, ok)
	}
	if strings.Contains(unknown.Error(), forbiddenCanary) || unknown.Error() != "outbound authentication failed" {
		t.Fatalf("unknown error = %q", unknown)
	}
	if _, ok := FailureClassOf(errors.New("ordinary downstream response")); ok {
		t.Fatal("ordinary downstream error was classified as an auth error")
	}
}
