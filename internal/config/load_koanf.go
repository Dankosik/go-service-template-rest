package config

import (
	"context"
	"fmt"
	"os"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

const keyDelimiter = "."

type loadMetadata struct {
	sectionScalarOverrideKeys []string
	malformedEnvironmentKeys  []string
	failedStage               string
}

// loadKoanf merges defaults, config files, and the environment. The caller
// checks ctx first; every error return sets metadata.failedStage.
func loadKoanf(ctx context.Context, opts LoadOptions) (*koanf.Koanf, loadMetadata, error) {
	k := koanf.New(keyDelimiter)
	metadata := loadMetadata{}

	if err := k.Load(confmap.Provider(defaultValues(), keyDelimiter), nil); err != nil {
		metadata.failedStage = StageLoadDefaults
		return nil, metadata, fmt.Errorf("%w: load defaults: %w", ErrLoad, err)
	}
	if err := checkLoadContext(ctx); err != nil {
		metadata.failedStage = StageLoadDefaults
		return nil, metadata, err
	}

	if err := validateLoadOptions(opts); err != nil {
		metadata.failedStage = StageLoadFile
		return nil, metadata, err
	}
	paths := opts.ConfigOverlays
	if opts.ConfigPath != "" {
		paths = append([]string{opts.ConfigPath}, opts.ConfigOverlays...)
	}
	for _, path := range paths {
		sectionScalarOverrideKeys, err := mergeConfigFile(ctx, k, path)
		if err != nil {
			metadata.failedStage = StageLoadFile
			return nil, metadata, err
		}
		metadata.sectionScalarOverrideKeys = append(metadata.sectionScalarOverrideKeys, sectionScalarOverrideKeys...)
	}
	if err := checkLoadContext(ctx); err != nil {
		metadata.failedStage = StageLoadFile
		return nil, metadata, err
	}

	envValues, malformedEnvironmentKeys := collectEnvironmentValues(os.Environ())
	metadata.malformedEnvironmentKeys = malformedEnvironmentKeys
	metadata.sectionScalarOverrideKeys = append(metadata.sectionScalarOverrideKeys, removeSectionScalarOverridesInPlace(envValues)...)
	if len(envValues) > 0 {
		if err := k.Load(confmap.Provider(envValues, keyDelimiter), nil); err != nil {
			metadata.failedStage = StageLoadEnv
			return nil, metadata, fmt.Errorf("%w: load namespace env: %w", ErrLoad, err)
		}
	}
	if err := checkLoadContext(ctx); err != nil {
		metadata.failedStage = StageLoadEnv
		return nil, metadata, err
	}

	return k, metadata, nil
}
