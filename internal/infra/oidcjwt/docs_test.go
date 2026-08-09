package oidcjwt

// Proof for the two documents that describe this package: docs/authentication.md,
// which publishes a contract to operators, and doc.go, which is the extension
// manual the next maintainer reads first. Both make claims about code, and both
// are held to it here.

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/packagetest"
)

func TestDocumentedTokenExample(t *testing.T) {
	guide := readAuthenticationGuide(t)
	for _, required := range []string{
		"AUTHN=oidc-jwt",
		"APP__AUTHN__ISSUER=https://",
		"APP__AUTHN__AUDIENCE=",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS=",
		"RS256 access token",
		"non-empty `sub`, `client_id`, and `jti`",
		"Integer, fractional, and exponent notation",
		"`reqctx.Principal{Issuer, Subject,\nClientID}`",
		strconv.Itoa(int(MaxTokenLifetime/time.Minute)) + "-minute maximum issued lifetime",
		"`Health/Check` remains public",
		"protected streams receive a\ndeadline no later than the token's `exp`",
		"one-or-more ASCII spaces after the Bearer scheme",
		"bounded ±10% jitter",
		"## Bearer replay and revocation",
		"Sender-constrained access tokens",
		"provider-owned query is allowed on\n`jwks_uri`",
		"native gRPC is enabled with this profile, its server transport must be TLS",
		"There is no bypass, fake principal mode, or accept-all switch",
	} {
		if !strings.Contains(guide, required) {
			t.Fatalf("authentication guide is missing %q", required)
		}
	}

	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	signed := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	principal, err := verifier.verify(t.Context(), signed, transportHTTP)
	if err != nil ||
		principal.Issuer != testIssuer ||
		principal.Subject != "opaque-subject" ||
		principal.ClientID != "client-1" {
		t.Fatalf("documented signed-token flow = (%+v, %v), want verified issuer, subject, and client ID", principal, err)
	}
}

// TestDocumentedMetricReasonsMatchTheGuide holds every authn.reason label this
// package can emit to the closed set docs/authentication.md publishes.
//
// The exhaustive linter makes a new [Kind] fail every switch in the package, but
// nothing made it reach the operator guide — so a category added here and not
// there left operators reading a list that no longer described the service.
//
// The corpus is walked rather than listed, so there is no second enumeration of
// Kind to keep in step. Kind values are a contiguous iota run and [Error.Error]
// answers the default arm for anything outside it, which makes the first unnamed
// value the end of the run — which relies on every declared Kind having its own
// message, exactly what the exhaustive linter already holds Error to.
func TestDocumentedMetricReasonsMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	unnamed := NewError(0).Error()

	labels := []string{verificationReason(context.Canceled)}
	for kind := Kind(1); ; kind++ {
		failure := NewError(kind)
		if failure.Error() == unnamed {
			break
		}
		labels = append(labels, verificationReason(failure))
	}
	// A walk that stopped early would pass the loop below while covering nothing,
	// so it has to be shown to have reached the end of the declared run. The last
	// declared Kind is the cheapest witness for that.
	if !slices.Contains(labels, verificationReason(NewError(KindUntrustedTransport))) {
		t.Fatalf("the Kind walk collected %v and never reached the last declared category", labels)
	}

	for _, label := range labels {
		if !strings.Contains(guide, "`"+label+"`") {
			t.Errorf(
				"authentication guide does not publish the %q reason label; "+
					"add it to the authn.reason set operators are told to expect",
				label,
			)
		}
	}
}

// TestDocumentedTriggersMatchTheGuide holds every authn.refresh.trigger value
// this package can emit to the closed set docs/authentication.md publishes.
//
// It is [TestDocumentedMetricReasonsMatchTheGuide] for the other vocabulary that
// reaches operators. Making refreshTrigger a named type closed the compiler half
// — a fourth trigger now fails rateLimited under the exhaustive linter — but
// nothing made it reach the guide, so a trigger added and not published left
// operators reading a list that no longer described the service.
//
// The declarations are read from the source rather than listed here, so there is
// no second enumeration of refreshTrigger to keep in step.
func TestDocumentedTriggersMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	triggers := declaredConstants(t, ".", "refreshTrigger")

	// A walk that found nothing would pass the loop below while covering nothing.
	// The rate-limited trigger is the cheapest witness that it read the block.
	if !slices.Contains(triggers, string(triggerKeyMiss)) {
		t.Fatalf("the refreshTrigger walk collected %v and never reached the rate-limited trigger", triggers)
	}

	for _, trigger := range triggers {
		if !strings.Contains(guide, "`"+trigger+"`") {
			t.Errorf(
				"authentication guide does not publish the %q refresh trigger; "+
					"add it to the authn.refresh.trigger set operators are told to expect",
				trigger,
			)
		}
	}
}

func TestDocumentedProviderFailureReasonsMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	reasons := append(declaredConstants(t, ".", "providerError"), "unknown")
	if !slices.Contains(reasons, string(errProviderTransport)) {
		t.Fatalf("the providerError walk collected %v and never reached transport", reasons)
	}
	for _, reason := range reasons {
		if !strings.Contains(guide, "`"+reason+"`") {
			t.Errorf("authentication guide does not publish the %q provider failure reason", reason)
		}
	}
}

func readAuthenticationGuide(t *testing.T) string {
	t.Helper()
	document, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "authentication.md"))
	if err != nil {
		t.Fatalf("read authentication guide: %v", err)
	}
	return string(document)
}

// commentProseWords are the hump-shaped words in this package's comments that
// name a technology, an RFC term, or a standard-library method rather than a
// declaration this repository owns. Anything else that lands here is an
// identifier, and a new entry should be viewed with suspicion.
var commentProseWords = map[string]struct{}{
	"CloseIdleConnections": {},
	"OpenAPI":              {},
	"gRPC":                 {},
	"KiB":                  {},
	"MeasurementOption":    {},
	"MiB":                  {},
	"MeterProvider":        {},
	"NumericDate":          {},
	"OpenTelemetry":        {},
	"ServerStream":         {},
	"UseNumber":            {},
}

// commentedNameSources are the other packages whose declarations this package's
// comments may name. Reading a directory's declarations is not importing it, so
// neither the config_no_runtime_owners depguard rule nor the import direction
// those packages enforce is in play. A directory an inactive profile removed is
// skipped rather than failed: init-module.sh drops internal/infra/grpc and this
// package's gRPC adapter together, so the comments naming its declarations are
// removed with them by the same profile markers.
var commentedNameSources = []string{
	filepath.Join("..", "..", "authntrust"),
	filepath.Join("..", "..", "config"),
	filepath.Join("..", "..", "failure"),
	filepath.Join("..", "..", "infra", "grpc"),
	filepath.Join("..", "..", "infra", "http"),
	filepath.Join("..", "..", "packagetest"),
}

// TestCommentNamesResolve holds every name this package's comments navigate by
// to the code they navigate to.
//
// doc.go is the extension manual, but it is not what a reader meets first:
// arriving from a call site means opening verifier.go or lifecycle.go, and the
// comments outside doc.go carry roughly six times its prose while naming tests,
// sibling files, and declarations in other packages just as freely. Nothing in
// the Go toolchain checks any of it — the [Symbol] doc-link syntax is not
// validated, and a bare name in prose is just prose — so a rename would leave
// the comment a reader trusts most pointing at something that no longer exists.
// That is the one failure mode this package's comments cannot survive, because
// a reader who catches it stops trusting the rest.
func TestCommentNamesResolve(t *testing.T) {
	packagetest.CheckCommentNames(t, packagetest.CommentContract{
		Prose:   commentProseWords,
		Sources: commentedNameSources,
	})
}

// TestPackageDocPathsResolve holds the repository paths doc.go navigates by.
//
// Only doc.go is walked for these. It names paths deliberately, as the manual's
// navigation; elsewhere a slash is usually prose — a standard-library import
// path, an elision, or a pair of names like Run/Close — so widening this check
// would cost an allowlist worth more than the coverage.
// [TestCommentNamesResolve] still holds every comment in the package to the
// sibling files it names.
func TestPackageDocPathsResolve(t *testing.T) {
	document, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read package documentation: %v", err)
	}

	for _, path := range documentedPaths(string(document)) {
		if _, err := os.Stat(filepath.Join("..", "..", "..", path)); err != nil {
			t.Errorf("doc.go names the path %q, which does not exist: %v", path, err)
		}
	}
}

// documentedPaths returns every repository path doc.go names, with a bare Go
// filename resolved against this package. A basename that also appears inside a
// longer path is dropped, so cmd/.../startup_authn.go is not also looked for
// here.
//
// This test therefore keeps its own walk rather than calling
// packagetest.CheckDocumentedPaths. The shared one skips a slashed token it
// cannot resolve to a repository path, which buys the gRPC packages a prose-free
// check of doc.go and costs them a mistyped first segment; this document names
// no such prose, so reporting every token it cannot find is strictly the
// stronger check here.
func documentedPaths(doc string) []string {
	slashed := regexp.MustCompile(`([A-Za-z0-9_.-]+/)+[A-Za-z0-9_.-]+`)
	bare := regexp.MustCompile(`\b[a-z_]+\.go\b`)

	paths := make([]string, 0)
	nested := make(map[string]struct{})
	for _, path := range slashed.FindAllString(doc, -1) {
		path = strings.TrimRight(path, ".,;:")
		nested[filepath.Base(path)] = struct{}{}
		paths = append(paths, path)
	}
	for _, name := range bare.FindAllString(doc, -1) {
		if _, inPath := nested[name]; !inPath {
			paths = append(paths, filepath.Join("internal", "infra", "oidcjwt", name))
		}
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

// declaredConstants returns the string value of every constant in dir declared
// with the named type. Reading them out of the source is what keeps a test that
// checks a published vocabulary from holding a second copy of it.
func declaredConstants(t *testing.T, dir, typeName string) []string {
	t.Helper()
	values := make([]string, 0)
	packagetest.EachGoFile(t, dir, parser.SkipObjectResolution, func(fileName string, parsed *ast.File, _ *token.FileSet) {
		ast.Inspect(parsed, func(node ast.Node) bool {
			spec, isValue := node.(*ast.ValueSpec)
			if !isValue {
				return true
			}
			// A grouped const block repeats the type on each spec that states one,
			// so an untyped or differently typed neighbour is skipped rather than
			// inherited.
			if named, isIdent := spec.Type.(*ast.Ident); !isIdent || named.Name != typeName {
				return true
			}
			for _, value := range spec.Values {
				literal, isLiteral := value.(*ast.BasicLit)
				if !isLiteral || literal.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatalf("unquote %s in %s: %v", literal.Value, fileName, err)
				}
				values = append(values, unquoted)
			}
			return true
		})
	})
	slices.Sort(values)
	return slices.Compact(values)
}
