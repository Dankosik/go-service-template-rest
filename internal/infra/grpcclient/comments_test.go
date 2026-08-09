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
// half. The walkers are duplicated rather than shared: a support package
// importable by each copy would be non-test surface built for tests, and every
// package's corpus, prose allowlist, and path root are its own anyway.
// profile:authn-oidc-jwt:start
//
// internal/infra/oidcjwt/docs_test.go is a third copy on the same reasoning.
// profile:authn-oidc-jwt:end

package grpcclient

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// commentIdentifier matches a token a comment could only have meant as a Go
// identifier: one interior lowercase-then-uppercase hump. Ordinary English does
// not produce that shape, which is what lets the corpus be walked instead of
// listed.
var commentIdentifier = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_]*[a-z][A-Z][A-Za-z0-9_]*\b`)

// commentFile matches a bare Go filename, which a comment in this package can
// only have meant as one of its own files. The leading letter is required so
// that a suffix such as _test.go is not read as a file.
var commentFile = regexp.MustCompile(`\b[a-z][a-z_]*\.go\b`)

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
// comments may name. Reading a directory's declarations is not importing it, so
// the import direction those packages enforce is not in play. A directory an
// inactive profile removed is skipped rather than failed.
var commentedNameSources = []string{
	filepath.Join("..", "..", "reqctx"),
	filepath.Join("..", "grpc"),
}

func TestCommentNamesResolve(t *testing.T) {
	t.Parallel()

	declared := packageDeclarations(t, ".")
	for _, source := range commentedNameSources {
		for name := range packageDeclarations(t, source) {
			declared[name] = struct{}{}
		}
	}
	files := packageFiles(t, ".")
	comments := packageComments(t, ".")
	// The walk answers empty for a directory that is not there, so a walk that
	// collected nothing would pass every check below while covering nothing.
	if len(comments) == 0 {
		t.Fatal("the comment walk collected nothing, and this package is not commentless")
	}

	for _, comment := range comments {
		for _, name := range commentIdentifier.FindAllString(comment.text, -1) {
			if _, prose := commentProseWords[name]; prose {
				continue
			}
			if _, ok := declared[name]; !ok {
				t.Errorf(
					"%s:%d names %q, which none of the packages this one may name declares; "+
						"update the comment to the current name, or add it to commentProseWords if it is not an identifier",
					comment.file, comment.line, name,
				)
			}
		}
		for _, name := range commentFile.FindAllString(comment.text, -1) {
			// A filename inside a longer path names a file in another directory,
			// which TestPackageDocPathsResolve owns.
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
// navigation; elsewhere a slash is usually prose — an import path, an elision,
// or a pair of names like Serve/Shutdown — so widening this check would cost an
// allowlist worth more than the coverage. [TestCommentNamesResolve] still holds
// every comment in the package to the sibling files it names.
func TestPackageDocPathsResolve(t *testing.T) {
	t.Parallel()

	document, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read package documentation: %v", err)
	}

	for _, path := range documentedRepositoryPaths(string(document)) {
		if _, err := os.Stat(filepath.Join(repositoryRoot, path)); err != nil {
			t.Errorf("doc.go names the path %q, which does not exist: %v", path, err)
		}
	}
}

// repositoryRoot is this package's distance from the top of the repository,
// which every documented path is resolved against.
const repositoryRoot = "../../.."

// versionedFileExtensions are the suffixes that make a slashed token a file
// this repository keeps, whatever its first segment is. Without them a renamed
// directory under a mistyped root would be skipped rather than reported.
var versionedFileExtensions = []string{".go", ".md", ".proto", ".sh", ".sql", ".yaml", ".yml"}

// namesRepositoryPath separates a repository path from the slashes ordinary
// prose produces — "I/O", "interval/timeout", an import path like
// google.golang.org/grpc — without keeping a list of the prose.
//
// A path qualifies by naming a versioned file, or by starting at a directory
// that exists at the repository root. A token failing both is skipped, so a
// mistyped first segment is not reported; that is the price of not maintaining
// the prose list, and the rename this check exists for changes a later segment.
func namesRepositoryPath(path string) bool {
	if slices.Contains(versionedFileExtensions, filepath.Ext(path)) {
		return true
	}
	root, _, _ := strings.Cut(path, "/")
	info, err := os.Stat(filepath.Join(repositoryRoot, root))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// documentedRepositoryPaths returns every repository path doc.go names, with a
// bare Go filename resolved against this package. A basename that also appears
// inside a longer path is dropped, so a file named both ways is looked for once.
func documentedRepositoryPaths(document string) []string {
	slashed := regexp.MustCompile(`([A-Za-z0-9_.-]+/)+[A-Za-z0-9_.-]+`)
	bare := regexp.MustCompile(`\b[a-z_]+\.go\b`)

	paths := make([]string, 0)
	nested := make(map[string]struct{})
	for _, path := range slashed.FindAllString(document, -1) {
		path = strings.TrimRight(path, ".,;:")
		if !namesRepositoryPath(path) {
			continue
		}
		nested[filepath.Base(path)] = struct{}{}
		paths = append(paths, path)
	}
	for _, name := range bare.FindAllString(document, -1) {
		if _, inPath := nested[name]; !inPath {
			paths = append(paths, filepath.Join("internal", "infra", "grpcclient", name))
		}
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

// packageDeclarations returns every name one package directory declares, test
// files included, so a comment may name the test that owns a property the types
// cannot enforce. Struct fields are collected alongside top-level names because
// field comments name their own fields.
//
// A directory that is not there answers empty rather than failing: an inactive
// profile removes whole packages these comments may name, and the comments
// naming them are removed by the same profile markers.
func packageDeclarations(t *testing.T, dir string) map[string]struct{} {
	t.Helper()

	declared := make(map[string]struct{})
	eachCommentedGoFile(t, dir, parser.SkipObjectResolution, func(_ string, parsed *ast.File, _ *token.FileSet) {
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
	})
	return declared
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

	comments := make([]packageComment, 0)
	eachCommentedGoFile(t, dir, parser.ParseComments, func(name string, parsed *ast.File, fileSet *token.FileSet) {
		for _, group := range parsed.Comments {
			for _, line := range group.List {
				comments = append(comments, packageComment{
					file: name,
					line: fileSet.Position(line.Slash).Line,
					text: line.Text,
				})
			}
		}
	})
	return comments
}

// packageFiles lists one directory's files. It does not parse, so it keeps its
// own walk rather than paying eachCommentedGoFile's parse to learn a name.
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

func eachCommentedGoFile(
	t *testing.T,
	dir string,
	mode parser.Mode,
	visit func(name string, parsed *ast.File, fileSet *token.FileSet),
) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read directory %s: %v", dir, err)
	}

	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, entry.Name()), nil, mode)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		visit(entry.Name(), parsed, fileSet)
	}
}
