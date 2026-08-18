package postgreswebhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/samber/lo"
)

type AcceptanceDisposition string

const (
	AcceptanceAccepted       AcceptanceDisposition = "accepted"
	AcceptanceRejected       AcceptanceDisposition = "rejected"
	AcceptanceConflict       AcceptanceDisposition = "conflict"
	AcceptancePrivacyDeleted AcceptanceDisposition = "privacy_deleted"
	AcceptanceUnknown        AcceptanceDisposition = "unknown"
)

type AcceptanceReceipt struct {
	Disposition     AcceptanceDisposition
	OwnerScope      string
	AcceptanceID    string
	BusinessEventID string
	FanoutID        string
	DeliveryIDs     []string
	AcceptedAt      time.Time
}

// AcceptAtomic commits one feature mutation and its complete webhook fan-out in
// one transaction. Accept is always the final SQL operation before commit.
func (s *Store) AcceptAtomic(ctx context.Context, prepared PreparedAcceptance, mutate func(context.Context, pgx.Tx) error) (AcceptanceReceipt, error) {
	if !s.valid() || mutate == nil {
		return AcceptanceReceipt{}, fmt.Errorf("%w: store and feature mutation are required", ErrConfig)
	}
	if err := validatePrepared(prepared); err != nil {
		return AcceptanceReceipt{}, err
	}
	var receipt AcceptanceReceipt
	err := s.pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := mutate(ctx, tx); err != nil {
			return err
		}
		var err error
		receipt, err = s.Accept(ctx, tx, prepared)
		return err
	})
	if errors.Is(err, postgres.ErrCommitUnknown) {
		receipt.Disposition = AcceptanceUnknown
	} else if err != nil && receipt.Disposition == AcceptanceAccepted {
		receipt.Disposition = AcceptanceRejected
	}
	if err != nil {
		return receipt, fmt.Errorf("accept webhook transaction: %w", err)
	}
	return receipt, nil
}

//nolint:gocognit,cyclop // Acceptance keeps its ordered same-transaction failure boundary explicit.
func (s *Store) Accept(ctx context.Context, tx pgx.Tx, prepared PreparedAcceptance) (AcceptanceReceipt, error) {
	if !s.valid() || tx == nil {
		return AcceptanceReceipt{}, fmt.Errorf("%w: store and caller transaction are required", ErrConfig)
	}
	if err := validatePrepared(prepared); err != nil {
		return AcceptanceReceipt{}, err
	}
	memberCount, err := int32Value(len(prepared.Destinations))
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	opCtx, cancel := context.WithTimeout(ctx, s.options.OperationTimeout)
	defer cancel()
	queries := sqlcgen.New(tx)
	acceptedAt, err := advanceClock(opCtx, queries)
	if err != nil {
		return AcceptanceReceipt{}, err
	}
	if err := lockAcceptance(opCtx, queries, prepared.Acceptance.OwnerScope, prepared.Acceptance.BusinessEventID); err != nil {
		return AcceptanceReceipt{}, err
	}
	if _, err := queries.ReadWebhookTombstone(opCtx, tombstoneLookup(prepared)); err == nil {
		return AcceptanceReceipt{Disposition: AcceptancePrivacyDeleted, OwnerScope: prepared.Acceptance.OwnerScope, AcceptanceID: prepared.Acceptance.AcceptanceID, BusinessEventID: prepared.Acceptance.BusinessEventID, FanoutID: prepared.Acceptance.FanoutSnapshotID}, ErrPrivacyDeleted
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AcceptanceReceipt{}, fmt.Errorf("read webhook tombstone: %w", err)
	}
	if receipt, resolveErr := resolveAcceptance(opCtx, queries, prepared); receipt.Disposition != AcceptanceRejected || resolveErr != nil {
		return receipt, resolveErr
	}
	capacity, err := queries.ReadWebhookCapacity(opCtx)
	if err != nil {
		return AcceptanceReceipt{}, fmt.Errorf("read webhook capacity: %w", err)
	}
	if capacity.RevisionCount != 1 || capacity.CapacityRevision != s.options.CapacityRevision || int(capacity.SlotCount) != s.options.GlobalConcurrency {
		return AcceptanceReceipt{Disposition: AcceptanceRejected}, fmt.Errorf("%w: capacity authority mismatch", ErrConfig)
	}
	for _, destination := range prepared.Destinations {
		if destination.Policy.GlobalConcurrency < s.options.GlobalConcurrency || !s.admits(destination.Policy) {
			return AcceptanceReceipt{Disposition: AcceptanceRejected}, fmt.Errorf("%w: worker bounds exceed destination policy", ErrConfig)
		}
		if err := insertAndMatchDestination(opCtx, queries, prepared.Acceptance.OwnerScope, destination, acceptedAt, s.options.ManifestRevision); err != nil {
			return AcceptanceReceipt{Disposition: AcceptanceConflict}, err
		}
	}
	input := prepared.Acceptance
	accepted := pgtime(acceptedAt)
	rows, err := queries.InsertWebhookEvent(opCtx, sqlcgen.InsertWebhookEventParams{
		OwnerScope: input.OwnerScope, BusinessEventID: input.BusinessEventID, AcceptanceID: input.AcceptanceID,
		FanoutSnapshotID: input.FanoutSnapshotID, EventType: input.EventType, BusinessSchemaVersion: input.BusinessSchemaVersion,
		ContentType: input.ContentType, Body: input.Body, DeliveryEnvelopeVersion: input.DeliveryEnvelopeVersion,
		SubscriberPolicyRevision: input.SubscriberPolicyRevision, IntentFingerprint: prepared.Fingerprint[:],
		RetentionPolicyIdentity: input.SubscriberPolicyRevision, AcceptedAt: accepted,
	})
	if err != nil {
		return AcceptanceReceipt{}, fmt.Errorf("insert webhook event: %w", err)
	}
	if rows != 1 {
		return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
	}
	rows, err = queries.InsertWebhookFanout(opCtx, sqlcgen.InsertWebhookFanoutParams{OwnerScope: input.OwnerScope, FanoutSnapshotID: input.FanoutSnapshotID, BusinessEventID: input.BusinessEventID, MemberCount: memberCount, MemberFingerprint: prepared.Fingerprint[:], AcceptedAt: accepted})
	if err != nil {
		return AcceptanceReceipt{}, fmt.Errorf("insert webhook fanout: %w", err)
	}
	if rows != 1 {
		return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
	}
	for _, destination := range prepared.Destinations {
		maximumAttempts, err := int32Value(destination.Policy.MaximumAttempts)
		if err != nil {
			return AcceptanceReceipt{}, err
		}
		policy, err := json.Marshal(destination.Policy)
		if err != nil {
			return AcceptanceReceipt{}, fmt.Errorf("encode webhook policy: %w", err)
		}
		deadline := acceptedAt.Add(destination.Policy.MaximumDeliveryAge)
		horizons := destination.Policy.Horizons
		redriveUntil := acceptedAt.Add(horizons.RedriveEligibility)
		rows, err := queries.InsertWebhookDelivery(opCtx, sqlcgen.InsertWebhookDeliveryParams{
			OwnerScope: input.OwnerScope, DeliveryID: destination.DeliveryID, BusinessEventID: input.BusinessEventID,
			FanoutSnapshotID: input.FanoutSnapshotID, DestinationID: destination.DestinationID,
			DestinationGeneration: destination.Generation, UrlSnapshot: destination.URL, PolicySnapshot: policy,
			NextDueAt: accepted, RedriveEligibleUntil: pgtime(redriveUntil),
			PayloadRetainedUntil: pgtime(acceptedAt.Add(horizons.Payload)), ActiveRetainedUntil: pgtime(acceptedAt.Add(horizons.Active)),
			TerminalSummaryRetainedUntil: pgtime(acceptedAt.Add(horizons.TerminalSummary)), AttemptRetainedUntil: pgtime(acceptedAt.Add(horizons.Attempt)),
			ActionRetainedUntil: pgtime(acceptedAt.Add(horizons.Action)), DestinationGenerationRetainedUntil: pgtime(acceptedAt.Add(horizons.DestinationGeneration)),
			ReceiverDedupRetainedUntil: pgtime(acceptedAt.Add(horizons.ReceiverDedup)), CreatedAt: accepted,
		})
		if err != nil {
			return AcceptanceReceipt{}, fmt.Errorf("insert webhook delivery: %w", err)
		}
		if rows != 1 {
			return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
		}
		rows, err = queries.InsertWebhookCycle(opCtx, sqlcgen.InsertWebhookCycleParams{OwnerScope: input.OwnerScope, DeliveryID: destination.DeliveryID, CycleNumber: 0, CycleKind: "automatic", AcceptedAt: accepted, DeadlineAt: pgtime(deadline), MaximumAttempts: maximumAttempts})
		if err != nil {
			return AcceptanceReceipt{}, fmt.Errorf("insert webhook cycle: %w", err)
		}
		if rows != 1 {
			return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
		}
	}
	receipt, err := resolveAcceptance(opCtx, queries, prepared)
	if err != nil {
		return receipt, err
	}
	return receipt, nil
}

func (s *Store) ResolveAcceptance(ctx context.Context, prepared PreparedAcceptance) (AcceptanceReceipt, error) {
	if !s.valid() {
		return AcceptanceReceipt{}, fmt.Errorf("%w: store is required", ErrConfig)
	}
	var receipt AcceptanceReceipt
	err := s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if err := lockAcceptance(ctx, queries, prepared.Acceptance.OwnerScope, prepared.Acceptance.BusinessEventID); err != nil {
			return err
		}
		var err error
		receipt, err = resolveAcceptance(ctx, queries, prepared)
		return err
	})
	if err != nil && receipt.Disposition == "" {
		receipt.Disposition = AcceptanceUnknown
	}
	return receipt, err
}

func resolveAcceptance(ctx context.Context, queries *sqlcgen.Queries, prepared PreparedAcceptance) (AcceptanceReceipt, error) {
	input := prepared.Acceptance
	if _, err := queries.ReadWebhookTombstone(ctx, tombstoneLookup(prepared)); err == nil {
		return AcceptanceReceipt{Disposition: AcceptancePrivacyDeleted, OwnerScope: input.OwnerScope, AcceptanceID: input.AcceptanceID, BusinessEventID: input.BusinessEventID, FanoutID: input.FanoutSnapshotID}, ErrPrivacyDeleted
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AcceptanceReceipt{}, fmt.Errorf("read webhook tombstone: %w", err)
	}
	row, err := queries.ReadWebhookAcceptance(ctx, sqlcgen.ReadWebhookAcceptanceParams{OwnerScope: input.OwnerScope, AcceptanceID: input.AcceptanceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return AcceptanceReceipt{Disposition: AcceptanceRejected, OwnerScope: input.OwnerScope, AcceptanceID: input.AcceptanceID, BusinessEventID: input.BusinessEventID, FanoutID: input.FanoutSnapshotID}, nil
	}
	if err != nil {
		return AcceptanceReceipt{Disposition: AcceptanceUnknown}, fmt.Errorf("read webhook acceptance: %w", err)
	}
	fingerprintMatches := bytes.Equal(row.IntentFingerprint, prepared.Fingerprint[:]) && bytes.Equal(row.MemberFingerprint, prepared.Fingerprint[:])
	if !fingerprintMatches {
		legacy, legacyErr := legacyAcceptanceFingerprint(prepared)
		if legacyErr != nil {
			return AcceptanceReceipt{}, legacyErr
		}
		fingerprintMatches = bytes.Equal(row.IntentFingerprint, legacy[:]) && bytes.Equal(row.MemberFingerprint, legacy[:])
	}
	if row.BusinessEventID != input.BusinessEventID || row.FanoutSnapshotID != input.FanoutSnapshotID || !fingerprintMatches || int(row.MemberCount) != len(prepared.Destinations) {
		return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
	}
	deliveries, err := queries.ListWebhookAcceptanceDeliveries(ctx, sqlcgen.ListWebhookAcceptanceDeliveriesParams{OwnerScope: input.OwnerScope, FanoutSnapshotID: input.FanoutSnapshotID})
	if err != nil {
		return AcceptanceReceipt{}, fmt.Errorf("list webhook acceptance deliveries: %w", err)
	}
	if len(deliveries) != len(prepared.Destinations) {
		return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
	}
	ids := make([]string, 0, len(deliveries))
	for i, delivery := range deliveries {
		destination := prepared.Destinations[i]
		if delivery.DestinationID != destination.DestinationID || delivery.DestinationGeneration != destination.Generation || delivery.CurrentCycle != 0 {
			return AcceptanceReceipt{Disposition: AcceptanceConflict}, ErrConflict
		}
		ids = append(ids, delivery.DeliveryID)
	}
	return AcceptanceReceipt{Disposition: AcceptanceAccepted, OwnerScope: input.OwnerScope, AcceptanceID: input.AcceptanceID, BusinessEventID: input.BusinessEventID, FanoutID: input.FanoutSnapshotID, DeliveryIDs: ids, AcceptedAt: row.AcceptedAt.Time.UTC()}, nil
}

func tombstoneLookup(prepared PreparedAcceptance) sqlcgen.ReadWebhookTombstoneParams {
	input := prepared.Acceptance
	deliveryIDs := lo.Map(prepared.Destinations, func(destination PreparedDestination, _ int) string { return destination.DeliveryID })
	return sqlcgen.ReadWebhookTombstoneParams{
		OwnerScope: input.OwnerScope, BusinessEventID: input.BusinessEventID,
		AcceptanceID: &input.AcceptanceID, FanoutSnapshotID: &input.FanoutSnapshotID,
		DeliveryIds: deliveryIDs,
	}
}

func insertAndMatchDestination(ctx context.Context, queries *sqlcgen.Queries, owner string, destination PreparedDestination, acceptedAt time.Time, manifestRevision int64) error {
	if _, err := queries.ReadWebhookDestinationTombstone(ctx, sqlcgen.ReadWebhookDestinationTombstoneParams{OwnerScope: owner, DestinationID: destination.DestinationID, Generation: destination.Generation}); err == nil {
		return ErrConflict
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read webhook destination tombstone: %w", err)
	}
	policy, err := json.Marshal(destination.Policy)
	if err != nil {
		return fmt.Errorf("encode webhook destination policy: %w", err)
	}
	encodedPolicy, err := encodeDeliveryPolicy(destination.Policy)
	if err != nil {
		return err
	}
	digest := canonicalDigest(encodedPolicy)
	destinationConcurrency, err := int32Value(destination.Policy.DestinationConcurrency)
	if err != nil {
		return err
	}
	globalConcurrency, err := int32Value(destination.Policy.GlobalConcurrency)
	if err != nil {
		return err
	}
	params := sqlcgen.InsertWebhookDestinationParams{
		OwnerScope: owner, DestinationID: destination.DestinationID, Generation: destination.Generation,
		OwnershipVerificationReceipt: destination.OwnershipVerificationReceipt, Url: destination.URL,
		SelectionRevision: destination.SelectionRevision, PayloadVersionPreference: destination.PayloadVersionPreference,
		SignatureProfile: destination.SignatureProfile, SigningAuthorityBinding: destination.SigningAuthorityBinding,
		Policy: policy, PolicyFingerprint: digest[:], DestinationConcurrency: destinationConcurrency,
		GlobalConcurrency: globalConcurrency, RequiredSecretRevision: manifestRevision,
		ActiveKeyReference: destination.SigningAuthorityBinding, CreatedAt: pgtime(acceptedAt),
	}
	if _, err := queries.InsertWebhookDestination(ctx, params); err != nil {
		return fmt.Errorf("insert webhook destination: %w", err)
	}
	row, err := queries.ReadWebhookDestination(ctx, sqlcgen.ReadWebhookDestinationParams{OwnerScope: owner, DestinationID: destination.DestinationID, Generation: destination.Generation})
	if err != nil {
		return fmt.Errorf("read webhook destination: %w", err)
	}
	if row.Disposition != activeDisposition || row.OwnershipVerificationReceipt != params.OwnershipVerificationReceipt || row.Url != params.Url || row.SelectionRevision != params.SelectionRevision || row.PayloadVersionPreference != params.PayloadVersionPreference || row.SignatureProfile != params.SignatureProfile || row.SigningAuthorityBinding != params.SigningAuthorityBinding || !bytes.Equal(row.PolicyFingerprint, digest[:]) || row.DestinationConcurrency != params.DestinationConcurrency || row.GlobalConcurrency != params.GlobalConcurrency {
		return ErrConflict
	}
	return nil
}

func validatePrepared(prepared PreparedAcceptance) error {
	clone := prepared.Acceptance
	clone.Destinations = make([]DestinationSnapshot, len(prepared.Destinations))
	encodedDestinations := make([][]byte, len(prepared.Destinations))
	deliveryIDs := make(map[string]struct{}, len(prepared.Destinations))
	for i := range prepared.Destinations {
		clone.Destinations[i] = prepared.Destinations[i].DestinationSnapshot
		encoded, err := encodeDestinationIntent(clone.Destinations[i])
		if err != nil {
			return err
		}
		encodedDestinations[i] = encoded
		expectedID, err := deriveDeliveryID(prepared.Fingerprint, clone.Destinations[i])
		if err != nil || prepared.Destinations[i].DeliveryID != expectedID {
			return fmt.Errorf("%w: prepared delivery identity is invalid", ErrConfig)
		}
		if _, exists := deliveryIDs[prepared.Destinations[i].DeliveryID]; exists {
			return fmt.Errorf("%w: duplicate prepared delivery identity", ErrConfig)
		}
		deliveryIDs[prepared.Destinations[i].DeliveryID] = struct{}{}
	}
	if err := validateAcceptance(clone); err != nil {
		return err
	}
	destinationList, err := canonicalList(encodedDestinations)
	if err != nil {
		return fmt.Errorf("%w: prepared destination set: %w", ErrConfig, err)
	}
	input := prepared.Acceptance
	canonical, err := canonicalRecord("webhook-acceptance-intent-v2",
		[]byte(input.OwnerScope), []byte(input.AcceptanceID), []byte(input.BusinessEventID), []byte(input.FanoutSnapshotID),
		[]byte(input.EventType), []byte(input.BusinessSchemaVersion), []byte(input.ContentType), input.Body,
		[]byte(input.DeliveryEnvelopeVersion), []byte(input.SubscriberPolicyRevision), destinationList,
	)
	if err != nil {
		return err
	}
	if len(prepared.Destinations) == 0 || !bytes.Equal(canonical, prepared.CanonicalBytes) || canonicalDigest(canonical) != prepared.Fingerprint {
		return fmt.Errorf("%w: prepared acceptance is incomplete", ErrConfig)
	}
	return nil
}

func lockAcceptance(ctx context.Context, queries *sqlcgen.Queries, owner, event string) error {
	namespaceKey, err := advisoryKey(owner, targetKindNamespace, owner)
	if err != nil {
		return err
	}
	eventKey, err := advisoryKey(owner, targetKindEvent, event)
	if err != nil {
		return err
	}
	for _, key := range []int64{namespaceKey, eventKey} {
		if err := queries.LockWebhookAdvisoryKey(ctx, key); err != nil {
			return fmt.Errorf("lock webhook acceptance: %w", err)
		}
	}
	return nil
}

func advisoryKey(owner, kind, id string) (int64, error) {
	canonical, err := canonicalRecord("webhook-lock-v1", []byte(owner), []byte(kind), []byte(id))
	if err != nil {
		return 0, err
	}
	digest := sha256.Sum256(canonical)
	return int64(binary.BigEndian.Uint64(digest[:8]) & math.MaxInt64), nil
}

func pgtime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
