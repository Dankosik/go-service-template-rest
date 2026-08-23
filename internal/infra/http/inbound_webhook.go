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
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeProblem(w, r, problemResponse{code: problem.CodeRequestEntityTooLarge, detail: "request body exceeds limit"})
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
