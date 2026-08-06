package oidcjwt

// Proof for the two documents that describe this package: docs/authentication.md,
// which publishes a contract to operators, and doc.go, which is the extension
// manual the next maintainer reads first. Both make claims about code, and both
// are held to it here.

import (
	"context"
	"errors"
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
)

func TestDocumentedTokenExample(t *testing.T) {
	guide := readAuthenticationGuide(t)
	for _, required := range []string{
		"AUTHN=oidc-jwt",
		"APP__AUTHN__ISSUER=https://",
		"APP__AUTHN__AUDIENCE=",
		"APP__AUTHN__TRUSTED_PROXY_CIDRS=",
		"RS256 access token",
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
	token := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	principal, err := verifier.Verify(t.Context(), token, TransportHTTP)
	if err != nil || principal.Subject != "opaque-subject" {
		t.Fatalf("documented signed-token flow = (%+v, %v), want valid opaque principal", principal, err)
	}
}

// TestDocumentedMetricReasonsMatchTheGuide holds every authn.reason label this
// package can emit to the closed set docs/authentication.md publishes.
//
// This closes the one obligation doc.go named as having no automated owner. The
// exhaustive linter makes a new [Kind] fail every switch in the package,
// verificationReason included, but nothing made it reach the operator guide — so
// a category added here and not there left operators reading a list that no
// longer described the service.
//
// The corpus is walked rather than listed, so there is no second enumeration of
// Kind to keep in step. Kind values are a contiguous iota run and [Error.Error]
// answers the default arm for anything outside it, which makes the first unnamed
// value the end of the run. That relies on every declared Kind having its own
// message, which is exactly what the exhaustive linter already holds Error to.
func TestDocumentedMetricReasonsMatchTheGuide(t *testing.T) {
	guide := readAuthenticationGuide(t)
	unnamed := Failure(0).Error()

	labels := []string{verificationReason(context.Canceled)}
	for kind := Kind(1); ; kind++ {
		failure := Failure(kind)
		if failure.Error() == unnamed {
			break
		}
		labels = append(labels, verificationReason(failure))
	}
	// A walk that stopped early would pass the loop below while covering nothing,
	// so it has to be shown to have reached the end of the declared run. The last
	// declared Kind is the cheapest witness for that.
	if !slices.Contains(labels, verificationReason(Failure(KindUntrustedTransport))) {
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

func readAuthenticationGuide(t *testing.T) string {
	t.Helper()
	document, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "authentication.md"))
	if err != nil {
		t.Fatalf("read authentication guide: %v", err)
	}
	return string(document)
}

// docIdentifier matches a token a comment could only have meant as a Go
// identifier: one interior lowercase-then-uppercase hump. Ordinary English does
// not produce that shape, which is what lets the corpus be walked instead of
// listed.
var docIdentifier = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_]*[a-z][A-Z][A-Za-z0-9_]*\b`)

// siblingFile matches a bare Go filename, which a comment in this package can
// only have meant as one of its own files. The leading letter is required so
// that a suffix such as _test.go is not read as a file.
var siblingFile = regexp.MustCompile(`\b[a-z][a-z_]*\.go\b`)

// docProseWords are the hump-shaped words in this package's comments that name
// a technology, an RFC term, or a standard-library method rather than a
// declaration this repository owns. Anything else that lands here is an
// identifier, and a new entry should be viewed with suspicion.
var docProseWords = map[string]struct{}{
	"OpenAPI":       {},
	"gRPC":          {},
	"KiB":           {},
	"MeterProvider": {},
	"NumericDate":   {},
	"ServerStream":  {},
	"UseNumber":     {},
}

// commentedNameSources are the other packages whose declarations this package's
// comments may name. Reading a directory's declarations is not importing it, so
// neither the config_no_runtime_owners depguard rule nor the import direction
// those packages enforce is in play. A directory an inactive profile removed is
// skipped rather than failed: init-module.sh drops internal/infra/grpc and this
// package's gRPC adapter together, so the comments naming its declarations are
// removed with them by the same profile markers.
var commentedNameSources = []string{
	filepath.Join("..", "..", "config"),
	filepath.Join("..", "..", "infra", "grpc"),
	filepath.Join("..", "..", "infra", "http"),
}

// TestCommentNamesResolve holds every name this package's comments navigate by
// to the code they navigate to.
//
// doc.go is the extension manual, but it is not what a reader meets first:
// arriving from a call site means opening verifier.go or refresh.go, and the
// comments outside doc.go carry roughly six times its prose while naming tests,
// sibling files, and declarations in other packages just as freely. Nothing in
// the Go toolchain checks any of it — the [Symbol] doc-link syntax is not
// validated, and a bare name in prose is just prose — so a rename would leave
// the comment a reader trusts most pointing at something that no longer exists.
// That is the one failure mode this package's comments cannot survive, because
// a reader who catches it stops trusting the rest.
func TestCommentNamesResolve(t *testing.T) {
	declared := packageDeclarations(t, ".")
	for _, source := range commentedNameSources {
		for name := range packageDeclarations(t, source) {
			declared[name] = struct{}{}
		}
	}
	files := packageFiles(t, ".")

	for _, comment := range packageComments(t, ".") {
		for _, name := range docIdentifier.FindAllString(comment.text, -1) {
			if _, prose := docProseWords[name]; prose {
				continue
			}
			if _, ok := declared[name]; !ok {
				t.Errorf(
					"%s:%d names %q, which none of the packages this one may name declares; "+
						"update the comment to the current name, or add it to docProseWords if it is not an identifier",
					comment.file, comment.line, name,
				)
			}
		}
		for _, name := range siblingFile.FindAllString(comment.text, -1) {
			// A filename inside a longer path names a file in another directory,
			// which documentedPaths owns.
			if strings.Contains(comment.text, "/"+name) {
				continue
			}
			if _, ok := files[name]; !ok {
				t.Errorf(
					"%s:%d names the file %q, which this package does not contain",
					comment.file, comment.line, name,
				)
			}
		}
	}
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

// packageDeclarations returns every name one package directory declares, test
// files included, so a comment may name the test that owns an extension's
// accepting and rejecting cases. Struct fields are collected alongside top-level
// names because the field comments on [Verifier] and refresher name their own
// fields — baseCtx, runDone, fetchedAt — as the regimes they describe.
//
// A directory that does not exist yields nothing rather than failing, which is
// what lets an inactive profile remove a package this one's comments name.
func packageDeclarations(t *testing.T, dir string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]struct{}{}
	}
	if err != nil {
		t.Fatalf("read package directory %s: %v", dir, err)
	}
	declared := make(map[string]struct{})
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch named := node.(type) {
			case *ast.FuncDecl:
				declared[named.Name.Name] = struct{}{}
			case *ast.TypeSpec:
				declared[named.Name.Name] = struct{}{}
			case *ast.ValueSpec:
				for _, name := range named.Names {
					declared[name.Name] = struct{}{}
				}
			case *ast.Field:
				for _, name := range named.Names {
					declared[name.Name] = struct{}{}
				}
			}
			return true
		})
	}
	return declared
}

// declaredConstants returns the string value of every constant in dir declared
// with the named type. Reading them out of the source is what keeps a test that
// checks a published vocabulary from holding a second copy of it.
func declaredConstants(t *testing.T, dir, typeName string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package directory %s: %v", dir, err)
	}
	values := make([]string, 0)
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
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
					t.Fatalf("unquote %s in %s: %v", literal.Value, entry.Name(), err)
				}
				values = append(values, unquoted)
			}
			return true
		})
	}
	slices.Sort(values)
	return slices.Compact(values)
}

// packageComment is one comment line with the position that lets a failure name
// the line to edit rather than the package.
type packageComment struct {
	file string
	line int
	text string
}

func packageComments(t *testing.T, dir string) []packageComment {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package directory %s: %v", dir, err)
	}
	comments := make([]packageComment, 0)
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, entry.Name()), nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		for _, group := range parsed.Comments {
			for _, line := range group.List {
				comments = append(comments, packageComment{
					file: entry.Name(),
					line: fileSet.Position(line.Slash).Line,
					text: line.Text,
				})
			}
		}
	}
	return comments
}

func packageFiles(t *testing.T, dir string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package directory %s: %v", dir, err)
	}
	files := make(map[string]struct{})
	for _, entry := range entries {
		if !entry.IsDir() {
			files[entry.Name()] = struct{}{}
		}
	}
	return files
}
