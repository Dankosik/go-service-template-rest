package runtimeopts

// profile:messaging-nats-jetstream:start

import (
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

// Messaging builds the broker client options every messaging binary connects
// with: the API process producing events, the worker consuming them, and the
// outbox relay publishing them.
//
// The three connect in different roles and with different worker settings, but a
// bound on what may cross the wire is one deployment's answer, so a field added to
// natsjs.Config is added once here rather than in three composition roots — two of
// which would go on connecting without it and pass their own tests.
func Messaging(cfg config.MessagingConfig) natsjs.Config {
	return natsjs.Config{
		URLs:                 splitList(cfg.URLs),
		CredentialsFile:      cfg.CredentialsFile,
		RootCAFile:           cfg.RootCAFile,
		AllowPlaintext:       cfg.AllowPlaintext,
		AllowUnauthenticated: cfg.AllowUnauthenticated,
		Stream:               cfg.Stream,
		MaxPayloadBytes:      cfg.MaxPayloadBytes,
		MaxPendingPublishes:  cfg.MaxPendingPublishes,
	}
}

// splitList turns one comma-separated configured value into the slice an adapter
// takes. Entries are trimmed and empty ones are kept, because the adapter's own
// validation is what refuses a blank entry — dropping it here would let a
// trailing comma pass as a shorter list.
func splitList(value string) []string {
	values := make([]string, 0)
	for item := range strings.SplitSeq(value, ",") {
		values = append(values, strings.TrimSpace(item))
	}
	return values
}

// profile:messaging-nats-jetstream:end
