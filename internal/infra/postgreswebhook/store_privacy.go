package postgreswebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

func (s *Store) RequestEventPrivacyDeletion(ctx context.Context, request ActionRequest) (ActionReceipt, error) {
	payload, ok := request.Payload.(*PrivacyDeletionAction)
	if request.Kind != ActionPrivacyDelete || request.TargetKind != targetKindEvent || !ok || payload == nil || payload.TargetKind != targetKindEvent || payload.TargetID != request.TargetID || payload.Mode != "minimal_tombstone" {
		return ActionReceipt{}, fmt.Errorf("%w: event privacy request is invalid", ErrConfig)
	}
	fingerprint, err := request.Fingerprint()
	if err != nil {
		return ActionReceipt{}, err
	}
	var receipt ActionReceipt
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		now, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		if err := lockAcceptance(ctx, queries, request.OwnerScope, request.TargetID); err != nil {
			return err
		}
		if replay, handled, err := tombstoneReplay(ctx, queries, request, fingerprint); handled || err != nil {
			receipt = replay
			return err
		}
		receipt, err = deleteWebhookEventForPrivacy(ctx, queries, request, *payload, fingerprint, now)
		return err
	})
	return receipt, err
}

func deleteWebhookEventForPrivacy(ctx context.Context, queries *sqlcgen.Queries, request ActionRequest, payload PrivacyDeletionAction, fingerprint [32]byte, now time.Time) (ActionReceipt, error) {
	event, err := queries.ReadWebhookEventForPrivacy(ctx, sqlcgen.ReadWebhookEventForPrivacyParams{OwnerScope: request.OwnerScope, BusinessEventID: request.TargetID})
	if errors.Is(err, pgx.ErrNoRows) {
		rows, insertErr := queries.InsertWebhookTombstone(ctx, sqlcgen.InsertWebhookTombstoneParams{
			OwnerScope: request.OwnerScope, TargetKind: targetKindEvent, TargetID: request.TargetID,
			DeliveryIdentities: []byte("[]"), DestinationIdentities: []byte("[]"),
			LastSemanticClass: "privacy_deleted", ActionID: request.ActionID,
			RequestFingerprint: fingerprint[:], FirstDisposition: actionResultNotFound,
			DeletionAuthority: payload.DeletionAuthority, CreatedAt: pgtime(now),
		})
		if insertErr != nil {
			return ActionReceipt{}, fmt.Errorf("insert absent webhook event tombstone: %w", insertErr)
		}
		if rows != 1 {
			return ActionReceipt{}, ErrConflict
		}
		return ActionReceipt{ActionID: request.ActionID, Result: actionResultNotFound}, nil
	}
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("read webhook event for privacy: %w", err)
	}
	deliveries, err := json.Marshal(event.DeliveryIdentities)
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("encode webhook deletion identities: %w", err)
	}
	destinations, err := json.Marshal(event.DestinationIdentities)
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("encode webhook destination identities: %w", err)
	}
	rows, err := queries.InsertWebhookTombstone(ctx, sqlcgen.InsertWebhookTombstoneParams{
		OwnerScope: request.OwnerScope, TargetKind: targetKindEvent, TargetID: request.TargetID,
		AcceptanceID: &event.AcceptanceID, FanoutSnapshotID: &event.FanoutSnapshotID,
		DeliveryIdentities: deliveries, DestinationIdentities: destinations,
		LastSemanticClass: event.LastSemanticClass, ActionID: request.ActionID,
		RequestFingerprint: fingerprint[:], FirstDisposition: "applied",
		DeletionAuthority: payload.DeletionAuthority, CreatedAt: pgtime(now),
	})
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("insert webhook event tombstone: %w", err)
	}
	if rows != 1 {
		return ActionReceipt{}, ErrConflict
	}
	rows, err = queries.DeleteWebhookEvent(ctx, sqlcgen.DeleteWebhookEventParams{OwnerScope: request.OwnerScope, BusinessEventID: request.TargetID})
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("delete webhook event: %w", err)
	}
	if rows != 1 {
		return ActionReceipt{}, ErrConflict
	}
	return ActionReceipt{ActionID: request.ActionID, Result: "applied"}, nil
}

//nolint:cyclop // Namespace retirement is one bounded resumable transaction with explicit inventory checks.
func (s *Store) RequestNamespaceRetirement(ctx context.Context, request ActionRequest, batchSize int) (ActionReceipt, error) {
	payload, ok := request.Payload.(*NamespaceRetirementAction)
	if request.Kind != ActionNamespaceRetire || request.TargetKind != targetKindNamespace || request.TargetID != request.OwnerScope || !ok || payload == nil || payload.Mode != "full_erasure" || batchSize < 1 || batchSize > 1000 {
		return ActionReceipt{}, fmt.Errorf("%w: namespace retirement request is invalid", ErrConfig)
	}
	fingerprint, err := request.Fingerprint()
	if err != nil {
		return ActionReceipt{}, err
	}
	batch, err := int32Value(batchSize)
	if err != nil {
		return ActionReceipt{}, err
	}
	var receipt ActionReceipt
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		now, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		namespaceKey, err := advisoryKey(request.OwnerScope, targetKindNamespace, request.OwnerScope)
		if err != nil {
			return err
		}
		if err := queries.LockWebhookAdvisoryKey(ctx, namespaceKey); err != nil {
			return fmt.Errorf("lock webhook namespace: %w", err)
		}
		if replay, handled, err := tombstoneReplay(ctx, queries, request, fingerprint); err != nil {
			return err
		} else if handled && replay.Result == "completed" {
			receipt = replay
			return nil
		}
		if _, err := queries.ReadWebhookTombstoneAction(ctx, sqlcgen.ReadWebhookTombstoneActionParams{OwnerScope: request.OwnerScope, ActionID: request.ActionID}); errors.Is(err, pgx.ErrNoRows) {
			rows, insertErr := queries.InsertWebhookTombstone(ctx, sqlcgen.InsertWebhookTombstoneParams{
				OwnerScope: request.OwnerScope, TargetKind: targetKindNamespace, TargetID: request.OwnerScope,
				DeliveryIdentities: []byte("[]"), DestinationIdentities: []byte("[]"), LastSemanticClass: "none",
				ActionID: request.ActionID, RequestFingerprint: fingerprint[:], FirstDisposition: "pending",
				DeletionAuthority: payload.DeletionAuthority, CreatedAt: pgtime(now),
			})
			if insertErr != nil || rows != 1 {
				if insertErr != nil {
					return fmt.Errorf("insert webhook namespace tombstone: %w", insertErr)
				}
				return ErrConflict
			}
		} else if err != nil {
			return fmt.Errorf("read webhook namespace tombstone: %w", err)
		}
		receipt, err = retireNamespaceBatch(ctx, queries, request.OwnerScope, request.ActionID, batch)
		return err
	})
	return receipt, err
}

func (s *Store) ResumeNamespaceRetirements(ctx context.Context, batchSize int) (int, error) {
	if !s.valid() || batchSize < 1 || batchSize > 1000 {
		return 0, fmt.Errorf("%w: namespace retirement batch is invalid", ErrConfig)
	}
	batch, err := int32Value(batchSize)
	if err != nil {
		return 0, err
	}
	progressed := 0
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if _, err := advanceClock(ctx, queries); err != nil {
			return err
		}
		pending, err := queries.LockPendingWebhookNamespaceRetirement(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("lock pending webhook namespace retirement: %w", err)
		}
		namespaceKey, err := advisoryKey(pending.OwnerScope, targetKindNamespace, pending.OwnerScope)
		if err != nil {
			return err
		}
		if err := queries.LockWebhookAdvisoryKey(ctx, namespaceKey); err != nil {
			return fmt.Errorf("lock webhook namespace: %w", err)
		}
		before, err := queries.CountWebhookNamespaceRows(ctx, pending.OwnerScope)
		if err != nil {
			return fmt.Errorf("inventory webhook namespace before resume: %w", err)
		}
		receipt, err := retireNamespaceBatch(ctx, queries, pending.OwnerScope, pending.ActionID, batch)
		if err != nil {
			return err
		}
		after, err := queries.CountWebhookNamespaceRows(ctx, pending.OwnerScope)
		if err != nil {
			return fmt.Errorf("inventory webhook namespace after resume: %w", err)
		}
		if after < before || receipt.Result == "completed" {
			progressed = 1
		}
		return nil
	})
	return progressed, err
}

func retireNamespaceBatch(ctx context.Context, queries *sqlcgen.Queries, owner, actionID string, batch int32) (ActionReceipt, error) {
	deleted, err := queries.DeleteWebhookNamespaceBatch(ctx, sqlcgen.DeleteWebhookNamespaceBatchParams{OwnerScope: owner, BatchSize: batch})
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("delete webhook namespace batch: %w", err)
	}
	if deleted.PossibleSend {
		if err := queries.MarkWebhookNamespacePossibleSend(ctx, sqlcgen.MarkWebhookNamespacePossibleSendParams{OwnerScope: owner, ActionID: actionID}); err != nil {
			return ActionReceipt{}, fmt.Errorf("preserve webhook namespace ambiguity: %w", err)
		}
	}
	remaining, err := remainingBatch(batch, deleted.Deleted)
	if err != nil {
		return ActionReceipt{}, err
	}
	if remaining > 0 {
		count, err := queries.DeleteWebhookNamespaceActions(ctx, sqlcgen.DeleteWebhookNamespaceActionsParams{OwnerScope: owner, BatchSize: remaining})
		if err != nil {
			return ActionReceipt{}, fmt.Errorf("delete webhook namespace actions: %w", err)
		}
		remaining, err = remainingBatch(remaining, count)
		if err != nil {
			return ActionReceipt{}, err
		}
	}
	if remaining > 0 {
		count, err := queries.DeleteWebhookNamespaceDestinations(ctx, sqlcgen.DeleteWebhookNamespaceDestinationsParams{OwnerScope: owner, BatchSize: remaining})
		if err != nil {
			return ActionReceipt{}, fmt.Errorf("delete webhook namespace destinations: %w", err)
		}
		remaining, err = remainingBatch(remaining, count)
		if err != nil {
			return ActionReceipt{}, err
		}
	}
	if remaining > 0 {
		if _, err := queries.DeleteWebhookNamespaceDestinationTombstones(ctx, sqlcgen.DeleteWebhookNamespaceDestinationTombstonesParams{OwnerScope: owner, BatchSize: remaining}); err != nil {
			return ActionReceipt{}, fmt.Errorf("delete webhook namespace destination tombstones: %w", err)
		}
	}
	inventory, err := queries.CountWebhookNamespaceRows(ctx, owner)
	if err != nil {
		return ActionReceipt{}, fmt.Errorf("inventory webhook namespace: %w", err)
	}
	result := "pending"
	if inventory == 0 {
		rows, err := queries.CompleteWebhookNamespaceTombstone(ctx, sqlcgen.CompleteWebhookNamespaceTombstoneParams{OwnerScope: owner, ActionID: actionID})
		if err != nil || rows != 1 {
			if err != nil {
				return ActionReceipt{}, fmt.Errorf("complete webhook namespace tombstone: %w", err)
			}
			return ActionReceipt{}, ErrConflict
		}
		result = "completed"
	}
	return ActionReceipt{ActionID: actionID, Result: result}, nil
}

func tombstoneReplay(ctx context.Context, queries *sqlcgen.Queries, request ActionRequest, fingerprint [32]byte) (ActionReceipt, bool, error) {
	row, err := queries.ReadWebhookTombstoneAction(ctx, sqlcgen.ReadWebhookTombstoneActionParams{OwnerScope: request.OwnerScope, ActionID: request.ActionID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ActionReceipt{}, false, nil
	}
	if err != nil {
		return ActionReceipt{}, false, fmt.Errorf("read webhook tombstone action: %w", err)
	}
	if !bytes.Equal(row.RequestFingerprint, fingerprint[:]) {
		return ActionReceipt{}, true, ErrConflict
	}
	return ActionReceipt{ActionID: request.ActionID, Result: row.FirstDisposition, Replay: true}, true, nil
}
