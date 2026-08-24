package packagetest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// newRepositoryRoot builds the directory shape DocumentedPaths resolves a first
// segment against: two real top-level directories and nothing else.
func newRepositoryRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{"internal", "docs"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o750); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n"), 0o600); err != nil {
		t.Fatalf("create Makefile: %v", err)
	}
	return root
}

// TestDocumentedPathsRefusesToLeaveTheRepository is the property that makes the
// first segment a name rather than a path: the tokens come out of prose, so a
// document that names "../" must not decide anything about a directory outside
// the repository.
func TestDocumentedPathsRefusesToLeaveTheRepository(t *testing.T) {
	t.Parallel()

	root := newRepositoryRoot(t)
	// The parent of a t.TempDir() really does exist, so a check that joined the
	// segment onto the root would answer true here.
	document := "// see ../secrets/token.txt and ../../etc/passwd"

	if got := documentedPaths(document, root, "internal/example"); len(got) != 0 {
		t.Fatalf("documentedPaths() = %v, want no path outside the repository", got)
	}
}

func TestDocumentedPathsRefusesTraversalAfterRepositoryDirectory(t *testing.T) {
	t.Parallel()

	root := newRepositoryRoot(t)
	outside := filepath.Join(filepath.Dir(root), "outside.go")
	if err := os.WriteFile(outside, []byte("package outside\n"), 0o600); err != nil {
		t.Fatalf("write outside.go: %v", err)
	}

	if got := documentedPaths("// internal/../../outside.go", root, "internal/example"); len(got) != 0 {
		t.Fatalf("documentedPaths() = %v, want no path outside the repository", got)
	}
}

func TestDocumentedPathsKeepsRootDirectoriesAndVersionedFiles(t *testing.T) {
	t.Parallel()

	root := newRepositoryRoot(t)
	document := "// internal/infra/http owns it; docs/repo-architecture.md explains why"

	got := documentedPaths(document, root, "internal/example")

	want := []string{"docs/repo-architecture.md", "internal/infra/http"}
	if !slices.Equal(got, want) {
		t.Fatalf("documentedPaths() = %v, want %v", got, want)
	}
}

// TestDocumentedPathsSkipsProseSlashes covers the reason the check exists at
// all: ordinary prose and import paths produce slashed tokens that name nothing
// in this repository.
func TestDocumentedPathsSkipsProseSlashes(t *testing.T) {
	t.Parallel()

	root := newRepositoryRoot(t)
	document := "// I/O and interval/timeout, plus google.golang.org/grpc"

	if got := documentedPaths(document, root, "internal/example"); len(got) != 0 {
		t.Fatalf("documentedPaths() = %v, want no path", got)
	}
}

func TestDocumentedPathsResolvesBareGoFileAgainstPackageDir(t *testing.T) {
	t.Parallel()

	root := newRepositoryRoot(t)
	document := "// lifecycle.go owns the refresh loop"

	got := documentedPaths(document, root, "internal/example")

	want := []string{"internal/example/lifecycle.go"}
	if !slices.Equal(got, want) {
		t.Fatalf("documentedPaths() = %v, want %v", got, want)
	}
}

// TestDocumentedPathsLooksForAFileNamedBothWaysOnce keeps a basename that also
// appears inside a longer path from being resolved a second time against the
// package directory, where it does not live.
func TestDocumentedPathsLooksForAFileNamedBothWaysOnce(t *testing.T) {
	t.Parallel()

	root := newRepositoryRoot(t)
	document := "// internal/infra/http/router.go, and router.go again"

	got := documentedPaths(document, root, "internal/example")

	want := []string{"internal/infra/http/router.go"}
	if !slices.Equal(got, want) {
		t.Fatalf("documentedPaths() = %v, want %v", got, want)
	}
}

// TestRepositoryRootDirsReportsNothingForAnUnreadableRoot keeps an unreadable
// root from reporting every documented token as a repository path.
func TestRepositoryRootDirsReportsNothingForAnUnreadableRoot(t *testing.T) {
	t.Parallel()

	if got := repositoryRootDirs(filepath.Join(t.TempDir(), "absent")); len(got) != 0 {
		t.Fatalf("repositoryRootDirs(absent) = %v, want empty", got)
	}
}

// newPackageDir writes one package the walkers can read.
func newPackageDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	source := `package example

// Package comment for the walker.
`
	if err := os.WriteFile(filepath.Join(dir, "widget.go"), []byte(source), 0o600); err != nil {
		t.Fatalf("write widget.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}
	return dir
}

func TestCommentsCarryTheFileAndLineToEdit(t *testing.T) {
	t.Parallel()

	collected := comments(t, newPackageDir(t))

	if len(collected) == 0 {
		t.Fatal("comments() returned nothing")
	}
	for _, comment := range collected {
		if comment.File != "widget.go" || comment.Line == 0 || comment.Text == "" {
			t.Fatalf("comments() entry = %+v, want widget.go with a line and text", comment)
		}
	}
}

// TestEachGoFileSkipsAnAbsentDirectory is what lets one package's comments name
// a package an inactive build profile removed.
func TestEachGoFileSkipsAnAbsentDirectory(t *testing.T) {
	t.Parallel()

	visited := 0
	eachGoFile(t, filepath.Join(t.TempDir(), "absent"), parser.SkipObjectResolution,
		func(string, *ast.File, *token.FileSet) { visited++ })

	if visited != 0 {
		t.Fatalf("eachGoFile visited %d files in an absent directory", visited)
	}
}

func TestEachGoFileVisitsOnlyGoFiles(t *testing.T) {
	t.Parallel()

	visited := make([]string, 0)
	eachGoFile(t, newPackageDir(t), parser.SkipObjectResolution,
		func(fileName string, _ *ast.File, _ *token.FileSet) {
			visited = append(visited, fileName)
		})

	if !slices.Equal(visited, []string{"widget.go"}) {
		t.Fatalf("eachGoFile visited %v, want only widget.go", visited)
	}
}
