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
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestPackageDocPathsResolve(t *testing.T) {
	t.Parallel()

	commentsByFile, err := packageComments(".")
	if err != nil {
		t.Fatalf("collect package comments: %v", err)
	}
	if len(commentsByFile) == 0 {
		t.Fatal("the comment walk collected nothing, and this package is not commentless")
	}
	rootDirs, err := repositoryRootDirs(repositoryRoot)
	if err != nil {
		t.Fatalf("read repository root %s: %v", repositoryRoot, err)
	}

	for _, file := range slices.Sorted(maps.Keys(commentsByFile)) {
		document := strings.Join(commentsByFile[file], "\n")
		for _, path := range documentedPaths(document, rootDirs) {
			if _, err := os.Stat(filepath.Join(repositoryRoot, path)); err != nil {
				t.Errorf("%s names the path %q, which cannot be read: %v", file, path, err)
			}
		}
	}
}

func TestDocumentedPaths(t *testing.T) {
	t.Parallel()

	rootDirs := map[string]struct{}{"docs": {}, "internal": {}}
	for _, test := range []struct {
		name     string
		document string
		want     []string
	}{
		{
			name:     "refuses traversal before a repository directory",
			document: "// see ../secrets/token.txt and ../../etc/passwd",
		},
		{
			name:     "refuses traversal after a repository directory",
			document: "// internal/../../outside.go",
		},
		{
			name:     "keeps root directories and versioned files",
			document: "// internal/infra/http owns it; docs/repo-architecture.md explains why",
			want:     []string{"docs/repo-architecture.md", "internal/infra/http"},
		},
		{
			name:     "skips prose slashes",
			document: "// I/O and interval/timeout, plus google.golang.org/grpc",
		},
		{
			name:     "resolves bare Go files against the package",
			document: "// lifecycle.go owns the refresh loop",
			want:     []string{"internal/infra/natsjs/lifecycle.go"},
		},
		{
			name:     "checks a file named both ways once",
			document: "// internal/infra/http/router.go, and router.go again",
			want:     []string{"internal/infra/http/router.go"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := documentedPaths(test.document, rootDirs); !slices.Equal(got, test.want) {
				t.Fatalf("documentedPaths() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDirectoryReadersFailClosed(t *testing.T) {
	t.Parallel()

	absent := filepath.Join(t.TempDir(), "absent")
	if _, err := packageComments(absent); err == nil {
		t.Fatal("packageComments() error = nil, want a read failure")
	}
	if _, err := repositoryRootDirs(absent); err == nil {
		t.Fatal("repositoryRootDirs() error = nil, want a read failure")
	}
}

func TestPackageCommentsReadsOnlyGoFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := []byte("package example\n\n// Package comment for the scanner.\n")
	if err := os.WriteFile(filepath.Join(dir, "widget.go"), source, 0o600); err != nil {
		t.Fatalf("write widget.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}

	got, err := packageComments(dir)
	if err != nil {
		t.Fatalf("packageComments(): %v", err)
	}
	if len(got) != 1 || !slices.Equal(got["widget.go"], []string{"// Package comment for the scanner."}) {
		t.Fatalf("packageComments() = %v, want only the widget.go comment", got)
	}
}

const (
	repositoryRoot = "../../.."
	packageDir     = "internal/infra/natsjs"
)

var documentedSlashedPath = regexp.MustCompile(`([A-Za-z0-9_.-]+/)+[A-Za-z0-9_.-]+`)

var documentedBareGoFile = regexp.MustCompile(`\b[a-z_]+\.go\b`)

var versionedFileExtensions = []string{".go", ".md", ".proto", ".sh", ".sql", ".yaml", ".yml"}

func packageComments(dir string) (map[string][]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read package directory %s: %w", dir, err)
	}

	commentsByFile := make(map[string][]string)
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		parsed, err := parser.ParseFile(
			fileSet,
			filepath.Join(dir, entry.Name()),
			nil,
			parser.ParseComments|parser.SkipObjectResolution,
		)
		if err != nil {
			return nil, fmt.Errorf("parse package file %s: %w", entry.Name(), err)
		}
		for _, group := range parsed.Comments {
			for _, line := range group.List {
				commentsByFile[entry.Name()] = append(commentsByFile[entry.Name()], line.Text)
			}
		}
	}
	return commentsByFile, nil
}

func documentedPaths(document string, rootDirs map[string]struct{}) []string {
	paths := make([]string, 0)
	nested := make(map[string]struct{})
	for _, path := range documentedSlashedPath.FindAllString(document, -1) {
		path = strings.TrimRight(path, ".,;:")
		nested[filepath.Base(path)] = struct{}{}
		if namesRepositoryPath(path, rootDirs) {
			paths = append(paths, path)
		}
	}
	for _, name := range documentedBareGoFile.FindAllString(document, -1) {
		if _, inPath := nested[name]; !inPath {
			paths = append(paths, filepath.Join(packageDir, name))
		}
	}
	slices.Sort(paths)
	return slices.Compact(paths)
}

func repositoryRootDirs(root string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read repository root %s: %w", root, err)
	}
	dirs := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			dirs[entry.Name()] = struct{}{}
		}
	}
	return dirs, nil
}

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
