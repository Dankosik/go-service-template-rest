package postgreswebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

//nolint:gocognit,cyclop // Privacy deletion keeps tombstone, evidence, and erasure ordering explicit.
func (s *Store) RequestEventPrivacyDeletion(ctx context.Context, request ActionRequest) (ActionReceipt, error) {
	if request.Kind != ActionPrivacyDelete || request.TargetKind != "event" || request.Note != "" || len(request.Values) != 4 || request.Values[0] != "event" || request.Values[1] != request.TargetID || request.Values[2] != "minimal_tombstone" {
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
		event, err := queries.ReadWebhookEventForPrivacy(ctx, sqlcgen.ReadWebhookEventForPrivacyParams{OwnerScope: request.OwnerScope, BusinessEventID: request.TargetID})
		if errors.Is(err, pgx.ErrNoRows) {
			receipt = ActionReceipt{ActionID: request.ActionID, Result: "not_found"}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read webhook event for privacy: %w", err)
		}
		if _, err := queries.LockWebhookEventDeliveriesForPrivacy(ctx, sqlcgen.LockWebhookEventDeliveriesForPrivacyParams{OwnerScope: request.OwnerScope, BusinessEventID: request.TargetID}); err != nil {
			return fmt.Errorf("lock webhook deliveries for privacy: %w", err)
		}
		attempts, err := queries.LockWebhookEventAttemptsForPrivacy(ctx, sqlcgen.LockWebhookEventAttemptsForPrivacyParams{OwnerScope: request.OwnerScope, BusinessEventID: request.TargetID})
		if err != nil {
			return fmt.Errorf("lock webhook attempts for privacy: %w", err)
		}
		for _, attempt := range attempts {
			if attempt.MayHaveSent || attempt.SendAuthorized {
				event.LastSemanticClass = string(OutcomeUnknown)
				break
			}
		}
		deliveries, err := json.Marshal(event.DeliveryIdentities)
		if err != nil {
			return fmt.Errorf("encode webhook deletion identities: %w", err)
		}
		destinations, err := json.Marshal(event.DestinationIdentities)
		if err != nil {
			return fmt.Errorf("encode webhook destination identities: %w", err)
		}
		ownerScope := request.OwnerScope
		if err := queries.ReleaseWebhookEventCapacity(ctx, sqlcgen.ReleaseWebhookEventCapacityParams{OwnerScope: &ownerScope, BusinessEventID: request.TargetID}); err != nil {
			return fmt.Errorf("release webhook event capacity: %w", err)
		}
		rows, err := queries.InsertWebhookTombstone(ctx, sqlcgen.InsertWebhookTombstoneParams{
			OwnerScope: request.OwnerScope, TargetKind: "event", TargetID: request.TargetID,
			AcceptanceID: &event.AcceptanceID, FanoutSnapshotID: &event.FanoutSnapshotID,
			DeliveryIdentities: deliveries, DestinationIdentities: destinations,
			LastSemanticClass: event.LastSemanticClass, ActionID: request.ActionID,
			RequestFingerprint: fingerprint[:], FirstDisposition: "applied",
			DeletionAuthority: request.Values[3], CreatedAt: pgtime(now),
		})
		if err != nil || rows != 1 {
			if err != nil {
				return fmt.Errorf("insert webhook event tombstone: %w", err)
			}
			return ErrConflict
		}
		rows, err = queries.DeleteWebhookEvent(ctx, sqlcgen.DeleteWebhookEventParams{OwnerScope: request.OwnerScope, BusinessEventID: request.TargetID})
		if err != nil || rows != 1 {
			if err != nil {
				return fmt.Errorf("delete webhook event: %w", err)
			}
			return ErrConflict
		}
		receipt = ActionReceipt{ActionID: request.ActionID, Result: "applied"}
		return nil
	})
	return receipt, err
}

//nolint:gocognit,cyclop // Namespace retirement is one bounded resumable transaction with explicit inventory checks.
func (s *Store) RequestNamespaceRetirement(ctx context.Context, request ActionRequest, batchSize int) (ActionReceipt, error) {
	if request.Kind != ActionNamespaceRetire || request.TargetKind != "namespace" || request.TargetID != request.OwnerScope || request.Note != "" || len(request.Values) != 2 || request.Values[0] != "full_erasure" || batchSize < 1 || batchSize > 1000 {
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
		namespaceKey, err := advisoryKey(request.OwnerScope, "namespace", request.OwnerScope)
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
				OwnerScope: request.OwnerScope, TargetKind: "namespace", TargetID: request.OwnerScope,
				DeliveryIdentities: []byte("[]"), DestinationIdentities: []byte("[]"), LastSemanticClass: "none",
				ActionID: request.ActionID, RequestFingerprint: fingerprint[:], FirstDisposition: "pending",
				DeletionAuthority: request.Values[1], CreatedAt: pgtime(now),
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
		if _, err := queries.LockWebhookNamespaceDestinationsForPrivacy(ctx, request.OwnerScope); err != nil {
			return fmt.Errorf("lock webhook namespace destinations: %w", err)
		}
		if _, err := queries.LockWebhookNamespaceDeliveriesForPrivacy(ctx, request.OwnerScope); err != nil {
			return fmt.Errorf("lock webhook namespace deliveries: %w", err)
		}
		attempts, err := queries.LockWebhookNamespaceAttemptsForPrivacy(ctx, request.OwnerScope)
		if err != nil {
			return fmt.Errorf("lock webhook namespace attempts: %w", err)
		}
		for _, attempt := range attempts {
			if attempt.MayHaveSent || attempt.SendAuthorized {
				if _, err := queries.MarkWebhookNamespaceTombstoneUnknown(ctx, sqlcgen.MarkWebhookNamespaceTombstoneUnknownParams{OwnerScope: request.OwnerScope, ActionID: request.ActionID}); err != nil {
					return fmt.Errorf("mark webhook namespace ambiguity: %w", err)
				}
				break
			}
		}
		ownerScope := request.OwnerScope
		if err := queries.ReleaseWebhookNamespaceCapacity(ctx, &ownerScope); err != nil {
			return fmt.Errorf("release webhook namespace capacity: %w", err)
		}
		if _, err := queries.DeleteWebhookNamespaceBatch(ctx, sqlcgen.DeleteWebhookNamespaceBatchParams{OwnerScope: request.OwnerScope, BatchSize: batch}); err != nil {
			return fmt.Errorf("delete webhook namespace batch: %w", err)
		}
		if err := queries.DeleteWebhookNamespaceActions(ctx, request.OwnerScope); err != nil {
			return fmt.Errorf("delete webhook namespace actions: %w", err)
		}
		if err := queries.DeleteWebhookNamespaceDestinations(ctx, request.OwnerScope); err != nil {
			return fmt.Errorf("delete webhook namespace destinations: %w", err)
		}
		remaining, err := queries.CountWebhookNamespaceRows(ctx, request.OwnerScope)
		if err != nil {
			return fmt.Errorf("inventory webhook namespace: %w", err)
		}
		result := "pending"
		if remaining == 0 {
			rows, err := queries.CompleteWebhookNamespaceTombstone(ctx, sqlcgen.CompleteWebhookNamespaceTombstoneParams{OwnerScope: request.OwnerScope, ActionID: request.ActionID})
			if err != nil || rows != 1 {
				if err != nil {
					return fmt.Errorf("complete webhook namespace tombstone: %w", err)
				}
				return ErrConflict
			}
			result = "completed"
		}
		receipt = ActionReceipt{ActionID: request.ActionID, Result: result}
		return nil
	})
	return receipt, err
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
