package bootstrap

import (
	"context"
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

// initObjectStorage constructs the selected adapter without probing its provider.
func initObjectStorage(ctx context.Context, cfg config.ObjectStorageConfig) (objectStorageRuntime, error) { //nolint:ireturn // runtimeWiring needs the lifecycle seam.
	return initObjectStorageWith(ctx, cfg, func(ctx context.Context, cfg s3.Config) (objectStorageRuntime, error) {
		return s3.New(ctx, cfg)
	})
}

//nolint:ireturn // The injected constructor returns the lifecycle test fake.
func initObjectStorageWith(
	ctx context.Context,
	cfg config.ObjectStorageConfig,
	build func(context.Context, s3.Config) (objectStorageRuntime, error),
) (objectStorageRuntime, error) {
	runtime, err := build(ctx, s3.Config{
		Provider: s3.Provider(cfg.Provider), Endpoint: cfg.Endpoint, Region: cfg.Region,
		Bucket: cfg.Bucket, ExpectedBucketOwner: cfg.ExpectedBucketOwner,
		CredentialSource: cfg.CredentialSource, MaxObjectBytes: cfg.MaxObjectBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("build object storage runtime: %w", err)
	}
	return runtime, nil
}
