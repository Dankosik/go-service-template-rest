package postgreswebhook

import (
	"net/http"
	"testing"
	"time"
)

func TestWebhookOutcomeClassifier(t *testing.T) {
	for status := 100; status <= 599; status++ {
		got := ClassifyOutcome(TransportEvidence{StatusCode: status, MayHaveSent: true})
		if status >= 200 && status <= 299 && got != OutcomeHTTPAccepted {
			t.Fatalf("status %d = %s", status, got)
		}
		if status == 302 && got != OutcomeHTTPRejected {
			t.Fatalf("redirect = %s", got)
		}
	}
	if got := ClassifyOutcome(TransportEvidence{DefinitelyNotSent: true}); got != OutcomeDefinitelyNotSentRetry {
		t.Fatalf("before send = %s", got)
	}
	if got := ClassifyOutcome(TransportEvidence{MayHaveSent: true}); got != OutcomeTransportAmbiguous {
		t.Fatalf("possible send = %s", got)
	}
}

func TestWebhookRetryAndSummary(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	if got, ok := ParseRetryAfter("120", "", now, time.Minute); !ok || got != time.Minute {
		t.Fatalf("Retry-After = %v, %t", got, ok)
	}
	if got := CumulativeSummary(OutcomeUnknown, OutcomeDefinitelyNotSentRetry); got != OutcomeUnknown {
		t.Fatalf("summary = %s", got)
	}
	if got := CumulativeSummary(OutcomeUnknown, OutcomeHTTPAccepted); got != OutcomeHTTPAccepted {
		t.Fatalf("accepted summary = %s", got)
	}
	if got := DecorrelatedJitter(time.Second, time.Second, 10*time.Second, 0); got != time.Second {
		t.Fatalf("minimum jitter = %v", got)
	}
	if got := DecorrelatedJitter(time.Second, time.Second, 10*time.Second, uint64(2*time.Second)); got != 3*time.Second {
		t.Fatalf("maximum first jitter = %v", got)
	}
}

func TestRetryDueAndRetryAfterEdges(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	deadline := now.Add(time.Minute)

	if due, err := RetryDue(now, deadline, 10*time.Second, 20*time.Second); err != nil || !due.Equal(now.Add(20*time.Second)) {
		t.Fatalf("RetryDue() = %v, %v", due, err)
	}
	for _, test := range []struct {
		name        string
		deadline    time.Time
		local, hint time.Duration
	}{
		{name: "deadline exhausted", deadline: now},
		{name: "delay exceeds deadline", deadline: deadline, local: time.Minute + time.Second},
		{name: "delay equals deadline", deadline: deadline, local: time.Minute},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := RetryDue(now, test.deadline, test.local, test.hint); err == nil {
				t.Fatal("RetryDue() error = nil")
			}
		})
	}

	for _, test := range []struct {
		name      string
		raw, date string
		max       time.Duration
		want      time.Duration
		ok        bool
	}{
		{name: "invalid", raw: "no", max: time.Minute},
		{name: "past date", raw: now.Add(-time.Second).Format(http.TimeFormat), max: time.Minute},
		{name: "date relative", raw: now.Add(30 * time.Second).Format(http.TimeFormat), date: now.Add(10 * time.Second).Format(http.TimeFormat), max: time.Minute, want: 20 * time.Second, ok: true},
		{name: "disabled", raw: "1", ok: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseRetryAfter(test.raw, test.date, now, test.max)
			if got != test.want || ok != test.ok {
				t.Fatalf("ParseRetryAfter() = %v, %t; want %v, %t", got, ok, test.want, test.ok)
			}
		})
	}
}
