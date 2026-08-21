package oidcjwt

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCAuthenticationErrorsStayPrivate(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		err  error
		code codes.Code
	}{
		{err: NewError(KindInvalid), code: codes.Unauthenticated},
		{err: NewError(KindOversize), code: codes.ResourceExhausted},
		{err: NewError(KindUnavailable), code: codes.Unavailable},
		{err: context.Canceled, code: codes.Canceled},
	} {
		if got := status.Code(grpcAuthenticationError(testCase.err)); got != testCase.code {
			t.Fatalf("status.Code(%v) = %v, want %v", testCase.err, got, testCase.code)
		}
	}
}
