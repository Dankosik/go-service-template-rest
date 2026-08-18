package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

type Handler struct {
	secrets  *SecretManifest
	resolver *net.Resolver
}

func NewHandler(secrets *SecretManifest) (*Handler, error) {
	if secrets == nil {
		return nil, fmt.Errorf("%w: secret manifest is required", ErrConfig)
	}
	return &Handler{secrets: secrets, resolver: &net.Resolver{PreferGo: true}}, nil
}

func NewRegistry(secrets *SecretManifest) (*jobs.Registry, error) {
	definition, err := deliveryDefinition()
	if err != nil {
		return nil, err
	}
	handler, err := NewHandler(secrets)
	if err != nil {
		return nil, err
	}
	registry := new(jobs.Registry)
	if err := jobs.Register(registry, definition, handler.Handle); err != nil {
		return nil, fmt.Errorf("register webhook definition: %w", err)
	}
	return registry, nil
}

func (h *Handler) Handle(ctx context.Context, input jobs.HandlerInput[deliveryArgs]) jobs.HandlerResult {
	if h == nil || h.resolver == nil || h.secrets == nil {
		return jobs.HandlerResult{Outcome: jobs.OutcomePoison, Effect: jobs.EffectNone}
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return jobs.HandlerResult{Outcome: jobs.OutcomePoison, Effect: jobs.EffectNone}
	}
	attemptedAt := time.Now().UTC()
	attempt := DeliveryAttempt{
		ID: string(input.Identity.LogicalJobID), OwnerScope: input.Arguments.OwnerScope,
		ReceiverID: input.Arguments.ReceiverID, URL: input.Arguments.URL,
		Body: input.Arguments.Body, AttemptedAt: attemptedAt, Deadline: deadline,
		KeyReference:         input.Arguments.ActiveKeyReference,
		PredecessorReference: input.Arguments.PredecessorKeyReference,
	}
	prepared, err := PrepareSend(ctx, h.resolver, attempt, h.secrets)
	if err != nil {
		return prepareFailure(ctx, err)
	}
	result, sendErr := tryPreparedAddresses(ctx, prepared, Send)
	hint, _ := ParseRetryAfter(result.RetryAfter, result.ResponseDate, attemptedAt, 24*time.Hour)
	return classifyDelivery(result, sendErr, hint)
}

func prepareFailure(ctx context.Context, err error) jobs.HandlerResult {
	switch {
	case errors.Is(err, ErrDestinationDenied):
		return jobs.HandlerResult{Outcome: jobs.OutcomePermanent, Effect: jobs.EffectNone}
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return jobs.HandlerResult{Outcome: jobs.OutcomeTimeout, Effect: jobs.EffectNone}
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectNone}
	default:
		return jobs.HandlerResult{Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone}
	}
}

func classifyDelivery(result SendResult, err error, hint time.Duration) jobs.HandlerResult {
	evidence := result.Evidence
	switch {
	case evidence.LocalDenial:
		return jobs.HandlerResult{Outcome: jobs.OutcomePermanent, Effect: jobs.EffectNone}
	case evidence.StatusCode >= http.StatusOK && evidence.StatusCode <= 299:
		return jobs.HandlerResult{Outcome: jobs.OutcomeSuccess, Effect: jobs.EffectCompleted}
	case retryableWebhookStatus(evidence.StatusCode):
		return jobs.HandlerResult{Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectUnknown, RetryHint: hint}
	case evidence.StatusCode >= 100:
		return jobs.HandlerResult{Outcome: jobs.OutcomePermanent, Effect: jobs.EffectNone}
	case evidence.MayHaveSent:
		return jobs.HandlerResult{Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectUnknown}
	case errors.Is(err, context.DeadlineExceeded):
		return jobs.HandlerResult{Outcome: jobs.OutcomeTimeout, Effect: jobs.EffectNone}
	case errors.Is(err, context.Canceled):
		return jobs.HandlerResult{Outcome: jobs.OutcomeCancelled, Effect: jobs.EffectNone}
	default:
		return jobs.HandlerResult{Outcome: jobs.OutcomeRetryable, Effect: jobs.EffectNone}
	}
}

func retryableWebhookStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests ||
		status >= 500 && status <= 599 && status != http.StatusNotImplemented && status != http.StatusHTTPVersionNotSupported
}
