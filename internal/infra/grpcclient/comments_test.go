// Proof that the names this package's comments navigate by still resolve.
//
// The comments here carry the trust boundary itself: which three sources can
// otherwise put a reserved correlation key on the wire, which strip seam closes
// each, and the three places a new propagated value has to be added together.
// They navigate by naming those seams. Nothing in the Go toolchain checks any of
// it — doc links are not validated and a bare name in prose is just prose — so a
// rename that updated the code and not the comment would leave the only written
// statement of that invariant pointing at something that no longer exists.
//
// internal/infra/grpc/comments_test.go runs the same two checks over the server
// half, and internal/packagetest owns both checks and the walk under them. What
// stays here is what is this package's own: the corpus its comments may name,
// its prose allowlist, and its path root.
// profile:authn-oidc-jwt:start
//
// internal/infra/oidcjwt/docs_test.go is a third caller on the same reasoning.
// profile:authn-oidc-jwt:end

package grpcclient

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
	// google.golang.org/grpc types and constructors.
	"CallOption":              {},
	"ClientConn":              {},
	"ClientConnInterface":     {},
	"ClientStream":            {},
	"NewClient":               {},
	"StreamClientInterceptor": {},
	// google.golang.org/grpc dial options this package sets or refuses.
	"WithDefaultServiceConfig": {},
	"WithDisableServiceConfig": {},
	"WithNoProxy":              {},
	// The resolver and credentials contracts it implements against.
	"AuthorityOverrider": {},
	"GetDefaultScheme":   {},
	"NewCredentials":     {},
	"ServerName":         {},
	// OpenTelemetry, the technology.
	"OpenTelemetry": {},
}

// commentedNameSources are the other packages whose declarations this package's
// comments may name.
var commentedNameSources = []string{
	filepath.Join("..", "..", "reqctx"),
	filepath.Join("..", "..", "packagetest"),
	filepath.Join("..", "grpc"),
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

	packagetest.CheckDocumentedPaths(t, repositoryRoot, "internal/infra/grpcclient")
}

// repositoryRoot is this package's distance from the top of the repository,
// which every documented path is resolved against.
const repositoryRoot = "../../.."
