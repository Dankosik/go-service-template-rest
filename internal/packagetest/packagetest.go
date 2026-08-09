// Package packagetest reads a Go package's own source, for the tests that hold a
// package's prose to its code.
//
// Three packages run those tests — internal/infra/grpc, internal/infra/grpcclient,
// and internal/infra/oidcjwt — because their comments carry more of the design
// than the code does and navigate by naming things the toolchain never checks: a
// sibling file, a declaration in another package, the test that holds a property
// the types cannot. Both the walk and the checks over it are here rather than in
// each of them because three copies had already drifted into three spellings of
// one traversal and two of one rule, which is the cost that decision was taken
// to avoid.
//
// What stays with each package is what is actually its own: the corpus of
// packages its comments may name, its prose allowlist, and its distance from the
// repository root. This file takes a directory and answers what is in it;
// comment_contract.go is the rule a caller holds that answer to.
//
// It imports testing for the reason internal/config/configtest and
// internal/infra/postgres/pgtest do: a helper that returns an error instead would
// put four error branches in every test that walks a package, and no production
// binary imports it.
package packagetest

import (
	"errors"
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

// Comment is one comment line with the position that lets a failure name the
// line to edit rather than the package.
type Comment struct {
	File string
	Line int
	Text string
}

// EachGoFile parses every Go file in dir, test files included, and hands each to
// visit. mode selects what the parse retains: comment positions, or nothing
// beyond the declarations.
//
// A directory that does not exist yields nothing rather than failing, which is
// what lets an inactive build profile remove a package another one's comments
// name. That makes an empty walk indistinguishable from a walk that found
// nothing to object to, so a caller names its own witness that the walk ran.
func EachGoFile(
	tb testing.TB,
	dir string,
	mode parser.Mode,
	visit func(fileName string, parsed *ast.File, fileSet *token.FileSet),
) {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		tb.Fatalf("read package directory %s: %v", dir, err)
	}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, filepath.Join(dir, entry.Name()), nil, mode)
		if err != nil {
			tb.Fatalf("parse %s: %v", entry.Name(), err)
		}
		visit(entry.Name(), parsed, fileSet)
	}
}

// Declarations returns every name one package directory declares, test files
// included, so a comment may name the test that owns a property the types cannot
// enforce. Struct fields are collected alongside top-level names because field
// comments name their own fields.
func Declarations(tb testing.TB, dir string) map[string]struct{} {
	tb.Helper()
	declared := make(map[string]struct{})
	EachGoFile(tb, dir, parser.SkipObjectResolution, func(_ string, parsed *ast.File, _ *token.FileSet) {
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

// Comments returns every comment line in dir.
func Comments(tb testing.TB, dir string) []Comment {
	tb.Helper()
	comments := make([]Comment, 0)
	EachGoFile(tb, dir, parser.ParseComments, func(fileName string, parsed *ast.File, fileSet *token.FileSet) {
		for _, group := range parsed.Comments {
			for _, line := range group.List {
				comments = append(comments, Comment{
					File: fileName,
					Line: fileSet.Position(line.Slash).Line,
					Text: line.Text,
				})
			}
		}
	})
	return comments
}

// FileNames lists one directory's files. It does not parse, so it keeps its own
// walk rather than paying EachGoFile's parse to learn a name.
func FileNames(tb testing.TB, dir string) map[string]struct{} {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("read package directory %s: %v", dir, err)
	}
	files := make(map[string]struct{})
	for _, entry := range entries {
		if !entry.IsDir() {
			files[entry.Name()] = struct{}{}
		}
	}
	return files
}

// documentedSlashedPath matches a slash-separated token, which is what a path in
// prose looks like before [namesRepositoryPath] decides whether it is one.
var documentedSlashedPath = regexp.MustCompile(`([A-Za-z0-9_.-]+/)+[A-Za-z0-9_.-]+`)

// documentedBareGoFile matches a bare Go filename, which a package's own
// documentation can only have meant as one of its own files.
var documentedBareGoFile = regexp.MustCompile(`\b[a-z_]+\.go\b`)

// versionedFileExtensions are the suffixes that make a slashed token a file this
// repository keeps, whatever its first segment is. Without them a renamed
// directory under a mistyped root would be skipped rather than reported.
var versionedFileExtensions = []string{".go", ".md", ".proto", ".sh", ".sql", ".yaml", ".yml"}

// DocumentedPaths returns every repository path document names, with a bare Go
// filename resolved against packageDir. A basename that also appears inside a
// longer path is dropped, so a file named both ways is looked for once.
//
// repositoryRoot is the caller's distance from the top of the repository, which
// every returned path is resolved against; packageDir is that package's own
// repository-relative directory.
func DocumentedPaths(document, repositoryRoot, packageDir string) []string {
	paths := make([]string, 0)
	nested := make(map[string]struct{})
	for _, path := range documentedSlashedPath.FindAllString(document, -1) {
		path = strings.TrimRight(path, ".,;:")
		if !namesRepositoryPath(path, repositoryRoot) {
			continue
		}
		nested[filepath.Base(path)] = struct{}{}
		paths = append(paths, path)
	}
	for _, name := range documentedBareGoFile.FindAllString(document, -1) {
		if _, inPath := nested[name]; !inPath {
			paths = append(paths, filepath.Join(packageDir, name))
		}
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

// namesRepositoryPath separates a repository path from the slashes ordinary
// prose produces — "I/O", "interval/timeout", an import path like
// google.golang.org/grpc — without keeping a list of the prose.
//
// A path qualifies by naming a versioned file, or by starting at a directory
// that exists at the repository root. A token failing both is skipped, so a
// mistyped first segment is not reported; that is the price of not maintaining
// the prose list, and the rename this check exists for changes a later segment.
func namesRepositoryPath(path, repositoryRoot string) bool {
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
