package postgreswebhook

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func TestWebhookDeliveryClassification(t *testing.T) {
	tests := []struct {
		name    string
		result  SendResult
		err     error
		outcome jobs.OutcomeClass
		effect  jobs.EffectStatus
	}{
		{name: "accepted", result: SendResult{Evidence: TransportEvidence{StatusCode: http.StatusNoContent, MayHaveSent: true}}, outcome: jobs.OutcomeSuccess, effect: jobs.EffectCompleted},
		{name: "rate limited", result: SendResult{Evidence: TransportEvidence{StatusCode: http.StatusTooManyRequests, MayHaveSent: true}}, outcome: jobs.OutcomeRetryable, effect: jobs.EffectUnknown},
		{name: "rejected", result: SendResult{Evidence: TransportEvidence{StatusCode: http.StatusBadRequest, MayHaveSent: true}}, outcome: jobs.OutcomePermanent, effect: jobs.EffectNone},
		{name: "ambiguous", result: SendResult{Evidence: TransportEvidence{MayHaveSent: true}}, outcome: jobs.OutcomeRetryable, effect: jobs.EffectUnknown},
		{name: "local denial", result: SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true, LocalDenial: true}}, outcome: jobs.OutcomePermanent, effect: jobs.EffectNone},
		{name: "deadline", result: SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true}}, err: context.DeadlineExceeded, outcome: jobs.OutcomeTimeout, effect: jobs.EffectNone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := classifyDelivery(test.result, test.err, time.Minute)
			if got.Outcome != test.outcome || got.Effect != test.effect {
				t.Fatalf("classification = %+v, want %s/%s", got, test.outcome, test.effect)
			}
		})
	}
}
