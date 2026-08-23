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

type handler struct {
	river.WorkerDefaults[deliveryArgs]

	secrets  *SecretManifest
	resolver *net.Resolver
}

func AddWorker(workers *river.Workers, secrets *SecretManifest) error {
	if workers == nil {
		return fmt.Errorf("%w: River workers are required", ErrConfig)
	}
	if secrets == nil {
		return fmt.Errorf("%w: secret manifest is required", ErrConfig)
	}
	if err := river.AddWorkerSafely(workers, &handler{secrets: secrets, resolver: &net.Resolver{PreferGo: true}}); err != nil {
		return fmt.Errorf("register webhook worker: %w", err)
	}
	return nil
}

func (*handler) Timeout(*river.Job[deliveryArgs]) time.Duration { return 30 * time.Second }

func (*handler) NextRetry(job *river.Job[deliveryArgs]) time.Time {
	return webhookNextRetry(job, time.Now().UTC())
}

func (h *handler) Work(ctx context.Context, job *river.Job[deliveryArgs]) error {
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
	if webhookDeliveryExpired(job, attemptedAt) {
		return cancelJob("webhook delivery deadline exhausted")
	}
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
	if retryableWebhookStatus(result.Evidence.StatusCode) {
		if hint, ok := parseRetryAfter(result.RetryAfter, result.ResponseDate, attemptedAt, webhookMaxBackoff); ok {
			if err := rememberRetryAfter(ctx, job, attemptedAt.Add(hint)); err != nil {
				return errors.Join(errors.New("webhook delivery retryable"), err)
			}
		}
	}
	return classifyDelivery(result, sendErr)
}

func prepareFailure(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errDestinationDenied):
		return cancelJob("webhook destination denied")
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return context.Canceled
	default:
		return fmt.Errorf("prepare webhook delivery: %w", err)
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
		return fmt.Errorf("webhook delivery retryable: HTTP %d", evidence.StatusCode)
	case evidence.StatusCode >= 100:
		return cancelJob(fmt.Sprintf("webhook receiver rejected delivery: HTTP %d", evidence.StatusCode))
	case evidence.MayHaveSent:
		return deliveryFailure("webhook delivery outcome is ambiguous", err)
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, context.Canceled):
		return context.Canceled
	default:
		return deliveryFailure("webhook delivery failed", err)
	}
}

func deliveryFailure(message string, err error) error {
	if err == nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %w", message, err)
}

func cancelJob(reason string) error {
	return fmt.Errorf("cancel webhook job: %w", river.JobCancel(errors.New(reason)))
}

func retryableWebhookStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests ||
		status >= 500 && status <= 599 && status != http.StatusNotImplemented && status != http.StatusHTTPVersionNotSupported
}
