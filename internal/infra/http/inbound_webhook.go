// profile:inbound-webhooks-standard:start
package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
)

const inboundUnavailableRetryAfter = 1

type inboundRawServer struct {
	openapi.ServerInterface

	receiver inboundwebhook.Receiver
}

func (s inboundRawServer) ReceiveWebhook(w http.ResponseWriter, r *http.Request, endpointID string, params openapi.ReceiveWebhookParams) {
	if s.receiver == nil {
		w.Header().Set("Retry-After", strconv.Itoa(inboundUnavailableRetryAfter))
		writeProblem(w, r, problemResponse{code: problem.CodeServiceUnavailable, detail: "inbound webhook receiver is unavailable"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeProblem(w, r, timeBudgetExceededProblem())
			return
		}
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "inbound webhook request failed"})
		return
	}
	result, receiveErr := s.receiver.Receive(r.Context(), inboundwebhook.Delivery{
		EndpointID: endpointID,
		DeliveryID: params.WebhookId,
		Timestamp:  params.WebhookTimestamp,
		Signature:  params.WebhookSignature,
		Body:       body,
	})
	if receiveErr != nil {
		switch {
		case errors.Is(receiveErr, context.DeadlineExceeded), errors.Is(receiveErr, context.Canceled):
			writeProblem(w, r, timeBudgetExceededProblem())
		case errors.Is(receiveErr, inboundwebhook.ErrUnavailable):
			w.Header().Set("Retry-After", strconv.Itoa(inboundUnavailableRetryAfter))
			writeProblem(w, r, problemResponse{code: problem.CodeServiceUnavailable, detail: "inbound webhook storage is unavailable"})
		default:
			writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "inbound webhook request failed"})
		}
		return
	}
	switch result.Outcome {
	case inboundwebhook.OutcomeAccepted, inboundwebhook.OutcomeDuplicate:
		w.WriteHeader(http.StatusNoContent)
	case inboundwebhook.OutcomeUnknownEndpoint:
		writeProblem(w, r, problemResponse{code: problem.CodeNotFound, detail: "inbound webhook endpoint is unknown"})
	case inboundwebhook.OutcomeRejected:
		writeProblem(w, r, problemResponse{code: problem.CodeBadRequest, detail: malformedRequestProblemDetail})
	case inboundwebhook.OutcomeConflict:
		writeProblem(w, r, problemResponse{code: problem.CodeConflict, detail: "inbound webhook delivery conflicts"})
	case inboundwebhook.OutcomeUnavailable:
		w.Header().Set("Retry-After", strconv.Itoa(inboundUnavailableRetryAfter))
		writeProblem(w, r, problemResponse{code: problem.CodeServiceUnavailable, detail: "inbound webhook storage is unavailable"})
	default:
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "inbound webhook request failed"})
	}
}

// profile:inbound-webhooks-standard:end
