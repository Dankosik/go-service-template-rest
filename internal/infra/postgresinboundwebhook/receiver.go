// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/riverqueue/rivercontrib/otelriver"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
	"go.opentelemetry.io/otel/metric"
)

const (
	inboundJobKind        = "inbound_webhook_receipt"
	inboundJobMaxAttempts = 25
	signatureTolerance    = 5 * time.Minute
)

type receiptJobArgs struct {
	ReceiptID string `json:"receipt_id"`
}

func (receiptJobArgs) Kind() string { return inboundJobKind }

type receiptRecord struct {
	ReceiptID  string
	EndpointID string
	DeliveryID string
	BodySHA256 [sha256.Size]byte
	SignedAt   time.Time
	Payload    []byte
}

type receiptAcceptor interface {
	Accept(ctx context.Context, record receiptRecord) (inboundwebhook.Outcome, error)
}

// Receiver implements inboundwebhook.Receiver against PostgreSQL and River.
type Receiver struct {
	trust *TrustManifest
	store receiptAcceptor
	now   func() time.Time
	telem telemetry
}

// ReceiverOption adjusts optional receiver collaborators such as the clock.
type ReceiverOption func(*Receiver)

// WithClock injects the verification clock.
func WithClock(now func() time.Time) ReceiverOption {
	return func(r *Receiver) {
		if now != nil {
			r.now = now
		}
	}
}

func withStore(store receiptAcceptor) ReceiverOption {
	return func(r *Receiver) {
		if store != nil {
			r.store = store
		}
	}
}

// NewReceiver builds the concrete acceptance adapter. A nil meter disables
// ingress metrics.
func NewReceiver(pool *pgxpool.Pool, trust *TrustManifest, meter metric.MeterProvider, opts ...ReceiverOption) (*Receiver, error) {
	if trust == nil {
		return nil, errors.New("inbound webhook trust manifest is required")
	}
	receiver := &Receiver{
		trust: trust,
		now:   func() time.Time { return time.Now().UTC() },
		telem: newTelemetry(meter, nil),
	}
	for _, opt := range opts {
		opt(receiver)
	}
	if receiver.store == nil {
		if pool == nil {
			return nil, errors.New("inbound webhook postgres pool is required")
		}
		store, err := newPostgresStore(pool)
		if err != nil {
			return nil, err
		}
		receiver.store = store
	}
	return receiver, nil
}

// Receive verifies then durably accepts one signed delivery.
func (r *Receiver) Receive(ctx context.Context, delivery inboundwebhook.Delivery) (inboundwebhook.Outcome, error) {
	if r == nil || r.trust == nil || r.store == nil {
		return "", inboundwebhook.ErrUnavailable
	}
	delivery = delivery.Clone()
	if _, ok := r.trust.Lookup(delivery.EndpointID); !ok {
		return inboundwebhook.OutcomeUnknownEndpoint, nil
	}
	signedAt, ok := r.verify(delivery)
	if !ok {
		r.telem.recordIngress(ctx, inboundwebhook.OutcomeRejected)
		return inboundwebhook.OutcomeRejected, nil
	}
	digest := sha256.Sum256(delivery.Body)
	outcome, err := r.store.Accept(ctx, receiptRecord{
		ReceiptID:  rand.Text(),
		EndpointID: delivery.EndpointID,
		DeliveryID: delivery.DeliveryID,
		BodySHA256: digest,
		SignedAt:   signedAt,
		Payload:    delivery.Body,
	})
	if err != nil {
		// runInTx can join ctx.Err() with commit-unknown; an unknown commit stays unavailable.
		canceled := errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
		if canceled && !errors.Is(err, postgres.ErrCommitUnknown) {
			return "", fmt.Errorf("accept inbound webhook receipt: %w", err)
		}
		r.telem.recordIngress(ctx, ingressUnavailable)
		return "", inboundwebhook.ErrUnavailable
	}
	r.telem.recordIngress(ctx, outcome)
	return outcome, nil
}

// verify checks the delivery ID, timestamp, and tolerance before the signature.
func (r *Receiver) verify(delivery inboundwebhook.Delivery) (time.Time, bool) {
	if !validDeliveryID(delivery.DeliveryID) {
		return time.Time{}, false
	}
	signedAt, ok := parseSignedTimestamp(delivery.Timestamp)
	if !ok || !timestampInTolerance(signedAt, r.now()) {
		return time.Time{}, false
	}
	if !r.signatureOK(delivery) {
		return time.Time{}, false
	}
	return signedAt, true
}

func (r *Receiver) signatureOK(delivery inboundwebhook.Delivery) bool {
	secrets, ok := r.trust.secretsFor(delivery.EndpointID)
	if !ok {
		return false
	}
	headers := make(http.Header, 3)
	headers.Set(standardwebhooks.HeaderWebhookID, delivery.DeliveryID)
	headers.Set(standardwebhooks.HeaderWebhookTimestamp, delivery.Timestamp)
	headers.Set(standardwebhooks.HeaderWebhookSignature, delivery.Signature)
	if verifyWith(secrets.active, delivery.Body, headers) {
		return true
	}
	return len(secrets.predecessor) > 0 && verifyWith(secrets.predecessor, delivery.Body, headers)
}

func verifyWith(key, body []byte, headers http.Header) bool {
	webhook, err := standardwebhooks.NewWebhookRaw(key)
	if err != nil {
		return false
	}
	return webhook.VerifyIgnoringTimestamp(body, headers) == nil
}

func parseSignedTimestamp(raw string) (time.Time, bool) {
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0).UTC(), true
}

func timestampInTolerance(signedAt, now time.Time) bool {
	delta := now.Sub(signedAt)
	return delta <= signatureTolerance && delta >= -signatureTolerance
}

type postgresStore struct {
	pool *pgxpool.Pool
	inTx func(context.Context, *pgxpool.Pool, pgx.TxOptions, func(pgx.Tx) error) error
	jobs *river.Client[pgx.Tx]
}

func newPostgresStore(pool *pgxpool.Pool) (*postgresStore, error) {
	client, err := river.NewClient(riverpgxv5.New(nil), &river.Config{
		Plugins: []rivertype.Plugin{
			otelriver.NewMiddleware(&otelriver.MiddlewareConfig{EnableTracePropagation: true}),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize inbound webhook River producer: %w", err)
	}
	return &postgresStore{pool: pool, inTx: postgres.InTx, jobs: client}, nil
}

func (s *postgresStore) Accept(ctx context.Context, record receiptRecord) (inboundwebhook.Outcome, error) {
	var outcome inboundwebhook.Outcome
	err := s.inTx(ctx, s.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		inserted, err := queries.ClaimInboundWebhookReceipt(ctx, sqlcgen.ClaimInboundWebhookReceiptParams{
			ReceiptID:  record.ReceiptID,
			EndpointID: record.EndpointID,
			DeliveryID: record.DeliveryID,
			BodySha256: record.BodySHA256[:],
			SignedAt:   pgtype.Timestamptz{Time: record.SignedAt, Valid: true},
			Payload:    record.Payload,
		})
		if err != nil {
			return fmt.Errorf("claim inbound webhook receipt: %w", err)
		}
		if inserted == 1 {
			if _, err := s.jobs.InsertTx(ctx, tx, receiptJobArgs{ReceiptID: record.ReceiptID}, &river.InsertOpts{
				MaxAttempts: inboundJobMaxAttempts,
			}); err != nil {
				return fmt.Errorf("insert inbound webhook job: %w", err)
			}
			outcome = inboundwebhook.OutcomeAccepted
			return nil
		}
		existing, err := queries.GetInboundWebhookReceiptByIdentity(ctx, sqlcgen.GetInboundWebhookReceiptByIdentityParams{
			EndpointID: record.EndpointID,
			DeliveryID: record.DeliveryID,
		})
		if err != nil {
			return fmt.Errorf("read inbound webhook receipt: %w", err)
		}
		if bytes.Equal(existing.BodySha256, record.BodySHA256[:]) {
			outcome = inboundwebhook.OutcomeDuplicate
			return nil
		}
		outcome = inboundwebhook.OutcomeConflict
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("accept inbound webhook receipt: %w", err)
	}
	return outcome, nil
}

// profile:inbound-webhooks-standard:end
