package bootstrap

import (
	"context"
	"errors"

	// profile:authn-bearer:start
	"log/slog"
	// profile:authn-bearer:end
	"slices"
	"testing"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/s3"

	// profile:authn-bearer:start
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	// profile:authn-bearer:end
)

func TestInitObjectStorageMapsConfig(t *testing.T) {
	want := validObjectStorageConfig()
	built := &countingObjectStorageRuntime{}
	runtime, err := initObjectStorageWith(t.Context(), want, func(ctx context.Context, got s3.Config) (objectStorageRuntime, error) {
		if ctx != t.Context() || got.Provider != s3.ProviderAmazonS3 || got.Endpoint != want.Endpoint ||
			got.Region != want.Region || got.Bucket != want.Bucket || got.ExpectedBucketOwner != want.ExpectedBucketOwner ||
			string(got.CredentialSource) != want.CredentialSource || got.MaxObjectBytes != want.MaxObjectBytes {
			t.Fatalf("mapped S3 config = %#v", got)
		}
		return built, nil
	})
	if err != nil {
		t.Fatalf("initObjectStorageWith() error = %v", err)
	}
	runtime.Close()
	if built.calls != 1 {
		t.Fatalf("runtime Close calls = %d, want 1", built.calls)
	}
}

func TestObjectStorageRuntimeClosesAfterServing(t *testing.T) {
	resetShutdownConfigEnv(t)
	var events []string
	closed := &countingObjectStorageRuntime{onClose: func() { events = append(events, "closed") }}
	stopServing := errors.New("stop serving")
	wiring := objectStorageTestWiring(closed)
	wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error {
		events = append(events, "serve")
		return stopServing
	}

	if err := runWithRuntime(nil, wiring); !errors.Is(err, stopServing) {
		t.Fatalf("runWithRuntime() error = %v, want %v", err, stopServing)
	}
	if closed.calls != 1 {
		t.Fatalf("object storage Close calls = %d, want 1", closed.calls)
	}
	want := []string{"serve", "closed"}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func objectStorageTestWiring(runtime objectStorageRuntime) runtimeWiring {
	wiring := testRuntimeWiring()
	wiring.initObjectStorage = func(context.Context, config.ObjectStorageConfig) (objectStorageRuntime, error) { return runtime, nil }
	wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
		return runtimeDependencies{}, nil
	}
	// profile:authn-bearer:start
	wiring.initAuthn = func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error) {
		return fakeAuthnRuntime{}, nil
	}
	// profile:authn-bearer:end
	return wiring
}

type countingObjectStorageRuntime struct {
	calls   int
	onClose func()
}

func (r *countingObjectStorageRuntime) Close() {
	r.calls++
	if r.onClose != nil {
		r.onClose()
	}
}

func validObjectStorageConfig() config.ObjectStorageConfig {
	return config.ObjectStorageConfig{
		Provider: "amazon_s3", Region: "us-east-1", Bucket: "examplebucket",
		ExpectedBucketOwner: "123456789012", CredentialSource: "aws_default",
		MaxObjectBytes: 10 << 20,
	}
}
