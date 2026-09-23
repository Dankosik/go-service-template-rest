package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

// maxConfigFileSizeBytes bounds what is read into memory. The path comes from
// this process's own arguments, but the file behind it does not have to be what
// whoever wrote the manifest expected.
//
// There is deliberately no path allowlist, symlink refusal, or permission check
// here. The path arrives on argv, chosen by whoever chose the binary and the
// entrypoint, so a policy at this boundary defends against an attacker who could
// already run a different program — while breaking the only two ways Kubernetes
// supplies a config file, both of which are symlinks into a timestamped
// directory.
const maxConfigFileSizeBytes = int64(1 << 20)

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

// mergeConfigFile reads one YAML file, applies the secret-source policy, and
// merges its values into k. It returns the section-scalar override keys it
// dropped.
func mergeConfigFile(ctx context.Context, k *koanf.Koanf, path string) ([]string, error) {
	if err := checkLoadContext(ctx); err != nil {
		return nil, err
	}

	cleanPath, content, err := readConfigFile(path)
	if err != nil {
		return nil, err
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
	if err := checkLoadContext(ctx); err != nil {
		return nil, err
	}
	return sectionScalarOverrideKeys, nil
}

// readConfigFile reads at most maxConfigFileSizeBytes from path. validateLoadOptions
// has already rejected blank paths.
func readConfigFile(path string) (cleanPath string, content []byte, err error) {
	cleanPath = filepath.Clean(strings.TrimSpace(path))

	// #nosec G304 -- the path is this process's own -config argument, at the same
	// trust level as the binary and the entrypoint that supplied it.
	fileHandle, err := os.Open(cleanPath)
	if err != nil {
		return cleanPath, nil, fmt.Errorf("%w: open config file %q: %w", ErrLoad, cleanPath, err)
	}
	defer func() {
		_ = fileHandle.Close()
	}()

	fileInfo, err := fileHandle.Stat()
	if err != nil {
		return cleanPath, nil, fmt.Errorf("%w: stat config file %q: %w", ErrLoad, cleanPath, err)
	}
	if fileInfo.IsDir() {
		return cleanPath, nil, fmt.Errorf("%w: config file %q is a directory", ErrLoad, cleanPath)
	}

	content, err = io.ReadAll(io.LimitReader(fileHandle, maxConfigFileSizeBytes+1))
	if err != nil {
		return cleanPath, nil, fmt.Errorf("%w: read config file %q: %w", ErrLoad, cleanPath, err)
	}
	if int64(len(content)) > maxConfigFileSizeBytes {
		return cleanPath, nil, fmt.Errorf("%w: config file %q exceeds max size limit %d bytes", ErrLoad, cleanPath, maxConfigFileSizeBytes)
	}
	return cleanPath, content, nil
}
