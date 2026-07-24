package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	allowedConfigRootsEnvVar = "APP_CONFIG_ALLOWED_ROOTS"
	maxConfigFileSizeBytes   = int64(1 << 20)
)

type configFilePolicy uint8

const (
	configFilePolicyLocal configFilePolicy = iota
	configFilePolicyHardened
)

func validateLoadOptions(opts LoadOptions) error {
	if opts.ConfigPath != "" && strings.TrimSpace(opts.ConfigPath) == "" {
		return fmt.Errorf("%w: empty config path", ErrLoad)
	}
	for index, overlayPath := range opts.ConfigOverlays {
		if strings.TrimSpace(overlayPath) == "" {
			return fmt.Errorf("%w: config overlay path at index %d is empty", ErrLoad, index)
		}
	}
	return nil
}

func loadConfigFileWithMetadata(ctx context.Context, k *koanf.Koanf, path string, policy configFilePolicy) ([]string, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("%w: empty config path", ErrLoad)
	}
	cleanPath := filepath.Clean(trimmedPath)

	resolvedPath, pathInfo, err := enforceConfigFilePolicy(cleanPath, policy)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- resolvedPath is normalized and validated by enforceConfigFilePolicy.
	fileHandle, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("%w: open config file %q: %w", ErrLoad, cleanPath, err)
	}
	defer func() {
		_ = fileHandle.Close()
	}()

	openedInfo, err := fileHandle.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: stat opened config file %q: %w", ErrLoad, cleanPath, err)
	}
	if !os.SameFile(pathInfo, openedInfo) {
		return nil, fmt.Errorf("%w: config file %q changed during policy checks", ErrSecretPolicy, cleanPath)
	}

	content, err := io.ReadAll(io.LimitReader(fileHandle, maxConfigFileSizeBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read config file %q: %w", ErrLoad, cleanPath, err)
	}
	if int64(len(content)) > maxConfigFileSizeBytes {
		return nil, fmt.Errorf("%w: config file %q exceeds max size limit %d bytes", ErrSecretPolicy, cleanPath, maxConfigFileSizeBytes)
	}

	fileConfig := koanf.New(keyDelimiter)
	if err := fileConfig.Load(rawbytes.Provider(content), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("%w: parse config file %q: %w", ErrParse, cleanPath, err)
	}
	if err := enforceSecretSourcePolicy(fileConfig, cleanPath); err != nil {
		return nil, err
	}
	fileValues := fileConfig.Raw()
	sectionScalarOverrideKeys := removeSectionScalarOverridesInPlace(fileValues)
	if len(fileValues) > 0 {
		if err := k.Load(confmap.Provider(fileValues, keyDelimiter), nil); err != nil {
			return nil, fmt.Errorf("%w: merge config file %q: %w", ErrLoad, cleanPath, err)
		}
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	return sectionScalarOverrideKeys, nil
}

func enforceConfigFilePolicy(path string, policy configFilePolicy) (string, os.FileInfo, error) {
	if policy != configFilePolicyLocal && !filepath.IsAbs(path) {
		return "", nil, fmt.Errorf("%w: config file path %q must be absolute outside local environment", ErrSecretPolicy, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, fmt.Errorf("%w: resolve config file %q: %w", ErrLoad, path, err)
	}
	absPath = filepath.Clean(absPath)

	fileInfo, err := os.Lstat(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("%w: stat config file %q: %w", ErrLoad, path, err)
	}
	if fileInfo.IsDir() {
		return "", nil, fmt.Errorf("%w: config file %q is a directory", ErrLoad, path)
	}
	if policy != configFilePolicyLocal && fileInfo.Mode()&os.ModeSymlink != 0 {
		return "", nil, fmt.Errorf("%w: symlink config file %q is not allowed outside local environment", ErrSecretPolicy, path)
	}
	if policy != configFilePolicyLocal && !fileInfo.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%w: config file %q must be a regular file", ErrSecretPolicy, path)
	}

	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("%w: resolve config file %q: %w", ErrSecretPolicy, path, err)
	}
	resolvedPath = filepath.Clean(resolvedPath)
	resolvedInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return "", nil, fmt.Errorf("%w: stat resolved config file %q: %w", ErrSecretPolicy, path, err)
	}
	if resolvedInfo.IsDir() {
		return "", nil, fmt.Errorf("%w: config file %q is a directory", ErrLoad, path)
	}
	if !resolvedInfo.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%w: config file %q must be a regular file", ErrSecretPolicy, path)
	}
	if resolvedInfo.Size() > maxConfigFileSizeBytes {
		return "", nil, fmt.Errorf("%w: config file %q exceeds max size limit %d bytes", ErrSecretPolicy, path, maxConfigFileSizeBytes)
	}

	if policy == configFilePolicyLocal {
		return resolvedPath, resolvedInfo, nil
	}
	allowedRoots, err := resolveAllowedConfigRoots()
	if err != nil {
		return "", nil, err
	}
	if !isPathUnderAllowedRoots(resolvedPath, allowedRoots) {
		return "", nil, fmt.Errorf("%w: config file %q is outside allowed roots", ErrSecretPolicy, path)
	}
	if resolvedPath != absPath {
		return "", nil, fmt.Errorf("%w: symlink components in config file path %q are not allowed outside local environment", ErrSecretPolicy, path)
	}
	if resolvedInfo.Mode().Perm()&0o022 != 0 {
		return "", nil, fmt.Errorf("%w: config file %q must not be group/other writable", ErrSecretPolicy, path)
	}
	return resolvedPath, resolvedInfo, nil
}

func configFilePolicyForLoad(hasConfigFiles bool) configFilePolicy {
	if value, ok := lookupNonEmptyEnv(namespaceEnvForConfigKey("app.env")); ok {
		if strings.EqualFold(value, "local") {
			return configFilePolicyLocal
		}
		return configFilePolicyHardened
	}
	if hasConfigFiles {
		// Fail closed for file-based configuration if environment intent is unknown.
		return configFilePolicyHardened
	}
	return configFilePolicyLocal
}

func hasExplicitConfigFiles(opts LoadOptions) bool {
	return opts.ConfigPath != "" || len(opts.ConfigOverlays) > 0
}

func resolveAllowedConfigRoots() ([]string, error) {
	rootsValue, hasRoots := lookupNonEmptyEnv(allowedConfigRootsEnvVar)
	if !hasRoots {
		defaultRoots := []string{
			"/etc/config",
			"/etc/service/config",
			"/run/secrets",
		}
		return normalizeRoots(defaultRoots)
	}

	parts := strings.FieldsFunc(rootsValue, func(r rune) bool {
		return r == ',' || r == ';' || r == os.PathListSeparator
	})
	return normalizeRoots(parts)
}

func normalizeRoots(roots []string) ([]string, error) {
	normalized := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))

	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cleanRoot := filepath.Clean(trimmed)
		if !filepath.IsAbs(cleanRoot) {
			return nil, fmt.Errorf("%w: %s entries must be absolute paths", ErrSecretPolicy, allowedConfigRootsEnvVar)
		}
		if _, exists := seen[cleanRoot]; exists {
			continue
		}
		seen[cleanRoot] = struct{}{}
		normalized = append(normalized, cleanRoot)
	}

	sort.Strings(normalized)
	return normalized, nil
}

func isPathUnderAllowedRoots(path string, roots []string) bool {
	cleanPath := filepath.Clean(path)
	for _, root := range roots {
		rel, err := filepath.Rel(root, cleanPath)
		if err != nil {
			continue
		}
		if rel == "." {
			return true
		}
		if rel == ".." {
			continue
		}
		if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		return true
	}
	return false
}
