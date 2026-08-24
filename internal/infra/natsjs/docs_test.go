package natsjs

// Proof that the repository paths this package's comments navigate by still
// resolve.
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
	"testing"

	"github.com/example/go-service-template-rest/internal/packagetest"
)

func TestPackageDocPathsResolve(t *testing.T) {
	t.Parallel()

	packagetest.CheckDocumentedPaths(t, repositoryRoot, "internal/infra/natsjs")
}

// repositoryRoot is this package's distance from the top of the repository,
// which every documented path is resolved against.
const repositoryRoot = "../../.."
