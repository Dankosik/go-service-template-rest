package bootstrap

import (
	"fmt"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/s3"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

type objectStorageRuntime interface {
	objectstorage.Store
	Close()
}

var _ objectStorageRuntime = (*s3.Client)(nil)

// initObjectStorage validates and constructs the selected local adapter. It
// intentionally does not probe the provider or contribute a readiness probe.
func initObjectStorage(cfg config.ObjectStorageConfig) (objectStorageRuntime, error) { //nolint:ireturn // runtimeWiring needs the narrow test seam.
	return initObjectStorageWith(cfg, func(cfg s3.Config) (objectStorageRuntime, error) { return s3.New(cfg) })
}

//nolint:ireturn // The injected constructor returns the lifecycle test fake.
func initObjectStorageWith(
	cfg config.ObjectStorageConfig,
	build func(s3.Config) (objectStorageRuntime, error),
) (objectStorageRuntime, error) {
	runtime, err := build(s3.Config{
		Provider:                s3.Provider(cfg.Provider),
		Endpoint:                cfg.Endpoint,
		Region:                  cfg.Region,
		Bucket:                  cfg.Bucket,
		AccessKeyID:             cfg.AccessKeyID,
		SecretAccessKey:         cfg.SecretAccessKey,
		SessionToken:            cfg.SessionToken,
		MaxObjectBytes:          cfg.MaxObjectBytes,
		MultipartChunkBytes:     cfg.MultipartChunkBytes,
		MaxActiveOperations:     cfg.MaxActiveOperations,
		MaxOperationDuration:    cfg.MaxOperationDuration,
		MaxPresignLifetime:      cfg.MaxPresignLifetime,
		MaxResponseHeaderBytes:  cfg.MaxResponseHeaderBytes,
		MaxControlResponseBytes: cfg.MaxControlResponseBytes,
		MaxWorkingMemoryBytes:   cfg.MaxWorkingMemoryBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("build object storage runtime: %w", err)
	}
	return runtime, nil
}
