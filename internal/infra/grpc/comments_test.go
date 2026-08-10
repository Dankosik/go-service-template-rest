// Proof that the names this package's comments navigate by still resolve.
//
// The comments here carry more of the design than the code does — the
// interceptor order, why one policy shape serves both RPC kinds, which test
// proves a composition the types cannot — and they navigate by naming things:
// sibling files, declarations in other packages, and the tests that hold a
// property nothing structural enforces. Nothing in the Go toolchain checks any
// of it. Doc links are not validated, a bare name in prose is just prose, and a
// rename therefore leaves the comment a reader trusts most pointing at
// something that no longer exists.
//
// internal/infra/grpcclient/comments_test.go runs the same two checks over the
// client half, and internal/packagetest owns both checks and the walk under
// them. What stays here is what is this package's own: the corpus its comments
// may name, its prose allowlist, and its path root.
// profile:authn-oidc-jwt:start
//
// internal/infra/oidcjwt/docs_test.go is a third caller on the same reasoning.
// profile:authn-oidc-jwt:end

package grpcx

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
	// google.golang.org/grpc/codes.
	"AlreadyExists":     {},
	"InvalidArgument":   {},
	"ResourceExhausted": {},
	// google.golang.org/grpc server options, and the type they are.
	"ServerOption":      {},
	"MaxHeaderListSize": {},
	"MaxRecvMsgSize":    {},
	"MaxSendMsgSize":    {},
	// google.golang.org/genproto/googleapis/rpc/errdetails.
	"ErrorInfo": {},
	// The standard library.
	"MaxUint32":   {},
	"WithTimeout": {},
	// OpenTelemetry, the technology and one span method this package explains
	// why it does not call.
	"OpenTelemetry": {},
	"RecordError":   {},
}

// commentedNameSources are the other packages whose declarations this package's
// comments may name.
var commentedNameSources = []string{
	filepath.Join("..", "..", "failure"),
	filepath.Join("..", "..", "reqctx"),
	filepath.Join("..", "..", "config"),
	filepath.Join("..", "..", "infra", "http"),
	filepath.Join("..", "..", "infra", "oidcjwt"),
	filepath.Join("..", "..", "packagetest"),
	filepath.Join("..", "..", "..", "cmd", "service", "internal", "bootstrap"),
	"grpctest",
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

	packagetest.CheckDocumentedPaths(t, repositoryRoot, "internal/infra/grpc")
}

// repositoryRoot is this package's distance from the top of the repository,
// which every documented path is resolved against.
const repositoryRoot = "../../.."
