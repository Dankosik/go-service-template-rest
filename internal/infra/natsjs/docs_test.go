package natsjs

// Proof that the names this package's comments navigate by still resolve.
//
// This package navigates by filename more than most: doc.go's per-file map of
// the delivery cycle, and the headers atop worker.go, message.go, telemetry.go,
// vocabulary.go, errors.go, and message_wire.go all point a reader at a sibling
// file by name. Nothing in the Go toolchain checks any of it. Doc links are not
// validated and a bare name in prose is just prose, so a rename leaves the
// comment a reader trusts most pointing at something that no longer exists.
//
// This header deliberately names no profile-removable file. It is itself a
// comment in this package, so every file it names has to exist in each
// generated service too, and the outbox publisher's file does not survive
// OUTBOX=none.

import (
	"path/filepath"
	"testing"

	"github.com/example/go-service-template-rest/internal/packagetest"
)

// commentProseWords are the hump-shaped words in this package's comments that
// name a technology, a protocol term, or a dependency's own declaration rather
// than something this repository declares. Anything else that lands here is an
// identifier, and a new entry should be viewed with suspicion.
var commentProseWords = map[string]struct{}{
	// NATS and JetStream.
	"AckWait":            {},
	"ClosedHandler":      {},
	"FetchBytes":         {},
	"IsReconnecting":     {},
	"JetStream":          {},
	"MaxAckPending":      {},
	"MaxMsgSize":         {},
	"MaxReconnects":      {},
	"MaxRequestBatch":    {},
	"MaxRequestExpires":  {},
	"MaxRequestMaxBytes": {},
	"MaxWaiting":         {},
	// OpenAPI and Buf, the contract-check technologies message.go contrasts
	// event schemas with.
	"OpenAPI": {},
	// OpenTelemetry and semconv.
	"MeterProvider":  {},
	"OpenTelemetry":  {},
	"WithAttributes": {},
	// The standard library.
	"IsControl":     {},
	"WithoutCancel": {},
	// PostgreSQL, the database postgresoutbox and postgresinbox front.
	"PostgreSQL": {},
}

// commentedNameSources are the other packages whose declarations this
// package's comments may name.
var commentedNameSources = []string{
	filepath.Join("..", "postgresoutbox"),
	filepath.Join("..", "telemetry"),
	filepath.Join("..", "..", "config"),
	filepath.Join("..", "..", "reqctx"),
	filepath.Join("..", "..", "packagetest"),
	filepath.Join("..", "..", "..", "cmd", "worker", "internal", "bootstrap"),
}

func TestCommentNamesResolve(t *testing.T) {
	t.Parallel()

	packagetest.CheckCommentNames(t, packagetest.CommentContract{
		Prose:   commentProseWords,
		Sources: commentedNameSources,
	})
}

func TestPackageDocPathsResolve(t *testing.T) {
	t.Parallel()

	packagetest.CheckDocumentedPaths(t, repositoryRoot, "internal/infra/natsjs")
}

// repositoryRoot is this package's distance from the top of the repository,
// which every documented path is resolved against.
const repositoryRoot = "../../.."
