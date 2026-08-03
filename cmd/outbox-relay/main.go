package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/example/go-service-template-rest/cmd/outbox-relay/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

func main() {
	// A derived service replaces nil with its selected transport adapter builder.
	// The template has no noop fallback: missing publication fails startup.
	if err := bootstrap.Run(os.Args[1:], nil); err != nil {
		reportFailure(os.Stderr, err)
		os.Exit(1)
	}
}

func reportFailure(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "outbox relay failed: error_class=%s\n", failureClass(err))
}

func failureClass(err error) string {
	switch {
	case config.ErrorType(err) != config.ErrorTypeUnknown:
		return "config"
	case errors.Is(err, postgres.ErrConfig), errors.Is(err, postgresoutbox.ErrConfig):
		return "config"
	case errors.Is(err, postgres.ErrConnect), errors.Is(err, postgres.ErrHealthcheck), errors.Is(err, postgres.ErrSaturated):
		return "postgres_unavailable"
	case errors.Is(err, postgresoutbox.ErrPublisherStuck):
		return "publisher_stuck"
	case errors.Is(err, postgresoutbox.ErrPublisherPanic):
		return "publisher_panic"
	case errors.Is(err, postgresoutbox.ErrProgressUnknown):
		return "progress_unknown"
	default:
		return "runtime"
	}
}
