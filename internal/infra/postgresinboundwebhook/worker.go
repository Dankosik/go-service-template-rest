// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
)

const (
	quarantineReasonInvalidJSON = "invalid_json"
	quarantineReasonRejected    = "schema_rejected"
	// resnoozeDelay re-checks a receipt later; a River snooze does not consume an attempt.
	resnoozeDelay = time.Second
)

// Receipt states match the outcome CHECK in the inbound webhook receipts table.
const (
	receiptPending     = "pending"
	receiptHandled     = "handled"
	receiptQuarantined = "quarantined"
	receiptFailed      = "failed"
)

type storedReceipt struct {
	ReceiptID  string
	EndpointID string
	DeliveryID string
	SignedAt   time.Time
	ReceivedAt time.Time
	Payload    []byte
	State      string
}

// receiptRows reads and terminalizes receipts; the worker never enqueues jobs.
type receiptRows struct {
	pool *pgxpool.Pool
}

func (r receiptRows) loadByID(ctx context.Context, receiptID string) (storedReceipt, error) {
	row, err := sqlcgen.New(r.pool).GetInboundWebhookReceiptByID(ctx, receiptID)
	if err != nil {
		return storedReceipt{}, fmt.Errorf("load inbound webhook receipt: %w", err)
	}
	return storedReceipt{
		ReceiptID:  row.ReceiptID,
		EndpointID: row.EndpointID,
		DeliveryID: row.DeliveryID,
		SignedAt:   row.SignedAt.Time,
		ReceivedAt: row.ReceivedAt.Time,
		Payload:    row.Payload,
		State:      row.Outcome,
	}, nil
}

func (r receiptRows) MarkHandled(ctx context.Context, receiptID string) (bool, error) {
	n, err := sqlcgen.New(r.pool).MarkInboundWebhookHandled(ctx, receiptID)
	if err != nil {
		return false, fmt.Errorf("mark inbound webhook handled: %w", err)
	}
	return n == 1, nil
}

func (r receiptRows) MarkQuarantined(ctx context.Context, receiptID, reason string) (bool, error) {
	n, err := sqlcgen.New(r.pool).MarkInboundWebhookQuarantined(ctx, sqlcgen.MarkInboundWebhookQuarantinedParams{
		ReceiptID:      receiptID,
		TerminalReason: &reason,
	})
	if err != nil {
		return false, fmt.Errorf("mark inbound webhook quarantined: %w", err)
	}
	return n == 1, nil
}

func (r receiptRows) MarkFailed(ctx context.Context, receiptID string) (bool, error) {
	n, err := sqlcgen.New(r.pool).MarkInboundWebhookFailed(ctx, receiptID)
	if err != nil {
		return false, fmt.Errorf("mark inbound webhook failed: %w", err)
	}
	return n == 1, nil
}

type receiptStateStore interface {
	loadByID(ctx context.Context, receiptID string) (storedReceipt, error)
	MarkHandled(ctx context.Context, receiptID string) (bool, error)
	MarkQuarantined(ctx context.Context, receiptID, reason string) (bool, error)
	MarkFailed(ctx context.Context, receiptID string) (bool, error)
}

// Worker processes one inbound receipt job.
type Worker struct {
	river.WorkerDefaults[receiptJobArgs]

	store    receiptStateStore
	registry *inboundwebhook.Registry
	telem    telemetry
}

// newWorker builds the River worker.
func newWorker(store receiptStateStore, registry *inboundwebhook.Registry, telem telemetry) (*Worker, error) {
	if store == nil || registry == nil {
		return nil, errors.New("inbound webhook store and registry are required")
	}
	return &Worker{store: store, registry: registry, telem: telem}, nil
}

// AddWorker registers the inbound worker on workers.
func AddWorker(workers *river.Workers, pool *pgxpool.Pool, registry *inboundwebhook.Registry, meter metric.MeterProvider, log *slog.Logger) error {
	if workers == nil || pool == nil {
		return errors.New("inbound webhook workers and postgres pool are required")
	}
	worker, err := newWorker(receiptRows{pool: pool}, registry, newTelemetry(meter, log))
	if err != nil {
		return err
	}
	if err := river.AddWorkerSafely(workers, worker); err != nil {
		return fmt.Errorf("register inbound webhook worker: %w", err)
	}
	return nil
}

func (*Worker) Timeout(*river.Job[receiptJobArgs]) time.Duration { return 30 * time.Second }

//nolint:cyclop // One River lifecycle owner keeps retry and terminal state in one linear path.
func (w *Worker) Work(ctx context.Context, job *river.Job[receiptJobArgs]) (err error) {
	if w == nil || w.store == nil || w.registry == nil || job == nil {
		return errStorageUnavailable
	}
	receiptID := job.Args.ReceiptID
	if receiptID == "" {
		return errStorageUnavailable
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			w.telem.recordRetryingFailure(ctx, receiptID, logClassPanicRecovered)
			err = errPanicRecovered
		}
	}()

	receipt, err := w.store.loadByID(ctx, receiptID)
	if err != nil {
		w.telem.recordRetryingFailure(ctx, receiptID, logClassStorageRetryable)
		if job.Attempt >= job.MaxAttempts {
			return river.JobSnooze(resnoozeDelay)
		}
		return errStorageUnavailable
	}
	if receipt.State != receiptPending {
		return nil
	}
	if !w.registry.HasBinding(receipt.EndpointID) {
		w.telem.recordRetryingFailure(ctx, receiptID, logClassBindingUnavailable)
		// ponytail: reuse the existing snooze; isolate a queue if binding drift becomes load.
		return river.JobSnooze(resnoozeDelay)
	}
	if job.Attempt >= job.MaxAttempts {
		return w.finalize(ctx, receiptID)
	}

	delivery := inboundwebhook.VerifiedDelivery{
		EndpointID: receipt.EndpointID,
		DeliveryID: receipt.DeliveryID,
		SignedAt:   receipt.SignedAt,
		Body:       json.RawMessage(receipt.Payload),
		ReceivedAt: receipt.ReceivedAt,
	}

	dispatchErr := w.registry.Dispatch(ctx, delivery)
	switch {
	case dispatchErr == nil:
		updated, markErr := w.store.MarkHandled(ctx, receiptID)
		if markErr != nil {
			w.telem.recordRetryingFailure(ctx, receiptID, logClassStorageRetryable)
			return errStorageUnavailable
		}
		if updated {
			w.telem.recordProcessing(ctx, receiptHandled)
		}
		return nil
	case inboundwebhook.IsDecodeError(dispatchErr) && errors.Is(dispatchErr, inboundwebhook.ErrDecodeRejected):
		reason := quarantineReasonRejected
		if !json.Valid(receipt.Payload) {
			reason = quarantineReasonInvalidJSON
		}
		updated, markErr := w.store.MarkQuarantined(ctx, receiptID, reason)
		if markErr != nil {
			w.telem.recordRetryingFailure(ctx, receiptID, logClassStorageRetryable)
			return errStorageUnavailable
		}
		if updated {
			w.telem.recordProcessing(ctx, receiptQuarantined)
		}
		return nil
	case inboundwebhook.IsDecodeError(dispatchErr):
		w.telem.recordRetryingFailure(ctx, receiptID, logClassDecoderInternal)
		return errDecoderFailed
	default:
		w.telem.recordRetryingFailure(ctx, receiptID, logClassHandlerRetryable)
		return errHandlerFailed
	}
}

func (w *Worker) finalize(ctx context.Context, receiptID string) error {
	updated, err := w.store.MarkFailed(ctx, receiptID)
	if err != nil {
		w.telem.logFailure(ctx, receiptID, logClassTerminalizationRetryable)
		return river.JobSnooze(resnoozeDelay)
	}
	if updated {
		w.telem.recordProcessing(ctx, receiptFailed)
	}
	return nil
}

// profile:inbound-webhooks-standard:end
