package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

const (
	keyDelimiter             = "."
	namespacePrefix          = "APP__"
	allowedConfigRootsEnvVar = "APP_CONFIG_ALLOWED_ROOTS"
	maxConfigFileSizeBytes   = int64(1 << 20)
)

type configFilePolicy uint8

const (
	configFilePolicyLocal configFilePolicy = iota
	configFilePolicyHardened
)

type loadMetadata struct {
	loadDefaultsDuration time.Duration
	loadFileDuration     time.Duration
	loadEnvDuration      time.Duration
	failedStage          string
	failedStageDuration  time.Duration
}

func loadKoanf(ctx context.Context, opts LoadOptions) (*koanf.Koanf, loadMetadata, error) {
	if err := checkContext(ctx); err != nil {
		return nil, loadMetadata{}, err
	}

	k := koanf.New(keyDelimiter)
	metadata := loadMetadata{}

	defaultsStarted := time.Now()
	if err := k.Load(confmap.Provider(defaultValues(), keyDelimiter), nil); err != nil {
		metadata.failedStage = StageLoadDefaults
		metadata.failedStageDuration = time.Since(defaultsStarted)
		return nil, metadata, fmt.Errorf("%w: load defaults: %w", ErrLoad, err)
	}
	metadata.loadDefaultsDuration = time.Since(defaultsStarted)
	if err := checkContext(ctx); err != nil {
		metadata.failedStage = StageLoadDefaults
		metadata.failedStageDuration = metadata.loadDefaultsDuration
		return nil, metadata, err
	}

	filePolicy := configFilePolicyHardened
	if isLocalEnvironmentHint(hasExplicitConfigFiles(opts)) {
		filePolicy = configFilePolicyLocal
	}
	filesStarted := time.Now()
	if strings.TrimSpace(opts.ConfigPath) != "" {
		if err := loadConfigFile(ctx, k, opts.ConfigPath, filePolicy); err != nil {
			metadata.failedStage = StageLoadFile
			metadata.failedStageDuration = time.Since(filesStarted)
			return nil, metadata, err
		}
	}
	for _, overlayPath := range opts.ConfigOverlays {
		if strings.TrimSpace(overlayPath) == "" {
			continue
		}
		if err := loadConfigFile(ctx, k, overlayPath, filePolicy); err != nil {
			metadata.failedStage = StageLoadFile
			metadata.failedStageDuration = time.Since(filesStarted)
			return nil, metadata, err
		}
	}
	metadata.loadFileDuration = time.Since(filesStarted)
	if err := checkContext(ctx); err != nil {
		metadata.failedStage = StageLoadFile
		metadata.failedStageDuration = metadata.loadFileDuration
		return nil, metadata, err
	}

	envStarted := time.Now()
	namespaceValues := collectNamespaceValues(os.Environ())
	if len(namespaceValues) > 0 {
		if err := k.Load(confmap.Provider(namespaceValues, keyDelimiter), nil); err != nil {
			metadata.failedStage = StageLoadEnv
			metadata.failedStageDuration = time.Since(envStarted)
			return nil, metadata, fmt.Errorf("%w: load namespace env: %w", ErrLoad, err)
		}
	}
	metadata.loadEnvDuration = time.Since(envStarted)
	if err := checkContext(ctx); err != nil {
		metadata.failedStage = StageLoadEnv
		metadata.failedStageDuration = metadata.loadEnvDuration
		return nil, metadata, err
	}

	return k, metadata, nil
}

func loadConfigFile(ctx context.Context, k *koanf.Koanf, path string, policy configFilePolicy) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return fmt.Errorf("%w: empty config path", ErrLoad)
	}
	cleanPath := filepath.Clean(trimmedPath)

	resolvedPath, pathInfo, err := enforceConfigFilePolicy(cleanPath, policy)
	if err != nil {
		return err
	}

	// #nosec G304 -- resolvedPath is normalized and validated by enforceConfigFilePolicy.
	fileHandle, err := os.Open(resolvedPath)
	if err != nil {
		return fmt.Errorf("%w: open config file %q: %w", ErrLoad, cleanPath, err)
	}
	defer func() {
		_ = fileHandle.Close()
	}()

	openedInfo, err := fileHandle.Stat()
	if err != nil {
		return fmt.Errorf("%w: stat opened config file %q: %w", ErrLoad, cleanPath, err)
	}
	if !os.SameFile(pathInfo, openedInfo) {
		return fmt.Errorf("%w: config file %q changed during policy checks", ErrSecretPolicy, cleanPath)
	}

	content, err := io.ReadAll(io.LimitReader(fileHandle, maxConfigFileSizeBytes+1))
	if err != nil {
		return fmt.Errorf("%w: read config file %q: %w", ErrLoad, cleanPath, err)
	}
	if int64(len(content)) > maxConfigFileSizeBytes {
		return fmt.Errorf("%w: config file %q exceeds max size limit %d bytes", ErrSecretPolicy, cleanPath, maxConfigFileSizeBytes)
	}

	fileConfig := koanf.New(keyDelimiter)
	if err := fileConfig.Load(rawbytes.Provider(content), yaml.Parser()); err != nil {
		return fmt.Errorf("%w: parse config file %q: %w", ErrParse, cleanPath, err)
	}
	if err := enforceSecretSourcePolicy(fileConfig, cleanPath); err != nil {
		return err
	}
	if err := k.Load(confmap.Provider(fileConfig.Raw(), keyDelimiter), nil); err != nil {
		return fmt.Errorf("%w: merge config file %q: %w", ErrLoad, cleanPath, err)
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	return nil
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
	if !fileInfo.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%w: config file %q must be a regular file", ErrSecretPolicy, path)
	}
	if fileInfo.Size() > maxConfigFileSizeBytes {
		return "", nil, fmt.Errorf("%w: config file %q exceeds max size limit %d bytes", ErrSecretPolicy, path, maxConfigFileSizeBytes)
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

	if policy == configFilePolicyLocal {
		return resolvedPath, resolvedInfo, nil
	}
	allowedRoots := resolveAllowedConfigRoots()
	if !isPathUnderAllowedRoots(resolvedPath, allowedRoots) {
		return "", nil, fmt.Errorf("%w: config file %q is outside allowed roots", ErrSecretPolicy, path)
	}
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return "", nil, fmt.Errorf("%w: symlink config file %q is not allowed outside local environment", ErrSecretPolicy, path)
	}
	if resolvedPath != absPath {
		return "", nil, fmt.Errorf("%w: symlink components in config file path %q are not allowed outside local environment", ErrSecretPolicy, path)
	}
	if resolvedInfo.Mode().Perm()&0o022 != 0 {
		return "", nil, fmt.Errorf("%w: config file %q must not be group/other writable", ErrSecretPolicy, path)
	}
	return resolvedPath, resolvedInfo, nil
}

func enforceSecretSourcePolicy(k *koanf.Koanf, path string) error {
	keys := k.Keys()
	sort.Strings(keys)
	for _, key := range keys {
		if !isSecretLikeConfigKey(key) {
			continue
		}
		if hasNonEmptyConfigValue(k.Get(key)) {
			return fmt.Errorf("%w: secret-like key %q is not allowed in config file %q", ErrSecretPolicy, key, path)
		}
	}
	return nil
}

func collectNamespaceValues(environ []string) map[string]any {
	values := make(map[string]any)

	for _, entry := range environ {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		envKey := parts[0]
		envValue := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(envKey, namespacePrefix) {
			continue
		}
		targetKey := namespaceEnvToKey(envKey)
		if targetKey == "" {
			continue
		}
		values[targetKey] = envValue
	}

	return values
}

func namespaceEnvToKey(envKey string) string {
	trimmed := strings.TrimPrefix(envKey, namespacePrefix)
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, "__")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			return ""
		}
		segments = append(segments, strings.ToLower(p))
	}
	return strings.Join(segments, keyDelimiter)
}

func namespaceEnvForConfigKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	return namespacePrefix + strings.ToUpper(strings.ReplaceAll(trimmed, keyDelimiter, "__"))
}

func isLocalEnvironmentHint(hasConfigFiles bool) bool {
	if value, ok := lookupNonEmptyEnv(namespaceEnvForConfigKey("app.env")); ok {
		return strings.EqualFold(value, "local")
	}
	if hasConfigFiles {
		// Fail closed for file-based configuration if environment intent is unknown.
		return false
	}
	return true
}

func hasExplicitConfigFiles(opts LoadOptions) bool {
	if strings.TrimSpace(opts.ConfigPath) != "" {
		return true
	}
	for _, overlay := range opts.ConfigOverlays {
		if strings.TrimSpace(overlay) != "" {
			return true
		}
	}
	return false
}

func resolveAllowedConfigRoots() []string {
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

func normalizeRoots(roots []string) []string {
	normalized := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))

	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		cleanRoot := filepath.Clean(trimmed)
		if !filepath.IsAbs(cleanRoot) {
			absRoot, err := filepath.Abs(cleanRoot)
			if err != nil {
				continue
			}
			cleanRoot = filepath.Clean(absRoot)
		}
		if _, exists := seen[cleanRoot]; exists {
			continue
		}
		seen[cleanRoot] = struct{}{}
		normalized = append(normalized, cleanRoot)
	}

	sort.Strings(normalized)
	return normalized
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

func isSecretLikeConfigKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if lower == "" {
		return false
	}

	switch lower {
	case "postgres.dsn", "mongo.uri", "redis.password", "observability.otel.exporter.otlp_headers":
		return true
	}

	segments := configKeySegments(lower)
	for i, segment := range segments {
		switch segment {
		case "password", "token", "secret", "authorization", "dsn":
			return true
		case "key":
			if i > 0 && (segments[i-1] == "api" || segments[i-1] == "private") {
				return true
			}
		case "headers":
			if i > 0 && segments[i-1] == "otlp" {
				return true
			}
		}
	}
	return false
}

func configKeySegments(key string) []string {
	return strings.FieldsFunc(key, func(r rune) bool {
		switch r {
		case '.', '_', '-':
			return true
		}
		return false
	})
}

func hasNonEmptyConfigValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value)) != ""
	}
}

func lookupNonEmptyEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}
