//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func copyCurrentRepository(t *testing.T, source, destination string) {
	t.Helper()
	listed := exec.CommandContext(t.Context(), "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	listed.Dir = source
	output, err := listed.Output()
	if err != nil {
		t.Fatalf("list current repository files: %v", err)
	}
	for _, rawPath := range bytes.Split(output, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		relativePath := filepath.Clean(string(rawPath))
		if filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			t.Fatalf("repository file escaped checkout: %q", relativePath)
		}
		sourcePath := filepath.Join(source, relativePath)
		info, err := os.Lstat(sourcePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("inspect repository file %q: %v", relativePath, err)
		}
		destinationPath := filepath.Join(destination, relativePath)
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o750); err != nil {
			t.Fatalf("create repository fixture directory for %q: %v", relativePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(sourcePath)
			if err != nil {
				t.Fatalf("read repository symlink %q: %v", relativePath, err)
			}
			if err := os.Symlink(target, destinationPath); err != nil {
				t.Fatalf("copy repository symlink %q: %v", relativePath, err)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("unsupported repository entry %q with mode %s", relativePath, info.Mode())
		}
		copyRepositoryFile(t, sourcePath, destinationPath, info.Mode().Perm())
	}
}

func copyRepositoryFile(t *testing.T, source, destination string, mode os.FileMode) {
	t.Helper()
	input, err := os.Open(source)
	if err != nil {
		t.Fatalf("open repository file %q: %v", source, err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		t.Fatalf("create repository fixture file %q: %v", destination, err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		t.Fatalf("copy repository fixture file %q: %v", destination, err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("close repository fixture file %q: %v", destination, err)
	}
}

func TestCopyCurrentRepositoryUsesWorkingTree(t *testing.T) {
	source := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		command := exec.CommandContext(t.Context(), "git", args...)
		command.Dir = source
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	git("init", "-q")
	if err := os.WriteFile(filepath.Join(source, "tracked.txt"), []byte("staged"), 0o600); err != nil {
		t.Fatalf("write tracked fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, ".gitignore"), []byte("ignored.txt\n"), 0o600); err != nil {
		t.Fatalf("write ignore fixture: %v", err)
	}
	git("add", "tracked.txt", ".gitignore")
	if err := os.WriteFile(filepath.Join(source, "tracked.txt"), []byte("working tree"), 0o600); err != nil {
		t.Fatalf("modify tracked fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "untracked.txt"), []byte("untracked"), 0o600); err != nil {
		t.Fatalf("write untracked fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "ignored.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatalf("write ignored fixture: %v", err)
	}

	destination := filepath.Join(t.TempDir(), "copy")
	copyCurrentRepository(t, source, destination)
	for name, want := range map[string]string{"tracked.txt": "working tree", "untracked.txt": "untracked"} {
		got, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil || string(got) != want {
			t.Fatalf("copied %s = %q, %v, want %q", name, got, err, want)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "ignored.txt")); !os.IsNotExist(err) {
		t.Fatalf("ignored file copy error = %v, want not exists", err)
	}
}
