package packagetest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The checks in this file are the rule three packages hold their own prose to.
// packagetest.go answers what is in a directory; this file is what a caller does
// with that answer, which is a second reason to exist and why the two are not
// one file.

// commentIdentifier matches a token a comment could only have meant as a Go
// identifier: one interior lowercase-then-uppercase hump. Ordinary English does
// not produce that shape, which is what lets the corpus be walked instead of
// listed.
var commentIdentifier = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_]*[a-z][A-Z][A-Za-z0-9_]*\b`)

// commentFile matches a bare Go filename, which a comment in the package under
// test can only have meant as one of its own files. The leading letter is
// required so that a suffix such as _test.go is not read as a file.
var commentFile = regexp.MustCompile(`\b[a-z][a-z_]*\.go\b`)

// CommentContract is what a package owns about its own comments, and is
// everything [CheckCommentNames] cannot derive from the directory itself.
type CommentContract struct {
	// Prose are the hump-shaped words in this package's comments that name a
	// technology, a protocol term, or a dependency's own declaration rather than
	// something this repository declares. Anything else that lands here is an
	// identifier, and a new entry should be viewed with suspicion.
	Prose map[string]struct{}
	// Sources are the other package directories whose declarations these
	// comments may name, relative to the package under test. Reading a
	// directory's declarations is not importing it, so the import direction
	// those packages enforce is not in play. A directory an inactive build
	// profile removed is skipped rather than failed.
	Sources []string
}

// CheckCommentNames holds every name a package's comments navigate by to the
// code they navigate to: a declaration in the package or in its named corpus,
// and a bare filename to a sibling file.
//
// Nothing in the Go toolchain checks any of it. Doc links are not validated and
// a bare name in prose is just prose, so a rename leaves the comment a reader
// trusts most pointing at something that no longer exists. That is the one
// failure mode a package whose comments carry its design cannot survive,
// because a reader who catches it stops trusting the rest.
//
// The directory is the caller's own, and is not a parameter: the contract is a
// statement about the package that declares it, so checking another package's
// comments against this one's allowlist would be answering a question nobody
// asked.
func CheckCommentNames(tb testing.TB, contract CommentContract) {
	tb.Helper()

	declared := Declarations(tb, ".")
	for _, source := range contract.Sources {
		for name := range Declarations(tb, source) {
			declared[name] = struct{}{}
		}
	}
	files := FileNames(tb, ".")
	comments := Comments(tb, ".")
	// The walk answers empty for a directory that is not there, so a walk that
	// collected nothing would pass every check below while covering nothing.
	if len(comments) == 0 {
		tb.Fatal("the comment walk collected nothing, and this package is not commentless")
	}

	for _, comment := range comments {
		for _, name := range commentIdentifier.FindAllString(comment.Text, -1) {
			if _, prose := contract.Prose[name]; prose {
				continue
			}
			if _, ok := declared[name]; !ok {
				tb.Errorf(
					"%s:%d names %q, which none of the packages this one may name declares; "+
						"update the comment to the current name, or add it to CommentContract.Prose "+
						"if it is not an identifier",
					comment.File, comment.Line, name,
				)
			}
		}
		for _, name := range commentFile.FindAllString(comment.Text, -1) {
			// A filename inside a longer path names a file in another directory,
			// which CheckDocumentedPaths owns.
			if strings.Contains(comment.Text, "/"+name) {
				continue
			}
			if _, ok := files[name]; !ok {
				tb.Errorf(
					"%s:%d names the file %q, which this package does not contain",
					comment.File, comment.Line, name,
				)
			}
		}
	}
}

// CheckDocumentedPaths holds the repository paths a package's doc.go navigates
// by.
//
// Only doc.go is walked. It names paths deliberately, as the manual's
// navigation; elsewhere a slash is usually prose — an import path, an elision,
// or a pair of names like Serve/Shutdown — so widening this check would cost an
// allowlist worth more than the coverage. [CheckCommentNames] still holds every
// comment in the package to the sibling files it names.
//
// repositoryRoot is the caller's distance from the top of the repository, and
// packageDir is that package's own repository-relative directory; [DocumentedPaths]
// owns what each resolves.
func CheckDocumentedPaths(tb testing.TB, repositoryRoot, packageDir string) {
	tb.Helper()

	document, err := os.ReadFile("doc.go")
	if err != nil {
		tb.Fatalf("read package documentation: %v", err)
	}
	for _, path := range DocumentedPaths(string(document), repositoryRoot, packageDir) {
		if _, err := os.Stat(filepath.Join(repositoryRoot, path)); err != nil {
			tb.Errorf("doc.go names the path %q, which does not exist: %v", path, err)
		}
	}
}
