package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/knadh/koanf/v2"
)

func TestLocalAllowsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)

	target := writeTempConfig(t, `
http:
  addr: ":18080"
`)

	linkPath := filepath.Join(t.TempDir(), "config-link.yaml")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	cfg, _, err := LoadDetailed(LoadOptions{ConfigPath: linkPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.Addr != ":18080" {
		t.Fatalf("HTTP.Addr = %q, want :18080", cfg.HTTP.Addr)
	}
}

func TestNonLocalRejectsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	target := writeTempConfig(t, `
http:
  addr: ":8080"
`)

	tempDir := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", tempDir)

	linkPath := filepath.Join(tempDir, "config-link.yaml")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: linkPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error for non-local symlink config")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestTST005NonLocalRejectsWorldWritableConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
`)
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", filepath.Dir(configPath))

	if err := os.Chmod(configPath, 0o666); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error for world-writable config")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestConfigFileWithoutEnvironmentHintFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "")

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
`)
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", filepath.Dir(configPath))

	if err := os.Chmod(configPath, 0o666); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected fail-closed hardening error without explicit local env hint")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestLoadConfigFileRejectsWhitespaceOnlyPath(t *testing.T) {
	t.Parallel()

	_, err := loadConfigFileWithMetadata(context.Background(), koanf.New(keyDelimiter), " \t\n ", configFilePolicyLocal)
	if err == nil {
		t.Fatal("loadConfigFileWithMetadata() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrLoad) {
		t.Fatalf("error = %v, want ErrLoad", err)
	}
	if !strings.Contains(err.Error(), "empty config path") {
		t.Fatalf("error = %v, want empty config path detail", err)
	}
}

func TestNonLocalRejectsConfigOutsideAllowedRoots(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	otherRoot := t.TempDir()
	path := filepath.Join(otherRoot, "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", "")

	repoRoot := filepath.Join(t.TempDir(), "repo")
	repoConfigDir := filepath.Join(repoRoot, "env", "config")
	if err := os.MkdirAll(repoConfigDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(repoConfigDir, "nonlocal-default-root-test.yaml")
	content := "app:\n  env: prod\nhttp:\n  addr: \":8080\"\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection for repository config path in non-local mode")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalAllowedRootsRejectsUnsafeInputs(t *testing.T) {
	testCases := []struct {
		name             string
		allowedRoots     string
		wantErrorMessage string
	}{
		{
			name:             "relative root",
			allowedRoots:     "relative-config-root",
			wantErrorMessage: "APP_CONFIG_ALLOWED_ROOTS entries must be absolute paths",
		},
		{
			name:             "empty value uses default roots",
			allowedRoots:     "",
			wantErrorMessage: "outside allowed roots",
		},
		{
			name:             "delimiter only value produces no roots",
			allowedRoots:     ",;;",
			wantErrorMessage: "outside allowed roots",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__APP__ENV", "prod")
			t.Setenv("APP_CONFIG_ALLOWED_ROOTS", tc.allowedRoots)

			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configPath, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want allowed-root policy rejection")
			}
			if !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("error = %v, want ErrSecretPolicy", err)
			}
			if !strings.Contains(err.Error(), tc.wantErrorMessage) {
				t.Fatalf("error = %v, want %q", err, tc.wantErrorMessage)
			}
		})
	}
}

func TestNonLocalRejectsSymlinkPathComponents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	realRoot := t.TempDir()
	configPath := filepath.Join(realRoot, "config.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	linkedDir := filepath.Join(allowedRoot, "linked")
	if err := os.Symlink(realRoot, linkedDir); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	pathViaSymlink := filepath.Join(linkedDir, "config.yaml")
	_, _, err := LoadDetailed(LoadOptions{ConfigPath: pathViaSymlink})
	if err == nil {
		t.Fatalf("LoadDetailed() expected symlink-path rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}
