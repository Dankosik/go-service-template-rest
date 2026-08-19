package postgreswebhook

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	if got, ok := ParseRetryAfter("120", "", base, time.Minute); !ok || got != time.Minute {
		t.Fatalf("delta Retry-After = %s, %t", got, ok)
	}
	date := base.Add(30 * time.Second).Format(http.TimeFormat)
	if got, ok := ParseRetryAfter(date, base.Format(http.TimeFormat), base, time.Minute); !ok || got != 30*time.Second {
		t.Fatalf("date Retry-After = %s, %t", got, ok)
	}
	if _, ok := ParseRetryAfter("invalid", "", base, time.Minute); ok {
		t.Fatal("invalid Retry-After accepted")
	}
}
