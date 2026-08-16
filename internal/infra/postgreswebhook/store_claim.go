package postgreswebhook

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type ClaimResult struct {
	Attempt  *ClaimedAttempt
	Progress bool
}

//nolint:gocognit,cyclop // Claim exposes the bounded fairness/slot/fence transaction in execution order.
func (s *Store) Claim(ctx context.Context, workerID string, pageSize int, leaseDuration time.Duration, manifest *SecretManifest) (ClaimResult, error) {
	if err := validateToken("worker_id", workerID); err != nil || pageSize < 1 || pageSize > MaxClaimScanPage || leaseDuration <= 0 || manifest == nil || manifest.Revision() != s.options.ManifestRevision {
		return ClaimResult{}, fmt.Errorf("%w: claim inputs are invalid", ErrConfig)
	}
	page, err := int32Value(pageSize)
	if err != nil {
		return ClaimResult{}, err
	}
	var result ClaimResult
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		sampledAt, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		capacity, err := queries.ReadWebhookCapacity(ctx)
		if err != nil {
			return fmt.Errorf("read webhook capacity: %w", err)
		}
		if capacity.RevisionCount != 1 || capacity.CapacityRevision != s.options.CapacityRevision || int(capacity.SlotCount) != s.options.GlobalConcurrency {
			return fmt.Errorf("%w: capacity authority mismatch", ErrConfig)
		}
		maxRevision, err := queries.ReadWebhookMaxRequiredSecretRevision(ctx)
		if err != nil {
			return fmt.Errorf("read webhook manifest requirement: %w", err)
		}
		if maxRevision > manifest.Revision() {
			return fmt.Errorf("%w: secret manifest revision is stale", ErrConfig)
		}
		candidates, err := queries.ListWebhookClaimDestinations(ctx, sqlcgen.ListWebhookClaimDestinationsParams{ManifestRevision: manifest.Revision(), SampledAt: pgtime(sampledAt), PageSize: page})
		if err != nil {
			return fmt.Errorf("list webhook claim destinations: %w", err)
		}
		for _, candidate := range candidates {
			advanced, err := queries.AdvanceWebhookDestinationCursor(ctx, sqlcgen.AdvanceWebhookDestinationCursorParams{SampledAt: pgtime(sampledAt), OwnerScope: candidate.OwnerScope, DestinationID: candidate.DestinationID, Generation: candidate.Generation})
			if err != nil {
				return fmt.Errorf("advance webhook fairness cursor: %w", err)
			}
			result.Progress = result.Progress || advanced == 1
			active, err := queries.CountWebhookDestinationAttempts(ctx, sqlcgen.CountWebhookDestinationAttemptsParams{OwnerScope: candidate.OwnerScope, DestinationID: candidate.DestinationID, Generation: candidate.Generation, SampledAt: pgtime(sampledAt)})
			if err != nil {
				return fmt.Errorf("count webhook destination attempts: %w", err)
			}
			if int(active) >= int(candidate.DestinationConcurrency) {
				continue
			}
			if candidate.ActiveKeyReference == nil {
				return fmt.Errorf("%w: active secret binding is erased", ErrConfig)
			}
			if _, err := manifest.Resolve(candidate.OwnerScope, candidate.DestinationID, *candidate.ActiveKeyReference); err != nil {
				return fmt.Errorf("%w: active secret binding is missing", ErrConfig)
			}
			if candidate.PredecessorKeyReference != nil && candidate.PredecessorValidUntil.Valid && sampledAt.Before(candidate.PredecessorValidUntil.Time) {
				if _, err := manifest.Resolve(candidate.OwnerScope, candidate.DestinationID, *candidate.PredecessorKeyReference); err != nil {
					return fmt.Errorf("%w: predecessor secret binding is missing", ErrConfig)
				}
			}
			delivery, err := queries.LockWebhookDueDelivery(ctx, sqlcgen.LockWebhookDueDeliveryParams{OwnerScope: candidate.OwnerScope, DestinationID: candidate.DestinationID, Generation: candidate.Generation, SampledAt: pgtime(sampledAt)})
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				return fmt.Errorf("lock webhook delivery: %w", err)
			}
			slot, err := queries.LockWebhookCapacitySlot(ctx, s.options.CapacityRevision)
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			if err != nil {
				return fmt.Errorf("lock webhook capacity slot: %w", err)
			}
			attemptID := rand.Text()
			fence := delivery.Fence + 1
			leaseExpiresAt := sampledAt.Add(leaseDuration)
			claimed, err := queries.ClaimWebhookDelivery(ctx, sqlcgen.ClaimWebhookDeliveryParams{WorkerID: &workerID, LeaseExpiresAt: pgtime(leaseExpiresAt), SampledAt: pgtime(sampledAt), OwnerScope: delivery.OwnerScope, DeliveryID: delivery.DeliveryID, PreviousFence: delivery.Fence})
			if err != nil || claimed != 1 {
				if err != nil {
					return fmt.Errorf("claim webhook delivery: %w", err)
				}
				return ErrConflict
			}
			if rows, err := queries.IncrementWebhookCycleAttempts(ctx, sqlcgen.IncrementWebhookCycleAttemptsParams{OwnerScope: delivery.OwnerScope, DeliveryID: delivery.DeliveryID, CycleNumber: delivery.CurrentCycle}); err != nil || rows != 1 {
				if err != nil {
					return fmt.Errorf("advance webhook attempt budget: %w", err)
				}
				return ErrConflict
			}
			if rows, err := queries.LeaseWebhookCapacitySlot(ctx, sqlcgen.LeaseWebhookCapacitySlotParams{OwnerScope: &delivery.OwnerScope, DeliveryID: &delivery.DeliveryID, CycleNumber: &delivery.CurrentCycle, AttemptID: &attemptID, LeaseExpiresAt: pgtime(leaseExpiresAt), Fence: &fence, SlotNumber: slot, CapacityRevision: s.options.CapacityRevision}); err != nil || rows != 1 {
				if err != nil {
					return fmt.Errorf("lease webhook capacity slot: %w", err)
				}
				return ErrConflict
			}
			payloadDigest := sha256.Sum256(delivery.Body)
			payloadBytes, err := int32Value(len(delivery.Body))
			if err != nil {
				return err
			}
			if err := queries.InsertWebhookAttempt(ctx, sqlcgen.InsertWebhookAttemptParams{OwnerScope: delivery.OwnerScope, DeliveryID: delivery.DeliveryID, CycleNumber: delivery.CurrentCycle, AttemptID: attemptID, Fence: fence, CapacitySlot: slot, AttemptedAt: pgtime(sampledAt), LeaseExpiresAt: pgtime(leaseExpiresAt), PayloadDigest: payloadDigest[:], PayloadBytes: payloadBytes, RetainedUntil: delivery.AttemptsRetainedUntil}); err != nil {
				return fmt.Errorf("insert webhook attempt: %w", err)
			}
			var policy DeliveryPolicy
			if err := json.Unmarshal(delivery.PolicySnapshot, &policy); err != nil {
				return fmt.Errorf("%w: decode retained webhook policy", ErrConflict)
			}
			result.Attempt = &ClaimedAttempt{
				Identity:      AttemptIdentity{OwnerScope: delivery.OwnerScope, DeliveryID: delivery.DeliveryID, Cycle: delivery.CurrentCycle, AttemptID: attemptID, Fence: fence},
				AttemptNumber: int(delivery.AttemptsUsed) + 1,
				DestinationID: delivery.DestinationID, DestinationGeneration: delivery.DestinationGeneration,
				URL: delivery.UrlSnapshot, Body: append([]byte(nil), delivery.Body...), ContentType: delivery.ContentType,
				AttemptedAt: sampledAt, Deadline: leaseExpiresAt, KeyReference: *candidate.ActiveKeyReference,
				ManifestRevision: candidate.RequiredSecretRevision, SignatureProfile: candidate.SignatureProfile,
				ControlRevision: candidate.ControlRevision, KeyStateRevision: candidate.KeyStateRevision, Policy: policy,
			}
			if candidate.PredecessorKeyReference != nil && candidate.PredecessorValidUntil.Valid && sampledAt.Before(candidate.PredecessorValidUntil.Time) {
				result.Attempt.PredecessorReference = *candidate.PredecessorKeyReference
			}
			return nil
		}
		return nil
	})
	return result, err
}
