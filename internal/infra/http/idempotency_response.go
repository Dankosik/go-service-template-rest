package httpx

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/problem"
)

func writeIdempotencyBadKey(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, problemResponse{code: problem.CodeBadRequest, detail: "idempotency key is missing or invalid"})
}

func writeIdempotencyForbidden(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, r, problemResponse{code: problem.CodeForbidden, detail: "request is not authorized"})
}

// writeIdempotencyDecision returns false only for execute, which is the one
// disposition that must continue into the endpoint's caller-owned transaction.
func writeIdempotencyDecision(w http.ResponseWriter, r *http.Request, contract httpidempotency.Contract, decision httpidempotency.Decision) bool {
	if err := decision.Validate(); err != nil {
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "idempotency decision is invalid"})
		return true
	}
	switch decision.Outcome {
	case httpidempotency.OutcomeExecute:
		return false
	case httpidempotency.OutcomeReplay:
		if err := writeIdempotencyReplay(w, contract, *decision.Result); err != nil {
			writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "idempotency retained result is invalid"})
		}
	case httpidempotency.OutcomeMismatch:
		writeProblem(w, r, problemResponse{code: problem.CodeIdempotencyKeyMismatch, detail: "idempotency key was used for a different request"})
	case httpidempotency.OutcomeInProgress:
		writeIdempotencyRetryProblem(w, r, contract.RetryAfter, problem.CodeIdempotencyInProgress, "idempotency request is in progress")
	case httpidempotency.OutcomeExpired:
		writeProblem(w, r, problemResponse{code: problem.CodeIdempotencyKeyExpired, detail: "idempotency key has expired"})
	case httpidempotency.OutcomeRateLimited:
		writeIdempotencyRetryProblem(w, r, contract.RetryAfter, problem.CodeTooManyRequests, "idempotency admission is limited")
	case httpidempotency.OutcomeUnavailable:
		writeIdempotencyRetryProblem(w, r, contract.RetryAfter, problem.CodeIdempotencyUnavailable, "idempotency service is unavailable")
	case httpidempotency.OutcomeUnknown:
		writeIdempotencyRetryProblem(w, r, contract.RetryAfter, problem.CodeIdempotencyOutcomeUnknown, "idempotency outcome is unknown")
	case httpidempotency.OutcomeResultTooLarge:
		writeProblem(w, r, problemResponse{code: problem.CodeIdempotencyResultTooLarge, detail: "idempotency result cannot be retained"})
	case httpidempotency.OutcomeIntegrityConflict:
		writeProblem(w, r, problemResponse{code: problem.CodeInternalError, detail: "idempotency evidence is inconsistent"})
	}
	return true
}

func writeIdempotencyRetryProblem(w http.ResponseWriter, r *http.Request, retryAfterDuration time.Duration, code problem.Code, detail string) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retryAfterDuration)))
	writeProblem(w, r, problemResponse{code: code, detail: detail})
}

func writeIdempotencyReplay(w http.ResponseWriter, contract httpidempotency.Contract, result httpidempotency.Result) error {
	encoded, err := httpidempotency.EncodeResult(contract, result)
	if err != nil {
		//nolint:wrapcheck // The envelope deliberately maps either codec failure to one closed Problem.
		return err
	}
	result, err = httpidempotency.DecodeResult(contract, encoded)
	if err != nil {
		//nolint:wrapcheck // The envelope deliberately maps either codec failure to one closed Problem.
		return err
	}
	for header, values := range result.Headers {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.Header().Set("Content-Type", result.MediaType)
	w.WriteHeader(result.Status)
	_, _ = w.Write(result.Payload)
	return nil
}
