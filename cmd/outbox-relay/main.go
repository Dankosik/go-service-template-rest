package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/example/go-service-template-rest/cmd/outbox-relay/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	// profile:messaging-nats-jetstream:start
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
	// profile:messaging-nats-jetstream:end
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

var selectedPublisherBuilder bootstrap.PublisherBuilder

func main() {
	// profile:messaging-nats-jetstream:start
	selectedPublisherBuilder = bootstrap.BuildNATSPublisher
	// profile:messaging-nats-jetstream:end
	// No noop fallback: an outbox-only generated service leaves the builder nil
	// and fails before it can claim work.
	if err := bootstrap.Run(os.Args[1:], selectedPublisherBuilder); err != nil {
		reportFailure(os.Stderr, err)
		os.Exit(1)
	}
}

func reportFailure(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "outbox relay failed: error_class=%s\n", failureClass(err))
}

// failureClass is the complete operator-facing vocabulary for why this process
// exited, and the only enumeration of it anywhere. Every error Relay.Run can
// return has a case here; anything unmatched exits as "runtime", which is honest
// but tells an operator nothing. A stop reason added to postgresoutbox therefore
// belongs in this switch, in TestReportFailureIsBoundedAndSanitized, and in the
// exit-class table in docs/postgres-transactional-outbox.md.
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
	// profile:messaging-nats-jetstream:start
	case errors.Is(err, natsjs.ErrTerminal):
		return "messaging_terminal"
	// profile:messaging-nats-jetstream:end
	// A lost lease stops the relay, and it is the ordinary way a lease that is
	// too short or a second replica shows up. It reports the same class the
	// outbox.relay.operations metric uses, so the exit line and the dashboard
	// name one condition rather than two.
	case errors.Is(err, postgresoutbox.ErrLeaseLost):
		return "lost_lease"
	default:
		return "runtime"
	}
}
