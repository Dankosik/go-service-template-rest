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

func TestReportFailureIsBoundedAndSanitized(t *testing.T) {
	const canary = "postgres://user:secret@db/app SELECT payload FROM outbox broker-detail"
	tests := []struct {
		name  string
		err   error
		class string
	}{
		{name: "config", err: fmt.Errorf("%w: %s", config.ErrValidate, canary), class: "config"},
		{name: "postgres", err: fmt.Errorf("%w: %s", postgres.ErrConnect, canary), class: "postgres_unavailable"},
		{name: "publisher stuck", err: errors.Join(postgresoutbox.ErrPublisherStuck, errors.New(canary)), class: "publisher_stuck"},
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
