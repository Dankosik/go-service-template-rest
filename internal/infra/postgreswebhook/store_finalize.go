package postgreswebhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Finalization struct {
	Evidence            TransportEvidence
	ResponseHeaderBytes int
	ResponseBodyBytes   int
	RetryAfter          string
	ResponseDate        string
	LocalRetryDelay     time.Duration
}

type finalizationValues struct {
	status, headerBytes, bodyBytes *int32
}

type finalizationUpdate struct {
	summary                         OutcomeClass
	deliveryState, cycleDisposition string
	nextDueAt                       time.Time
	terminalAt                      pgtype.Timestamptz
}

func (s *Store) FinalizeAttempt(ctx context.Context, attempt ClaimedAttempt, final Finalization) (OutcomeClass, error) {
	if !s.valid() || !validFinalization(attempt, final) {
		return "", fmt.Errorf("%w: finalization evidence is invalid", ErrConfig)
	}
	values, err := prepareFinalizationValues(final)
	if err != nil {
		return "", err
	}
	outcome := ClassifyOutcome(final.Evidence)
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		finalizedAt, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		identity := attempt.Identity
		locked, err := queries.LockWebhookFinalization(ctx, sqlcgen.LockWebhookFinalizationParams{OwnerScope: identity.OwnerScope, DeliveryID: identity.DeliveryID, CycleNumber: identity.Cycle, AttemptID: identity.AttemptID, Fence: identity.Fence})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStaleAttempt
		}
		if err != nil {
			return fmt.Errorf("lock webhook finalization: %w", err)
		}
		if final.Evidence.MayHaveSent && !locked.MayHaveSent {
			return fmt.Errorf("%w: possible-send evidence was not durably authorized", ErrConflict)
		}
		update := finalizationUpdateFor(locked, outcome, attempt, final, finalizedAt)
		var retryAfterDelayMS *int64
		var retryAfterSource *string
		if evidence, ok := NormalizeRetryAfter(final.RetryAfter, final.ResponseDate, attempt.AttemptedAt, attempt.Policy.RetryAfterCap); ok {
			delay := evidence.Delay.Milliseconds()
			retryAfterDelayMS, retryAfterSource = &delay, &evidence.Source
		}
		outcomeText := string(outcome)
		rows, err := queries.FinalizeWebhookAttempt(ctx, sqlcgen.FinalizeWebhookAttemptParams{
			ResponseHeaderBytes: values.headerBytes, ResponseBodyBytes: values.bodyBytes, ResponseStatus: values.status,
			RetryAfterDelayMs: retryAfterDelayMS, RetryAfterSource: retryAfterSource,
			OutcomeClass: &outcomeText, FinalizedAt: pgtime(finalizedAt),
			OwnerScope: identity.OwnerScope, DeliveryID: identity.DeliveryID, CycleNumber: identity.Cycle,
			AttemptID: identity.AttemptID, Fence: identity.Fence, DeliveryState: update.deliveryState,
			NextDueAt: pgtime(update.nextDueAt), CumulativeSummary: string(update.summary), TerminalAt: update.terminalAt,
			CycleDisposition: update.cycleDisposition,
		})
		if err != nil {
			return fmt.Errorf("finalize webhook attempt: %w", err)
		}
		if rows != 1 {
			return ErrStaleAttempt
		}
		return nil
	})
	return outcome, err
}

func finalizationUpdateFor(locked sqlcgen.LockWebhookFinalizationRow, outcome OutcomeClass, attempt ClaimedAttempt, final Finalization, finalizedAt time.Time) finalizationUpdate {
	summary := CumulativeSummary(OutcomeClass(locked.CumulativeSummary), outcome)
	if outcome == OutcomeDefinitelyNotSentRetry && locked.CumulativeSummary == "none" {
		summary = OutcomeClass("none")
	}
	update := finalizationUpdate{summary: summary, deliveryState: string(DeliveryTerminal), cycleDisposition: terminalDisposition(outcome, summary), nextDueAt: finalizedAt}
	if retryableOutcome(outcome) && locked.AttemptsUsed < locked.MaximumAttempts {
		hint, _ := ParseRetryAfter(final.RetryAfter, final.ResponseDate, attempt.AttemptedAt, attempt.Policy.RetryAfterCap)
		if due, err := RetryDue(finalizedAt, locked.DeadlineAt.Time, final.LocalRetryDelay, hint); err == nil {
			update.deliveryState = string(DeliveryScheduled)
			update.cycleDisposition = activeDisposition
			update.nextDueAt = due
		}
	}
	if update.cycleDisposition != activeDisposition {
		update.terminalAt = pgtime(finalizedAt)
		if retryableOutcome(outcome) && update.summary != OutcomeUnknown {
			update.summary = OutcomeAttemptsExhausted
			update.cycleDisposition = string(OutcomeAttemptsExhausted)
		}
	}
	return update
}

func validFinalization(attempt ClaimedAttempt, final Finalization) bool {
	return final.ResponseHeaderBytes >= 0 && final.ResponseHeaderBytes <= attempt.Policy.ResponseHeaderBytes &&
		final.ResponseBodyBytes >= 0 && final.ResponseBodyBytes <= attempt.Policy.ResponseBodyBytes && final.LocalRetryDelay >= 0
}

func retryableOutcome(outcome OutcomeClass) bool {
	return outcome == OutcomeDefinitelyNotSentRetry || outcome == OutcomeRetryableHTTPAmbiguous || outcome == OutcomeTransportAmbiguous
}

func terminalDisposition(outcome, summary OutcomeClass) string {
	if summary == OutcomeUnknown {
		return string(OutcomeUnknown)
	}
	switch outcome {
	case OutcomeHTTPAccepted, OutcomeHTTPRejected, OutcomeLocallyDenied:
		return string(outcome)
	case OutcomeDefinitelyNotSentRetry, OutcomeRetryableHTTPAmbiguous, OutcomeTransportAmbiguous,
		OutcomeAttemptsExhausted, OutcomeUnknown, OutcomeClosedUnknown:
		return string(OutcomeAttemptsExhausted)
	}
	return string(OutcomeAttemptsExhausted)
}

func prepareFinalizationValues(final Finalization) (finalizationValues, error) {
	status, statusSet, err := nullableInt32(final.Evidence.StatusCode)
	if err != nil {
		return finalizationValues{}, err
	}
	headerBytes, headerBytesSet, err := nullableInt32(final.ResponseHeaderBytes)
	if err != nil {
		return finalizationValues{}, err
	}
	bodyBytes, bodyBytesSet, err := nullableInt32(final.ResponseBodyBytes)
	if err != nil {
		return finalizationValues{}, err
	}
	values := finalizationValues{}
	if statusSet {
		values.status = &status
	}
	if headerBytesSet {
		values.headerBytes = &headerBytes
	}
	if bodyBytesSet {
		values.bodyBytes = &bodyBytes
	}
	return values, nil
}

func nullableInt32(value int) (int32, bool, error) {
	if value == 0 {
		return 0, false, nil
	}
	converted, err := int32Value(value)
	if err != nil {
		return 0, false, err
	}
	return converted, true, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
