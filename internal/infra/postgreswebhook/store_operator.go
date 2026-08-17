package postgreswebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type ActionReceipt struct {
	ActionID string
	Result   string
	Replay   bool
	Cycle    int64
}

const (
	actionResultNotFound      = "not_found"
	actionResultStateConflict = "state_conflict"
)

func (s *Store) ApplyAction(ctx context.Context, request ActionRequest, manifest *SecretManifest) (ActionReceipt, error) {
	fingerprint, err := request.Fingerprint()
	if err != nil {
		return ActionReceipt{}, err
	}
	if !s.valid() || manifest == nil || manifest.Revision() != s.options.ManifestRevision {
		return ActionReceipt{}, fmt.Errorf("%w: current secret manifest is required", ErrConfig)
	}
	if request.Kind == ActionPrivacyDelete || request.Kind == ActionNamespaceRetire {
		return ActionReceipt{}, fmt.Errorf("%w: privacy actions use their dedicated store owner", ErrConfig)
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
		if row, err := queries.ReadWebhookOperatorAction(ctx, sqlcgen.ReadWebhookOperatorActionParams{OwnerScope: request.OwnerScope, ActionID: request.ActionID}); err == nil {
			if !bytes.Equal(row.RequestFingerprint, fingerprint[:]) {
				return ErrConflict
			}
			receipt = ActionReceipt{ActionID: request.ActionID, Result: row.Result, Replay: true, Cycle: row.ResolvedResultCycle}
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read webhook action: %w", err)
		}
		result, cycle, err := s.applyActionMutation(ctx, queries, request, now, manifest)
		if err != nil {
			return err
		}
		retainUntil := now.Add(MaxRetentionTime)
		if result != actionResultNotFound {
			retainUntil, err = actionRetainUntil(ctx, queries, request, now)
			if err != nil && !errors.Is(err, ErrConflict) {
				return err
			}
			if errors.Is(err, ErrConflict) {
				retainUntil = now.Add(MaxRetentionTime)
			}
		}
		payload, err := json.Marshal(request.Payload)
		if err != nil {
			return fmt.Errorf("encode webhook action payload: %w", err)
		}
		if _, err := queries.InsertWebhookOperatorAction(ctx, sqlcgen.InsertWebhookOperatorActionParams{
			OwnerScope: request.OwnerScope, ActionID: request.ActionID, RequestFingerprint: fingerprint[:],
			ActorReference: request.Actor, ActionKind: string(request.Kind), TargetKind: request.TargetKind,
			TargetID: request.TargetID, TargetGeneration: request.TargetGeneration, ExpectedState: strconv.FormatInt(request.ExpectedRevision, 10),
			Reason: request.Reason, DuplicateRiskAcknowledged: actionDuplicateRisk(request), Result: result,
			CreatedAt: pgtime(now), RetainUntil: pgtime(retainUntil), RequestPayload: payload, ResultCycle: cycle,
		}); err != nil {
			return fmt.Errorf("insert webhook action: %w", err)
		}
		receipt = ActionReceipt{ActionID: request.ActionID, Result: result, Cycle: cycle}
		return nil
	})
	return receipt, err
}

//nolint:gocognit,cyclop // Each closed operator action keeps its validation beside its SQL transition.
func (s *Store) applyActionMutation(ctx context.Context, queries *sqlcgen.Queries, request ActionRequest, now time.Time, manifest *SecretManifest) (string, int64, error) {
	expected := request.ExpectedRevision
	switch request.Kind {
	case ActionDestinationState:
		payload, ok := request.Payload.(*DestinationStateAction)
		if !ok || payload == nil {
			return "", 0, fmt.Errorf("%w: destination state payload is invalid", ErrConfig)
		}
		disposition := map[string]string{activeDisposition: activeDisposition, "paused": "automatically_paused", "disabled": "administratively_disabled", "retired": "retired"}[payload.Disposition]
		if disposition == "" {
			return "", 0, fmt.Errorf("%w: destination disposition is invalid", ErrConfig)
		}
		rows, err := queries.ApplyWebhookDestinationState(ctx, sqlcgen.ApplyWebhookDestinationStateParams{Disposition: disposition, UpdatedAt: pgtime(now), OwnerScope: request.OwnerScope, DestinationID: request.TargetID, Generation: request.TargetGeneration, ExpectedRevision: expected})
		return actionMutationResult(rows, err)
	case ActionKeyRotation:
		payload, ok := request.Payload.(*KeyRotationAction)
		if !ok || payload == nil || payload.SecretRevision <= 0 || payload.SecretRevision > manifest.Revision() || payload.KeyRevision <= 0 || payload.ActiveKeyReference == "" || payload.PredecessorReference == "" || payload.ActiveKeyReference == payload.PredecessorReference || payload.OverlapStartsAt.After(now) || !payload.PredecessorValidUntil.After(now) || payload.AuthorityReceipt == "" {
			return "", 0, fmt.Errorf("%w: key rotation is invalid", ErrConfig)
		}
		if _, err := manifest.Resolve(request.OwnerScope, request.TargetID, payload.ActiveKeyReference); err != nil {
			return "", 0, fmt.Errorf("%w: active rotation secret is missing", ErrConfig)
		}
		if _, err := manifest.Resolve(request.OwnerScope, request.TargetID, payload.PredecessorReference); err != nil {
			return "", 0, fmt.Errorf("%w: predecessor rotation secret is missing", ErrConfig)
		}
		predecessor := payload.PredecessorReference
		rows, err := queries.ApplyWebhookKeyRotation(ctx, sqlcgen.ApplyWebhookKeyRotationParams{RequiredSecretRevision: payload.SecretRevision, KeyStateRevision: payload.KeyRevision, ActiveKeyReference: payload.ActiveKeyReference, PredecessorKeyReference: &predecessor, PredecessorValidUntil: pgtime(payload.PredecessorValidUntil), UpdatedAt: pgtime(now), OwnerScope: request.OwnerScope, DestinationID: request.TargetID, Generation: request.TargetGeneration, ExpectedRevision: expected})
		return actionMutationResult(rows, err)
	case ActionRedrive:
		payload, ok := request.Payload.(*RedriveAction)
		if !ok || payload == nil {
			return "", 0, fmt.Errorf("%w: redrive payload is invalid", ErrConfig)
		}
		if !payload.AcknowledgeDuplicateRisk {
			return "rejected", 0, nil
		}
		if payload.MaximumAttempts < 1 || payload.MaximumAttempts > MaxAttempts || payload.MaximumAge <= 0 {
			return "", 0, fmt.Errorf("%w: redrive bounds are invalid", ErrConfig)
		}
		maximumAttempts, err := int32Value(payload.MaximumAttempts)
		if err != nil {
			return "", 0, err
		}
		locked, err := queries.LockWebhookDeliveryForAction(ctx, sqlcgen.LockWebhookDeliveryForActionParams{OwnerScope: request.OwnerScope, DeliveryID: request.TargetID})
		if errors.Is(err, pgx.ErrNoRows) {
			return actionResultNotFound, 0, nil
		}
		if err != nil {
			return "", 0, fmt.Errorf("lock webhook redrive: %w", err)
		}
		var policy DeliveryPolicy
		if err := json.Unmarshal(locked.PolicySnapshot, &policy); err != nil || policy.validate() != nil || !s.admits(policy) {
			return "", 0, fmt.Errorf("%w: decode retained webhook redrive policy", ErrConflict)
		}
		if locked.CurrentCycle != expected || locked.State != string(DeliveryTerminal) && locked.State != string(DeliverySuspended) || locked.Disposition == activeDisposition || locked.DestinationDisposition != activeDisposition || locked.RequiredSecretRevision > manifest.Revision() || !locked.PayloadPresent || locked.CumulativeSummary == string(OutcomeHTTPAccepted) || payload.MaximumAttempts > policy.RedriveAttempts || payload.MaximumAge > policy.RedriveAge {
			return actionResultStateConflict, 0, nil
		}
		if !manifestContains(manifest, request.OwnerScope, locked.DestinationID, locked.ActiveKeyReference) {
			return actionResultStateConflict, 0, nil
		}
		if locked.PredecessorKeyReference != nil && locked.PredecessorValidUntil.Valid && now.Before(locked.PredecessorValidUntil.Time) {
			if !manifestContains(manifest, request.OwnerScope, locked.DestinationID, *locked.PredecessorKeyReference) {
				return actionResultStateConflict, 0, nil
			}
		}
		retainedUntil := locked.RedriveEligibleUntil.Time
		for _, limit := range []time.Time{locked.PayloadRetainedUntil.Time, locked.ActiveRetainedUntil.Time, locked.TerminalSummaryRetainedUntil.Time, locked.AttemptRetainedUntil.Time, locked.ActionRetainedUntil.Time, locked.DestinationGenerationRetainedUntil.Time, locked.ReceiverDedupRetainedUntil.Time} {
			if limit.Before(retainedUntil) {
				retainedUntil = limit
			}
		}
		if !retainedUntil.After(now) {
			return actionResultStateConflict, 0, nil
		}
		deadline := now.Add(payload.MaximumAge)
		if retainedUntil.Before(deadline) {
			deadline = retainedUntil
		}
		cycle := locked.CurrentCycle + 1
		rows, err := queries.InsertWebhookRedriveCycle(ctx, sqlcgen.InsertWebhookRedriveCycleParams{OwnerScope: request.OwnerScope, DeliveryID: request.TargetID, CycleNumber: cycle, ActionID: &request.ActionID, AcceptedAt: pgtime(now), DeadlineAt: pgtime(deadline), MaximumAttempts: maximumAttempts})
		if err != nil || rows != 1 {
			return actionMutationResult(rows, err)
		}
		rows, err = queries.ActivateWebhookRedriveCycle(ctx, sqlcgen.ActivateWebhookRedriveCycleParams{CycleNumber: cycle, AcceptedAt: pgtime(now), OwnerScope: request.OwnerScope, DeliveryID: request.TargetID, PreviousCycle: locked.CurrentCycle})
		result, _, err := actionMutationResult(rows, err)
		return result, cycle, err
	case ActionCloseUnknown:
		payload, ok := request.Payload.(*CloseUnknownAction)
		if !ok || payload == nil || payload.Disposition != string(OutcomeClosedUnknown) || !payload.AcknowledgeDuplicateRisk {
			return "", 0, fmt.Errorf("%w: close-unknown payload is invalid", ErrConfig)
		}
		locked, err := queries.LockWebhookDeliveryForAction(ctx, sqlcgen.LockWebhookDeliveryForActionParams{OwnerScope: request.OwnerScope, DeliveryID: request.TargetID})
		if errors.Is(err, pgx.ErrNoRows) {
			return actionResultNotFound, 0, nil
		}
		if err != nil {
			return "", 0, fmt.Errorf("lock webhook unknown delivery: %w", err)
		}
		if locked.CurrentCycle != expected || locked.CumulativeSummary != string(OutcomeUnknown) {
			return actionResultStateConflict, 0, nil
		}
		rows, err := queries.CloseWebhookUnknownCycle(ctx, sqlcgen.CloseWebhookUnknownCycleParams{FinalizedAt: pgtime(now), OwnerScope: request.OwnerScope, DeliveryID: request.TargetID, CycleNumber: locked.CurrentCycle})
		if err != nil || rows != 1 {
			return actionMutationResult(rows, err)
		}
		rows, err = queries.CloseWebhookUnknownDelivery(ctx, sqlcgen.CloseWebhookUnknownDeliveryParams{FinalizedAt: pgtime(now), OwnerScope: request.OwnerScope, DeliveryID: request.TargetID, CycleNumber: locked.CurrentCycle})
		return actionMutationResult(rows, err)
	case ActionRetentionHold:
		payload, ok := request.Payload.(*RetentionHoldAction)
		if request.TargetKind != targetKindDelivery || !ok || payload == nil {
			return "", 0, fmt.Errorf("%w: retention hold is invalid", ErrConfig)
		}
		rows, err := queries.ApplyWebhookRetentionHold(ctx, sqlcgen.ApplyWebhookRetentionHoldParams{
			LegalHold: payload.Enabled, UpdatedAt: pgtime(now), OwnerScope: request.OwnerScope,
			DeliveryID: request.TargetID, ExpectedCycle: expected,
		})
		return actionMutationResult(rows, err)
	case ActionPrivacyDelete, ActionNamespaceRetire:
		return "", 0, fmt.Errorf("%w: privacy actions use their dedicated store owner", ErrConfig)
	default:
		return "", 0, fmt.Errorf("%w: unsupported operator action", ErrConfig)
	}
}

func manifestContains(manifest *SecretManifest, owner, destination, reference string) bool {
	_, err := manifest.Resolve(owner, destination, reference)
	return err == nil
}

func actionDuplicateRisk(request ActionRequest) bool {
	switch payload := request.Payload.(type) {
	case *RedriveAction:
		return payload != nil && payload.AcknowledgeDuplicateRisk
	case *CloseUnknownAction:
		return payload != nil && payload.AcknowledgeDuplicateRisk
	default:
		return false
	}
}

func actionRetainUntil(ctx context.Context, queries *sqlcgen.Queries, request ActionRequest, now time.Time) (time.Time, error) {
	switch request.TargetKind {
	case targetKindDelivery:
		row, err := queries.LockWebhookDeliveryForAction(ctx, sqlcgen.LockWebhookDeliveryForActionParams{OwnerScope: request.OwnerScope, DeliveryID: request.TargetID})
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrConflict
		}
		if err != nil {
			return time.Time{}, fmt.Errorf("read webhook action retention: %w", err)
		}
		return row.ActionRetainedUntil.Time.UTC(), nil
	case targetKindDestination:
		row, err := queries.ReadWebhookDestination(ctx, sqlcgen.ReadWebhookDestinationParams{OwnerScope: request.OwnerScope, DestinationID: request.TargetID, Generation: request.TargetGeneration})
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrConflict
		}
		if err != nil {
			return time.Time{}, fmt.Errorf("read webhook destination action retention: %w", err)
		}
		var policy DeliveryPolicy
		if err := json.Unmarshal(row.Policy, &policy); err != nil {
			return time.Time{}, fmt.Errorf("%w: decode retained webhook action policy", ErrConflict)
		}
		return now.Add(policy.Horizons.Action), nil
	default:
		return time.Time{}, fmt.Errorf("%w: action target kind is invalid", ErrConfig)
	}
}

func actionMutationResult(rows int64, err error) (string, int64, error) {
	if err != nil {
		return "", 0, fmt.Errorf("apply webhook action mutation: %w", err)
	}
	if rows != 1 {
		return actionResultStateConflict, 0, nil
	}
	return "applied", 0, nil
}
