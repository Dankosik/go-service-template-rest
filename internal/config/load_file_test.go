package //nolint:paralleltest // This test mutates process-global environment or working directory.

// TestLoadsAKubernetesProjectedConfigFile is the deployment the deleted path
// policy made impossible.
//
// kubelet does not write a ConfigMap or Secret volume as plain files: it writes a
// timestamped directory, a `..data` symlink pointing at it, and one symlink per
// key pointing through `..data`, which is how an update stays atomic. The policy
// refused those symlinks twice over, so the only two ways Kubernetes supplies a
// config file both failed at boot. The path arrives on this process's own argv, at
// the same trust level as the binary, so there was nothing on the other side of
// that trade.
config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadsAKubernetesProjectedConfigFile(t *testing.T) {
	resetConfigEnv(t)
	// Anything other than local is what used to switch the hardened policy on.
	t.Setenv("APP__APP__ENV", "production")

	mountPath := projectedConfigMount(t, `
http:
  addr: ":18081"
`)

	cfg, _, err := LoadDetailed(LoadOptions{ConfigPath: mountPath})
	if err != nil {
		t.Fatalf("LoadDetailed() with a projected ConfigMap error = %v", err)
	}
	if cfg.HTTP.Addr != ":18081" {
		t.Fatalf("HTTP.Addr = %q, want the projected file's value", cfg.HTTP.Addr)
	}
}

// TestRejectsAConfigDirectory keeps the one structural check worth having: a path
// that names a directory is a manifest mistake, and reading it would otherwise
// fail with a message about YAML.
func TestRejectsAConfigDirectory(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: t.TempDir()})
	if err == nil {
		t.Fatal("LoadDetailed() with a directory error = nil, want rejection")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("error = %v, want the directory named", err)
	}
}

// TestRejectsAnOversizedConfigFile keeps the read bounded. The path is trusted;
// what is behind it still does not have to be what the manifest author expected.
func TestRejectsAnOversizedConfigFile(t *testing.T) {
	t.Parallel()
	resetConfigEnv(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	oversized := append([]byte("# "), make([]byte, maxConfigFileSizeBytes)...)
	if err := os.WriteFile(path, oversized, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatal("LoadDetailed() with an oversized file error = nil, want rejection")
	}
	if !strings.Contains(err.Error(), "exceeds max size limit") {
		t.Fatalf("error = %v, want the size limit named", err)
	}
}

// projectedConfigMount recreates kubelet's atomic-writer layout and returns the
// path a pod spec would name.
func projectedConfigMount(t *testing.T, content string) string {
	t.Helper()

	mount := t.TempDir()
	revision := filepath.Join(mount, "..2026_07_25_00_00_00.123456789")
	if err := os.Mkdir(revision, 0o750); err != nil {
		t.Fatalf("create revision directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(revision, "config.yaml"), []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write projected config: %v", err)
	}
	if err := os.Symlink(revision, filepath.Join(mount, "..data")); err != nil {
		t.Fatalf("link ..data: %v", err)
	}
	if err := os.Symlink(filepath.Join("..data", "config.yaml"), filepath.Join(mount, "config.yaml")); err != nil {
		t.Fatalf("link projected key: %v", err)
	}
	return filepath.Join(mount, "config.yaml")
}
