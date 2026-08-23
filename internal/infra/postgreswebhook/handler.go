package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/riverqueue/river"
)

type Handler struct {
	river.WorkerDefaults[deliveryArgs]

	secrets  *SecretManifest
	resolver *net.Resolver
}

func NewHandler(secrets *SecretManifest) (*Handler, error) {
	if secrets == nil {
		return nil, fmt.Errorf("%w: secret manifest is required", ErrConfig)
	}
	return &Handler{secrets: secrets, resolver: &net.Resolver{PreferGo: true}}, nil
}

func AddWorker(workers *river.Workers, secrets *SecretManifest) error {
	if workers == nil {
		return fmt.Errorf("%w: River workers are required", ErrConfig)
	}
	handler, err := NewHandler(secrets)
	if err != nil {
		return err
	}
	if err := river.AddWorkerSafely(workers, handler); err != nil {
		return fmt.Errorf("register webhook worker: %w", err)
	}
	return nil
}

func (*Handler) Timeout(*river.Job[deliveryArgs]) time.Duration { return 30 * time.Second }

func (h *Handler) Work(ctx context.Context, job *river.Job[deliveryArgs]) error {
	if h == nil || h.resolver == nil || h.secrets == nil || job == nil {
		return cancelJob("webhook worker is not configured")
	}
	if err := job.Args.validate(); err != nil {
		return cancelJob("webhook job arguments are invalid")
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return cancelJob("webhook attempt deadline is required")
	}
	attemptedAt := time.Now().UTC()
	attempt := deliveryAttempt{
		ID: job.Args.DeliveryID, OwnerScope: job.Args.OwnerScope,
		ReceiverID: job.Args.ReceiverID, URL: job.Args.URL,
		Body: job.Args.Body, AttemptedAt: attemptedAt, Deadline: deadline,
		KeyReference:         job.Args.ActiveKeyReference,
		PredecessorReference: job.Args.PredecessorKeyReference,
	}
	prepared, err := prepareSend(ctx, h.resolver, attempt, h.secrets)
	if err != nil {
		return prepareFailure(ctx, err)
	}
	result, sendErr := tryPreparedAddresses(ctx, prepared, send)
	return classifyDelivery(result, sendErr)
}

func prepareFailure(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrDestinationDenied):
		return cancelJob("webhook destination denied")
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return context.Canceled
	default:
		return errors.New("prepare webhook delivery")
	}
}

func classifyDelivery(result sendResult, err error) error {
	evidence := result.Evidence
	switch {
	case evidence.LocalDenial:
		return cancelJob("webhook destination denied")
	case evidence.StatusCode >= http.StatusOK && evidence.StatusCode <= 299:
		return nil
	case retryableWebhookStatus(evidence.StatusCode):
		return errors.New("webhook delivery retryable")
	case evidence.StatusCode >= 100:
		return cancelJob("webhook receiver rejected delivery")
	case evidence.MayHaveSent:
		return errors.New("webhook delivery outcome is ambiguous")
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, context.Canceled):
		return context.Canceled
	default:
		return errors.New("webhook delivery failed")
	}
}

func cancelJob(reason string) error {
	return fmt.Errorf("cancel webhook job: %w", river.JobCancel(errors.New(reason)))
}

func retryableWebhookStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests ||
		status >= 500 && status <= 599 && status != http.StatusNotImplemented && status != http.StatusHTTPVersionNotSupported
}
