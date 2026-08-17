package postgreswebhook

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestFinalizationHelpers(t *testing.T) {
	for _, outcome := range []OutcomeClass{
		OutcomeDefinitelyNotSentRetry,
		OutcomeRetryableHTTPAmbiguous,
		OutcomeTransportAmbiguous,
	} {
		if !retryableOutcome(outcome) {
			t.Fatalf("retryableOutcome(%q) = false", outcome)
		}
	}
	if retryableOutcome(OutcomeHTTPAccepted) {
		t.Fatal("accepted outcome is retryable")
	}

	for _, test := range []struct {
		outcome, summary OutcomeClass
		want             string
	}{
		{OutcomeHTTPAccepted, OutcomeHTTPAccepted, string(OutcomeHTTPAccepted)},
		{OutcomeHTTPRejected, OutcomeHTTPRejected, string(OutcomeHTTPRejected)},
		{OutcomeLocallyDenied, OutcomeLocallyDenied, string(OutcomeLocallyDenied)},
		{OutcomeTransportAmbiguous, OutcomeUnknown, string(OutcomeUnknown)},
		{OutcomeTransportAmbiguous, OutcomeTransportAmbiguous, string(OutcomeAttemptsExhausted)},
	} {
		if got := terminalDisposition(test.outcome, test.summary); got != test.want {
			t.Errorf("terminalDisposition(%q, %q) = %q, want %q", test.outcome, test.summary, got, test.want)
		}
	}

	zero, set, err := nullableInt32(0)
	if err != nil || set || zero != 0 {
		t.Fatal("zero finalization values must be NULL")
	}
	value, set, err := nullableInt32(42)
	if err != nil || !set || value != 42 {
		t.Fatalf("nullableInt32(42) = %v", value)
	}
	if got := terminalDisposition("unexpected", ""); got != string(OutcomeAttemptsExhausted) {
		t.Fatalf("terminalDisposition(unexpected) = %q", got)
	}
}

func TestPrepareFinalizationValues(t *testing.T) {
	values, err := prepareFinalizationValues(Finalization{})
	if err != nil || values.status != nil || values.headerBytes != nil || values.bodyBytes != nil {
		t.Fatalf("prepare zero values = %+v, %v", values, err)
	}
	values, err = prepareFinalizationValues(Finalization{Evidence: TransportEvidence{StatusCode: 200}, ResponseHeaderBytes: 10, ResponseBodyBytes: 20})
	if err != nil || values.status == nil || values.headerBytes == nil || values.bodyBytes == nil || *values.status != 200 || *values.headerBytes != 10 || *values.bodyBytes != 20 {
		t.Fatalf("prepare values = %+v, %v", values, err)
	}
	for _, final := range []Finalization{
		{Evidence: TransportEvidence{StatusCode: math.MaxInt32 + 1}},
		{ResponseHeaderBytes: math.MaxInt32 + 1},
		{ResponseBodyBytes: math.MaxInt32 + 1},
	} {
		if _, err := prepareFinalizationValues(final); !errors.Is(err, ErrConfig) {
			t.Fatalf("overflow error = %v", err)
		}
	}
}

func TestFinalizationUpdate(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	locked := sqlcgen.LockWebhookFinalizationRow{AttemptsUsed: 1, MaximumAttempts: 2, DeadlineAt: pgtype.Timestamptz{Time: now.Add(time.Minute), Valid: true}}
	final := Finalization{LocalRetryDelay: time.Second}

	update := finalizationUpdateFor(locked, OutcomeDefinitelyNotSentRetry, final, now, 0)
	if update.summary != OutcomeDefinitelyNotSentRetry || update.deliveryState != string(DeliveryScheduled) || update.cycleDisposition != activeDisposition || !update.nextDueAt.Equal(now.Add(time.Second)) || update.terminalAt.Valid {
		t.Fatalf("scheduled update = %+v", update)
	}

	locked.MaximumAttempts = 1
	update = finalizationUpdateFor(locked, OutcomeDefinitelyNotSentRetry, final, now, 0)
	if update.summary != OutcomeAttemptsExhausted || update.deliveryState != string(DeliveryTerminal) || update.cycleDisposition != string(OutcomeAttemptsExhausted) || !update.terminalAt.Valid {
		t.Fatalf("exhausted update = %+v", update)
	}

	locked.MaximumAttempts = 2
	locked.CumulativeSummary = "none"
	update = finalizationUpdateFor(locked, OutcomeDefinitelyNotSentRetry, final, now, 0)
	if update.summary != "none" || update.cycleDisposition != activeDisposition {
		t.Fatalf("none summary update = %+v", update)
	}

	locked.MaximumAttempts = 1
	locked.CumulativeSummary = string(OutcomeUnknown)
	update = finalizationUpdateFor(locked, OutcomeRetryableHTTPAmbiguous, final, now, 0)
	if update.summary != OutcomeUnknown || update.cycleDisposition != string(OutcomeUnknown) || !update.terminalAt.Valid {
		t.Fatalf("unknown update = %+v", update)
	}

	locked.CumulativeSummary = ""
	locked.DeadlineAt.Time = now
	update = finalizationUpdateFor(locked, OutcomeRetryableHTTPAmbiguous, final, now, 0)
	if update.deliveryState != string(DeliveryTerminal) || !update.nextDueAt.Equal(now) {
		t.Fatalf("deadline update = %+v", update)
	}
}

func TestValidFinalization(t *testing.T) {
	attempt := ClaimedAttempt{Policy: DeliveryPolicy{ResponseHeaderBytes: 10, ResponseBodyBytes: 20}}
	if !validFinalization(attempt, Finalization{ResponseHeaderBytes: 10, ResponseBodyBytes: 20}) {
		t.Fatal("valid finalization rejected")
	}
	for _, final := range []Finalization{
		{ResponseHeaderBytes: -1},
		{ResponseHeaderBytes: 11},
		{ResponseHeaderBytes: 1, ResponseBodyBytes: -1},
		{ResponseHeaderBytes: 1, ResponseBodyBytes: 21},
		{ResponseHeaderBytes: 1, ResponseBodyBytes: 1, LocalRetryDelay: -1},
	} {
		if validFinalization(attempt, final) {
			t.Fatalf("invalid finalization accepted: %+v", final)
		}
	}
}
