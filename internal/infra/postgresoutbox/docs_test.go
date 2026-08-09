package postgresoutbox

// Proof for the claims this package's own documentation makes about its layout.
//
// doc.go is the extension manual, and it navigates by file name: which file owns
// a step of the cycle, which one holds the budget, which statements moved where.
// It also states that those names are a lint contract — .golangci.yml exempts
// store*.go and notify*.go from the driver and sqlc boundaries, so a file renamed
// out of the glob moves pgx outside the exemption and depguard then reports the
// import rather than the rename that caused it.
//
// Nothing in the Go toolchain checks any of that. A bare filename in a comment is
// prose, a [Symbol] doc link is not validated, and an exemption glob that no
// longer matches anything is silently dead. The comments in this package carry
// more of its reasoning than most, which is exactly why a stale one is expensive:
// a reader who catches the manual pointing at a file that does not exist stops
// trusting the rest of it.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The two shapes the checks below read out of prose: a bare Go filename, which a
// comment here can only have meant as one of this package's own files, and a
// slashed repository path.
var (
	commentedFile = regexp.MustCompile(`\b[a-z][a-z0-9_]*\.go\b`)
	commentedPath = regexp.MustCompile(`([A-Za-z0-9_.-]+/)+[A-Za-z0-9_.-]+`)
)

// absentByDesign are the filenames this package's comments name in order to say
// the file must not exist. Each is a counter-example in prose — the driver
// doubles are "store_fixtures_test.go rather than a plain fixtures_test.go" — so
// one of these resolving would mean the contract it illustrates had been broken.
// TestDriverBoundaryExemptionsMatchTheDriverFiles is what would then fail.
var absentByDesign = map[string]struct{}{
	"fixtures_test.go": {},
}

// driverPackages are the imports the two exempted boundaries deny. depguard
// matches a denied package by prefix, so pgx subpackages such as pgtype are
// covered by the first entry.
var driverPackages = []string{
	"github.com/jackc/pgx/v5",
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen",
}

func TestCommentedFileNamesResolve(t *testing.T) {
	t.Parallel()

	files := packageGoFiles(t)
	comments := packageComments(t)
	// A walk that collected nothing would pass the loop below while covering
	// nothing, and this package is emphatically not commentless.
	if len(comments) == 0 {
		t.Fatal("the comment walk collected nothing")
	}

	for _, comment := range comments {
		for _, name := range commentedFile.FindAllString(comment.text, -1) {
			// A filename inside a longer path names a file in another directory,
			// which TestPackageDocPathsResolve owns.
			if strings.Contains(comment.text, "/"+name) {
				continue
			}
			if _, absent := absentByDesign[name]; absent {
				continue
			}
			if _, ok := files[name]; !ok {
				t.Errorf("%s:%d names the file %q, which this package does not contain",
					comment.file, comment.line, name)
			}
		}
	}
}

// TestPackageDocPathsResolve holds the repository paths doc.go navigates by.
//
// Only doc.go is walked. It names paths deliberately, as the manual's
// navigation; elsewhere in the package a slash is usually prose — an import
// path, an elision, or a pair of names like claim/publish — so widening this
// would cost an allowlist worth more than the coverage.
func TestPackageDocPathsResolve(t *testing.T) {
	t.Parallel()

	document, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read package documentation: %v", err)
	}
	root := filepath.Join("..", "..", "..")

	paths := make([]string, 0)
	for _, named := range commentedPath.FindAllString(string(document), -1) {
		named = strings.TrimRight(named, ".,;:")
		// An import path is not a repository path, and doc.go names a few.
		if strings.Contains(named, ".com/") || !strings.Contains(named, ".") {
			continue
		}
		paths = append(paths, named)
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	if len(paths) == 0 {
		t.Fatal("doc.go names no repository paths, and it is the package's navigation")
	}
	for _, named := range paths {
		if _, err := os.Stat(filepath.Join(root, named)); err != nil {
			t.Errorf("doc.go names the path %q, which does not exist: %v", named, err)
		}
	}
}

// TestDriverBoundaryExemptionsMatchTheDriverFiles holds the file-name contract
// doc.go describes to the globs that implement it.
//
// The linter already reports a driver import that fell outside the exemption, so
// the direction worth a test is the one it cannot see: a glob that no longer
// matches any driver file at all. That is what a rename leaves behind — the
// exemption survives as dead configuration, the next file to take the old name
// silently inherits it, and doc.go still tells a maintainer the prefix is
// load-bearing.
func TestDriverBoundaryExemptionsMatchTheDriverFiles(t *testing.T) {
	t.Parallel()

	globs := exemptedGlobs(t)
	if len(globs) == 0 {
		t.Fatal("no .golangci.yml exemption names this package, and doc.go says two do")
	}
	matched := make(map[string]int, len(globs))

	for name, imports := range packageImports(t) {
		if !slices.ContainsFunc(imports, isDriverImport) {
			continue
		}
		covered := false
		for _, glob := range globs {
			ok, err := path.Match(glob, name)
			if err != nil {
				t.Fatalf("match %q against %q: %v", name, glob, err)
			}
			if ok {
				matched[glob]++
				covered = true
			}
		}
		if !covered {
			t.Errorf("%s imports the PostgreSQL driver but matches none of the exemptions %v; "+
				"rename it to the owning prefix, or move the exemption with the code", name, globs)
		}
	}
	for _, glob := range globs {
		if matched[glob] == 0 {
			t.Errorf(".golangci.yml exempts %q for this package, and no file it holds imports the driver; "+
				"delete the exemption or restore the file it was written for", glob)
		}
	}
}

func isDriverImport(imported string) bool {
	return slices.ContainsFunc(driverPackages, func(denied string) bool {
		return strings.HasPrefix(imported, denied)
	})
}

// exemptedGlobs returns the basename globs .golangci.yml exempts for this
// package, deduplicated: two boundaries name store*.go and only one names
// notify*.go, and this test cares which files a glob covers rather than which
// rule wrote it.
func exemptedGlobs(t *testing.T) []string {
	t.Helper()
	configuration, err := os.ReadFile(filepath.Join("..", "..", "..", ".golangci.yml"))
	if err != nil {
		t.Fatalf("read linter configuration: %v", err)
	}

	const marker = `"!**/internal/infra/postgresoutbox/`
	globs := make([]string, 0, 2)
	for line := range strings.SplitSeq(string(configuration), "\n") {
		_, exempted, found := strings.Cut(line, marker)
		if !found {
			continue
		}
		globs = append(globs, strings.TrimSuffix(strings.TrimSpace(exempted), `"`))
	}
	slices.Sort(globs)
	return slices.Compact(globs)
}

type commentedText struct {
	file string
	line int
	text string
}

func packageComments(t *testing.T) []commentedText {
	t.Helper()
	comments := make([]commentedText, 0)
	eachGoFile(t, parser.ParseComments, func(name string, parsed *ast.File, fileSet *token.FileSet) {
		for _, group := range parsed.Comments {
			comments = append(comments, commentedText{
				file: name,
				line: fileSet.Position(group.Pos()).Line,
				text: group.Text(),
			})
		}
	})
	return comments
}

func packageGoFiles(t *testing.T) map[string]struct{} {
	t.Helper()
	files := make(map[string]struct{})
	eachGoFile(t, parser.SkipObjectResolution, func(name string, _ *ast.File, _ *token.FileSet) {
		files[name] = struct{}{}
	})
	return files
}

// packageImports returns each Go file's import paths, test files included: the
// exemptions cover them too, which is why the driver doubles are named
// store_fixtures_test.go rather than a plain fixtures_test.go.
func packageImports(t *testing.T) map[string][]string {
	t.Helper()
	imports := make(map[string][]string)
	eachGoFile(t, parser.ImportsOnly, func(name string, parsed *ast.File, _ *token.FileSet) {
		paths := make([]string, 0, len(parsed.Imports))
		for _, imported := range parsed.Imports {
			paths = append(paths, strings.Trim(imported.Path.Value, `"`))
		}
		imports[name] = paths
	})
	return imports
}

func eachGoFile(t *testing.T, mode parser.Mode, visit func(string, *ast.File, *token.FileSet)) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, entry.Name(), nil, mode)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		visit(entry.Name(), parsed, fileSet)
	}
}
