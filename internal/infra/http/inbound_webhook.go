// profile:inbound-webhooks-standard:start
package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
)

const inboundUnavailableRetryAfter = time.Second

var errInboundWebhookStrictFallback = errors.New("inbound webhook strict fallback is unreachable")

// ReceiveWebhook fails closed if the raw standard-server override is ever
// bypassed. The transport owns both paths; bootstrap does not implement an
// operation it never serves.
func (strictHandlers) ReceiveWebhook(context.Context, openapi.ReceiveWebhookRequestObject) (openapi.ReceiveWebhookResponseObject, error) {
	return nil, errInboundWebhookStrictFallback
}

type inboundRawServer struct {
	openapi.ServerInterface

	receiver inboundwebhook.Receiver
}

func (s inboundRawServer) ReceiveWebhook(w http.ResponseWriter, r *http.Request, endpointID string, params openapi.ReceiveWebhookParams) {
	if s.receiver == nil {
		writeInboundUnavailable(w, r, "inbound webhook receiver is unavailable")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if response, ok := contextFailureProblem(err); ok {
			writeProblem(w, r, response)
			return
		}
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeProblem(w, r, requestEntityTooLargeProblem())
			return
		}
		writeProblem(w, r, inboundFailureProblem())
		return
	}
	outcome, receiveErr := s.receiver.Receive(r.Context(), inboundwebhook.Delivery{
		EndpointID: endpointID,
		DeliveryID: params.WebhookId,
		Timestamp:  params.WebhookTimestamp,
		Signature:  params.WebhookSignature,
		Body:       body,
	})
	if receiveErr != nil {
		if response, ok := contextFailureProblem(receiveErr); ok {
			writeProblem(w, r, response)
			return
		}
		switch {
		case errors.Is(receiveErr, inboundwebhook.ErrUnavailable):
			writeInboundUnavailable(w, r, "inbound webhook storage is unavailable")
		default:
			writeProblem(w, r, inboundFailureProblem())
		}
		return
	}
	switch outcome {
	case inboundwebhook.OutcomeAccepted, inboundwebhook.OutcomeDuplicate:
		w.WriteHeader(http.StatusNoContent)
	case inboundwebhook.OutcomeUnknownEndpoint:
		writeProblem(w, r, problemResponse{code: problem.CodeNotFound, detail: "inbound webhook endpoint is unknown"})
	case inboundwebhook.OutcomeRejected:
		writeProblem(w, r, problemResponse{code: problem.CodeBadRequest, detail: malformedRequestProblemDetail})
	case inboundwebhook.OutcomeConflict:
		writeProblem(w, r, problemResponse{code: problem.CodeConflict, detail: "inbound webhook delivery conflicts"})
	default:
		writeProblem(w, r, inboundFailureProblem())
	}
}

// writeInboundUnavailable answers a delivery this instance cannot take right now;
// the sender retries after the short hint.
func writeInboundUnavailable(w http.ResponseWriter, r *http.Request, detail string) {
	writeProblem(w, r, problemResponse{
		code:       problem.CodeServiceUnavailable,
		detail:     detail,
		retryAfter: inboundUnavailableRetryAfter,
	})
}

// inboundFailureProblem is the sanitized answer for a delivery that failed for a
// reason the sender cannot act on.
func inboundFailureProblem() problemResponse {
	return problemResponse{code: problem.CodeInternalError, detail: "inbound webhook request failed"}
}

// profile:inbound-webhooks-standard:end
