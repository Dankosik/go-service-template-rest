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
		result    SendResult
		err       error
		cancelled bool
		retry     bool
	}{
		{name: "accepted", result: SendResult{Evidence: TransportEvidence{StatusCode: http.StatusNoContent, MayHaveSent: true}}},
		{name: "rate limited", result: SendResult{Evidence: TransportEvidence{StatusCode: http.StatusTooManyRequests, MayHaveSent: true}}, retry: true},
		{name: "rejected", result: SendResult{Evidence: TransportEvidence{StatusCode: http.StatusBadRequest, MayHaveSent: true}}, cancelled: true},
		{name: "ambiguous", result: SendResult{Evidence: TransportEvidence{MayHaveSent: true}}, retry: true},
		{name: "local denial", result: SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, cancelled: true},
		{name: "deadline", result: SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true}}, err: context.DeadlineExceeded, retry: true},
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
