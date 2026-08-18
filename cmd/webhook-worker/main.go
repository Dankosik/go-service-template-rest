package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/example/go-service-template-rest/cmd/webhook-worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
)

func main() {
	if err := bootstrap.Run(os.Args[1:]); err != nil {
		reportFailure(os.Stderr, err)
		os.Exit(1)
	}
}

func reportFailure(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "webhook worker failed: error_class=%s\n", failureClass(err))
}

func failureClass(err error) string {
	switch {
	case config.ErrorType(err) != config.ErrorTypeUnknown, errors.Is(err, postgres.ErrConfig), errors.Is(err, postgreswebhook.ErrConfig):
		return "config"
	case errors.Is(err, postgres.ErrConnect), errors.Is(err, postgres.ErrHealthcheck):
		return "postgres_unavailable"
	case errors.Is(err, postgreswebhook.ErrClockRegression):
		return "clock_regression"
	case errors.Is(err, postgreswebhook.ErrDrainUnsafe):
		return "drain_unsafe"
	default:
		return "runtime"
	}
}
