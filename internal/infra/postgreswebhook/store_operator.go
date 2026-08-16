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

const actionStateConflict = "state_conflict"

func (s *Store) ApplyAction(ctx context.Context, request ActionRequest) (ActionReceipt, error) {
	fingerprint, err := request.Fingerprint()
	if err != nil {
		return ActionReceipt{}, err
	}
	if request.Kind == ActionPrivacyDelete || request.Kind == ActionNamespaceRetire || request.Note != "" {
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
			receipt = ActionReceipt{ActionID: request.ActionID, Result: row.Result, Replay: true}
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read webhook action: %w", err)
		}
		result, cycle, err := applyActionMutation(ctx, queries, request, now, s.options.ManifestRevision)
		if err != nil {
			return err
		}
		retainedUntil, err := queries.ReadWebhookActionRetainedUntil(ctx, sqlcgen.ReadWebhookActionRetainedUntilParams{
			TargetKind: request.TargetKind, OwnerScope: request.OwnerScope, TargetID: request.TargetID,
			TargetGeneration: request.TargetGeneration, SampledAt: pgtime(now),
		})
		if err != nil {
			return fmt.Errorf("read webhook action retention: %w", err)
		}
		if _, err := queries.InsertWebhookOperatorAction(ctx, sqlcgen.InsertWebhookOperatorActionParams{
			OwnerScope: request.OwnerScope, ActionID: request.ActionID, RequestFingerprint: fingerprint[:],
			ActorReference: request.Actor, ActionKind: string(request.Kind), TargetKind: request.TargetKind,
			TargetID: request.TargetID, TargetGeneration: request.TargetGeneration, ExpectedState: request.Expected,
			Reason: request.Reason, DuplicateRiskAcknowledged: request.DuplicateRisk, Result: result,
			RetainedUntil: retainedUntil, CreatedAt: pgtime(now),
		}); err != nil {
			return fmt.Errorf("insert webhook action: %w", err)
		}
		receipt = ActionReceipt{ActionID: request.ActionID, Result: result, Cycle: cycle}
		return nil
	})
	return receipt, err
}

//nolint:gocognit,cyclop // Each closed operator action keeps its validation beside its SQL transition.
func applyActionMutation(ctx context.Context, queries *sqlcgen.Queries, request ActionRequest, now time.Time, manifestRevision int64) (string, int64, error) {
	expected, err := strconv.ParseInt(request.Expected, 10, 64)
	if err != nil || expected < 0 {
		return "", 0, fmt.Errorf("%w: expected revision is invalid", ErrConfig)
	}
	switch request.Kind {
	case ActionDestinationState:
		disposition := map[string]string{activeDisposition: activeDisposition, "disabled": "administratively_disabled", "retired": "retired"}[request.Values[0]]
		if disposition == "" {
			return "", 0, fmt.Errorf("%w: destination disposition is invalid", ErrConfig)
		}
		rows, err := queries.ApplyWebhookDestinationState(ctx, sqlcgen.ApplyWebhookDestinationStateParams{Disposition: disposition, UpdatedAt: pgtime(now), OwnerScope: request.OwnerScope, DestinationID: request.TargetID, Generation: request.TargetGeneration, ExpectedRevision: expected})
		return actionMutationResult(rows, err)
	case ActionKeyRotation:
		secretRevision, err1 := strconv.ParseInt(request.Values[0], 10, 64)
		keyRevision, err2 := strconv.ParseInt(request.Values[1], 10, 64)
		overlapStartUnix, err3 := strconv.ParseInt(request.Values[4], 10, 64)
		validUntilUnix, err4 := strconv.ParseInt(request.Values[5], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || secretRevision <= 0 || keyRevision <= 0 || overlapStartUnix > now.Unix() || validUntilUnix <= max(overlapStartUnix, now.Unix()) || request.Values[6] == "" {
			return "", 0, fmt.Errorf("%w: key rotation is invalid", ErrConfig)
		}
		predecessor := request.Values[3]
		active := request.Values[2]
		rows, err := queries.ApplyWebhookKeyRotation(ctx, sqlcgen.ApplyWebhookKeyRotationParams{RequiredSecretRevision: secretRevision, KeyStateRevision: keyRevision, ActiveKeyReference: &active, PredecessorKeyReference: &predecessor, PredecessorValidUntil: pgtime(time.Unix(validUntilUnix, 0)), UpdatedAt: pgtime(now), OwnerScope: request.OwnerScope, DestinationID: request.TargetID, Generation: request.TargetGeneration, ExpectedRevision: expected})
		return actionMutationResult(rows, err)
	case ActionRedrive:
		if !request.DuplicateRisk {
			return "rejected", 0, nil
		}
		attempts, err1 := strconv.Atoi(request.Values[0])
		ageNanos, err2 := strconv.ParseInt(request.Values[1], 10, 64)
		if err1 != nil || err2 != nil || attempts < 1 || attempts > MaxAttempts || ageNanos <= 0 {
			return "", 0, fmt.Errorf("%w: redrive bounds are invalid", ErrConfig)
		}
		maximumAttempts, err := int32Value(attempts)
		if err != nil {
			return "", 0, err
		}
		locked, err := queries.LockWebhookDeliveryForAction(ctx, sqlcgen.LockWebhookDeliveryForActionParams{OwnerScope: request.OwnerScope, DeliveryID: request.TargetID})
		if errors.Is(err, pgx.ErrNoRows) {
			return "not_found", 0, nil
		}
		if err != nil {
			return "", 0, fmt.Errorf("lock webhook redrive: %w", err)
		}
		var policy DeliveryPolicy
		if err := json.Unmarshal(locked.PolicySnapshot, &policy); err != nil || policy.validate() != nil {
			return "", 0, fmt.Errorf("%w: retained redrive policy is invalid", ErrConflict)
		}
		if locked.CurrentCycle != expected || locked.State != string(DeliveryTerminal) && locked.State != string(DeliverySuspended) ||
			locked.Disposition == activeDisposition || locked.DestinationDisposition != activeDisposition ||
			locked.RequiredSecretRevision > manifestRevision || locked.ActiveKeyReference == nil || *locked.ActiveKeyReference == "" || !locked.PayloadRetained ||
			!locked.RedriveEligibleUntil.Time.After(now) || !locked.PayloadRetainedUntil.Time.After(now) ||
			!locked.ActiveRetainedUntil.Time.After(now) || !locked.AttemptsRetainedUntil.Time.After(now) ||
			!locked.ActionsRetainedUntil.Time.After(now) || !locked.DestinationRetainedUntil.Time.After(now) ||
			!locked.KeyReferencesRetainedUntil.Time.After(now) || locked.CumulativeSummary == string(OutcomeHTTPAccepted) ||
			attempts > policy.RedriveAttempts || time.Duration(ageNanos) > policy.RedriveAge {
			return actionStateConflict, 0, nil
		}
		deadline := now.Add(time.Duration(ageNanos))
		for _, retainedUntil := range []time.Time{
			locked.RedriveEligibleUntil.Time, locked.PayloadRetainedUntil.Time, locked.ActiveRetainedUntil.Time,
			locked.AttemptsRetainedUntil.Time, locked.ActionsRetainedUntil.Time, locked.DestinationRetainedUntil.Time,
			locked.KeyReferencesRetainedUntil.Time,
		} {
			if retainedUntil.Before(deadline) {
				deadline = retainedUntil
			}
		}
		if !deadline.After(now) {
			return actionStateConflict, 0, nil
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
		locked, err := queries.LockWebhookDeliveryForAction(ctx, sqlcgen.LockWebhookDeliveryForActionParams{OwnerScope: request.OwnerScope, DeliveryID: request.TargetID})
		if errors.Is(err, pgx.ErrNoRows) {
			return "not_found", 0, nil
		}
		if err != nil {
			return "", 0, fmt.Errorf("lock webhook unknown delivery: %w", err)
		}
		if locked.CurrentCycle != expected || locked.CumulativeSummary != string(OutcomeUnknown) {
			return actionStateConflict, 0, nil
		}
		rows, err := queries.CloseWebhookUnknownCycle(ctx, sqlcgen.CloseWebhookUnknownCycleParams{FinalizedAt: pgtime(now), OwnerScope: request.OwnerScope, DeliveryID: request.TargetID, CycleNumber: locked.CurrentCycle})
		if err != nil || rows != 1 {
			return actionMutationResult(rows, err)
		}
		rows, err = queries.CloseWebhookUnknownDelivery(ctx, sqlcgen.CloseWebhookUnknownDeliveryParams{FinalizedAt: pgtime(now), OwnerScope: request.OwnerScope, DeliveryID: request.TargetID, CycleNumber: locked.CurrentCycle})
		return actionMutationResult(rows, err)
	case ActionPrivacyDelete, ActionNamespaceRetire:
		return "", 0, fmt.Errorf("%w: privacy actions use their dedicated store owner", ErrConfig)
	default:
		return "", 0, fmt.Errorf("%w: unsupported operator action", ErrConfig)
	}
}

func actionMutationResult(rows int64, err error) (string, int64, error) {
	if err != nil {
		return "", 0, fmt.Errorf("apply webhook action mutation: %w", err)
	}
	if rows != 1 {
		return actionStateConflict, 0, nil
	}
	return "applied", 0, nil
}
