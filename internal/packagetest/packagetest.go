// Package packagetest checks that repository paths named by package comments
// still exist. The NATS package uses the check because its documentation points
// readers to sibling files and repository-level contracts that Go does not
// resolve.
package packagetest

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// comment is one comment line with the position that lets a failure name the
// line to edit rather than the package.
type comment struct {
	File string
	Line int
	Text string
}

// eachGoFile parses every Go file in dir, test files included, and hands each to
// visit. mode selects what the parse retains: comment positions, or nothing
// beyond the declarations.
//
// A directory that does not exist yields nothing rather than failing, which is
// what lets an inactive build profile remove a package another one's comments
// name. That makes an empty walk indistinguishable from a walk that found
// nothing to object to, so a caller names its own witness that the walk ran.
func eachGoFile(
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

// comments returns every comment line in dir.
func comments(tb testing.TB, dir string) []comment {
	tb.Helper()
	comments := make([]comment, 0)
	eachGoFile(tb, dir, parser.ParseComments, func(fileName string, parsed *ast.File, fileSet *token.FileSet) {
		for _, group := range parsed.Comments {
			for _, line := range group.List {
				comments = append(comments, comment{
					File: fileName,
					Line: fileSet.Position(line.Slash).Line,
					Text: line.Text,
				})
			}
		}
	})
	return comments
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

// documentedPaths returns every repository path document names, with a bare Go
// filename resolved against packageDir. A basename that also appears inside a
// longer path is dropped, so a file named both ways is looked for once.
//
// repositoryRoot is the caller's distance from the top of the repository, which
// every returned path is resolved against; packageDir is that package's own
// repository-relative directory.
func documentedPaths(document, repositoryRoot, packageDir string) []string {
	paths := make([]string, 0)
	nested := make(map[string]struct{})
	rootDirs := repositoryRootDirs(repositoryRoot)
	for _, path := range documentedSlashedPath.FindAllString(document, -1) {
		path = strings.TrimRight(path, ".,;:")
		nested[filepath.Base(path)] = struct{}{}
		if !namesRepositoryPath(path, rootDirs) {
			continue
		}
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

// repositoryRootDirs names the directories at the top of the repository, which
// is the set a documented path's first segment has to belong to.
//
// It is a membership test rather than a stat of repositoryRoot joined with that
// segment, because the segment is read out of prose: ".." joins to a directory
// above the repository and would answer a question this check never asked. A
// name can only match a directory that is actually there, so no token reaches
// outside. An unreadable root yields the empty set, which reports no paths
// rather than every path.
func repositoryRootDirs(repositoryRoot string) map[string]struct{} {
	dirs := make(map[string]struct{})
	entries, err := os.ReadDir(repositoryRoot)
	if err != nil {
		return dirs
	}
	for _, entry := range entries {
		if entry.IsDir() {
			dirs[entry.Name()] = struct{}{}
		}
	}
	return dirs
}

// namesRepositoryPath separates a repository path from the slashes ordinary
// prose produces — "I/O", "interval/timeout", an import path like
// google.golang.org/grpc — without keeping a list of the prose.
//
// Two shapes are rejected before either test below runs. A first segment
// carrying a dot is a module or domain path, never a directory this repository
// has: that rule is what keeps github.com/nats-io/nats.go out, which the
// versioned-extension test would otherwise admit because the module's own name
// ends in .go. A "..." segment is an elision a comment wrote for a reader.
//
// Past those, a path qualifies by naming a versioned file, or by starting at a
// directory that exists at the repository root. A token failing both is skipped,
// so a mistyped first segment is not reported; that is the price of not
// maintaining the prose list, and the rename this check exists for changes a
// later segment.
func namesRepositoryPath(path string, rootDirs map[string]struct{}) bool {
	if !fs.ValidPath(path) {
		return false
	}
	root, _, _ := strings.Cut(path, "/")
	if strings.Contains(root, ".") || slices.Contains(strings.Split(path, "/"), "...") {
		return false
	}
	if slices.Contains(versionedFileExtensions, filepath.Ext(path)) {
		return true
	}
	_, atRoot := rootDirs[root]
	return atRoot
}
