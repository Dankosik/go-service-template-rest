package postgreswebhook

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/riverqueue/river"
)

func TestWebhookDeliveryClassification(t *testing.T) {
	tests := []struct {
		name      string
		result    sendResult
		err       error
		cancelled bool
		retry     bool
	}{
		{name: "accepted", result: sendResult{Evidence: transportEvidence{StatusCode: http.StatusNoContent, MayHaveSent: true}}},
		{name: "rate limited", result: sendResult{Evidence: transportEvidence{StatusCode: http.StatusTooManyRequests, MayHaveSent: true}}, retry: true},
		{name: "rejected", result: sendResult{Evidence: transportEvidence{StatusCode: http.StatusBadRequest, MayHaveSent: true}}, cancelled: true},
		{name: "ambiguous", result: sendResult{Evidence: transportEvidence{MayHaveSent: true}}, retry: true},
		{name: "local denial", result: sendResult{Evidence: transportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, cancelled: true},
		{name: "deadline", result: sendResult{Evidence: transportEvidence{DefinitelyNotSent: true}}, err: context.DeadlineExceeded, retry: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := classifyDelivery(test.result, test.err)
			var cancelErr *river.JobCancelError
			cancelled := errors.As(err, &cancelErr)
			if cancelled != test.cancelled {
				t.Fatalf("cancelled = %t, want %t; error = %v", cancelled, test.cancelled, err)
			}
			if test.retry && err == nil {
				t.Fatal("retryable classification returned nil")
			}
			if !test.retry && !test.cancelled && err != nil {
				t.Fatalf("successful classification error = %v", err)
			}
		})
	}
}

func TestWebhookFailuresPreserveSafeCause(t *testing.T) {
	cause := errors.New("resolver unavailable")
	if err := prepareFailure(t.Context(), cause); !errors.Is(err, cause) {
		t.Fatalf("prepareFailure() error = %v, want cause", err)
	}
	result := sendResult{Evidence: transportEvidence{MayHaveSent: true}}
	if err := classifyDelivery(result, cause); !errors.Is(err, cause) {
		t.Fatalf("classifyDelivery() error = %v, want cause", err)
	}
}
