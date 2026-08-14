package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
)

func TestFailureClassAndReportFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{config.ErrValidate, "config"},
		{postgres.ErrConfig, "config"},
		{postgres.ErrConnect, "postgres_unavailable"},
		{postgres.ErrHealthcheck, "postgres_unavailable"},
		{postgres.ErrSaturated, "postgres_unavailable"},
		{postgreswebhook.ErrClockRegression, "clock_regression"},
		{postgreswebhook.ErrDrainUnsafe, "drain_unsafe"},
		{errors.New("unexpected"), "runtime"},
	}
	for _, test := range tests {
		if got := failureClass(test.err); got != test.want {
			t.Fatalf("failureClass(%v) = %q, want %q", test.err, got, test.want)
		}
	}

	var output bytes.Buffer
	reportFailure(&output, postgreswebhook.ErrDrainUnsafe)
	if got, want := output.String(), "webhook worker failed: error_class=drain_unsafe\n"; got != want {
		t.Fatalf("reportFailure() = %q, want %q", got, want)
	}
}
