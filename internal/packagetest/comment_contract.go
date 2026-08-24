package packagetest

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// CheckDocumentedPaths holds the repository paths a package's comments navigate
// by.
//
// Every file's comments are walked, not only doc.go's. doc.go names paths
// deliberately, as the manual's navigation, but an ordinary file's header points
// across the repository just as often, and a path there is the one that survives
// a rename longest because nothing reads it: natsjs/config.go named
// internal/config/messaging.go for months after that file became
// messaging_config.go. Prose is separated from paths by [namesRepositoryPath]
// rather than by an allowlist.
//
// Each file's comments are joined before extraction so that a basename also
// written as part of a longer path in the same file is looked for once, which is
// what [documentedPaths] uses to tell the two apart.
//
// repositoryRoot is the caller's distance from the top of the repository, and
// packageDir is that package's own repository-relative directory; [documentedPaths]
// owns what each resolves.
func CheckDocumentedPaths(tb testing.TB, repositoryRoot, packageDir string) {
	tb.Helper()

	commentsByFile := make(map[string][]string)
	files := make([]string, 0)
	for _, comment := range comments(tb, ".") {
		if _, seen := commentsByFile[comment.File]; !seen {
			files = append(files, comment.File)
		}
		commentsByFile[comment.File] = append(commentsByFile[comment.File], comment.Text)
	}
	// The walk answers empty for a directory that is not there, so a walk that
	// collected nothing would pass every check below while covering nothing.
	if len(files) == 0 {
		tb.Fatal("the comment walk collected nothing, and this package is not commentless")
	}
	slices.Sort(files)

	for _, file := range files {
		document := strings.Join(commentsByFile[file], "\n")
		for _, path := range documentedPaths(document, repositoryRoot, packageDir) {
			if _, err := os.Stat(filepath.Join(repositoryRoot, path)); err != nil {
				tb.Errorf("%s names the path %q, which cannot be read: %v", file, path, err)
			}
		}
	}
}
