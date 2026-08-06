package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

// One case per way the relay process can exit, plus the unmatched fallback. The
// exit line carries the class and nothing else, so it can never leak a DSN, a
// payload, SQL, or a broker's own error text.
//
// The list mirrors the sentinels Relay.Run can return, and drives every branch
// of failureClass rather than one representative per class: the config class
// has three sources — a rejected configuration, a store or relay that could not
// be built, and a publisher that was never registered — and they reach it
// through different sentinels.
//
// postgresoutbox's remaining exported errors — ErrNotFound, ErrInvalidEvent,
// ErrOrderingSequence, ErrRedriveRejected, ErrRedriveConflict,
// ErrPermanentPublication, and ErrPublicationNotAccepted — belong to Append,
// Get, Redrive, or a single publication attempt, so none of them ends the
// process and none needs a class. A new sentinel that can stop the relay needs
// a case here and in failureClass.
func TestReportFailureIsBoundedAndSanitized(t *testing.T) {
	const canary = "postgres://user:secret@db/app SELECT payload FROM outbox broker-detail"
	tests := []struct {
		name  string
		err   error
		class string
	}{
		{name: "config", err: fmt.Errorf("%w: %s", config.ErrValidate, canary), class: "config"},
		{
			name:  "outbox config",
			err:   fmt.Errorf("initialize outbox relay: %w: %s", postgresoutbox.ErrConfig, canary),
			class: "config",
		},
		{
			name:  "postgres config",
			err:   fmt.Errorf("initialize outbox postgres: %w: %s", postgres.ErrConfig, canary),
			class: "config",
		},
		{name: "postgres", err: fmt.Errorf("%w: %s", postgres.ErrConnect, canary), class: "postgres_unavailable"},
		{name: "publisher stuck", err: errors.Join(postgresoutbox.ErrPublisherStuck, errors.New(canary)), class: "publisher_stuck"},
		{name: "publisher panic", err: fmt.Errorf("%w: %s", postgresoutbox.ErrPublisherPanic, canary), class: "publisher_panic"},
		{name: "progress unknown", err: fmt.Errorf("%w: %s", postgresoutbox.ErrProgressUnknown, canary), class: "progress_unknown"},
		{name: "lost lease", err: fmt.Errorf("schedule outbox retry: %w", postgresoutbox.ErrLeaseLost), class: "lost_lease"},
		{name: "unknown adapter", err: errors.New(canary), class: "runtime"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			reportFailure(&stderr, test.err)
			want := "outbox relay failed: error_class=" + test.class + "\n"
			if got := stderr.String(); got != want || strings.Contains(got, canary) {
				t.Fatalf("stderr = %q, want sanitized %q", got, want)
			}
		})
	}
}
