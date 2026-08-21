package bootstrap

import (
	"context"
	"errors"
	"io"

	// profile:authn-oidc-jwt:start
	"log/slog"
	// profile:authn-oidc-jwt:end
	"slices"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/s3"

	// profile:authn-oidc-jwt:start
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	// profile:authn-oidc-jwt:end
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestInitObjectStorageMapsConfig(t *testing.T) {
	want := validObjectStorageConfig()
	built := &countingObjectStorageRuntime{}
	runtime, err := initObjectStorageWith(t.Context(), want, func(ctx context.Context, got s3.Config) (objectStorageRuntime, error) {
		if ctx != t.Context() || got.Provider != s3.ProviderAmazonS3 || got.Bucket != want.Bucket ||
			got.CredentialSource != want.CredentialSource || got.MaxObjectBytes != want.MaxObjectBytes {
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

func TestObjectStorageConstructionFollowsMemoryPublication(t *testing.T) {
	resetShutdownConfigEnv(t)
	stopServing := errors.New("stop serving")
	wiring := objectStorageTestWiring(&countingObjectStorageRuntime{})
	var events []runtimeLifecycleStage
	wiring.lifecycle = func(stage runtimeLifecycleStage) { events = append(events, stage) }
	wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error { return stopServing }

	if err := runWithRuntime(nil, wiring); !errors.Is(err, stopServing) {
		t.Fatalf("runWithRuntime() error = %v, want %v", err, stopServing)
	}
	if len(events) < 2 || events[0] != runtimeLifecycleMemoryPublished || events[1] != runtimeLifecycleObjectStorageConstructed {
		t.Fatalf("startup lifecycle events = %v", events)
	}
}

func TestObjectStorageOutageDoesNotChangeReadiness(t *testing.T) {
	resetShutdownConfigEnv(t)
	runtime := &scriptedObjectStorageRuntime{outcomes: []error{errors.New("provider unavailable"), nil}}
	stopServing := errors.New("stop serving")
	wiring := objectStorageTestWiring(runtime)
	wiring.serve = func(ctx context.Context, _ context.Context, args serveRuntimeArgs) error {
		if err := args.readinessCheck(ctx); err != nil {
			t.Fatalf("initial readiness check error = %v", err)
		}
		args.admission.MarkReady()
		if _, err := runtime.Metadata(ctx, "object"); err == nil {
			t.Fatal("first storage operation error = nil, want provider outage")
		}
		if err := args.healthSvc.Cached(); err != nil {
			t.Fatalf("readiness after provider outage = %v, want unchanged", err)
		}
		return stopServing
	}
	if err := runWithRuntime(nil, wiring); !errors.Is(err, stopServing) {
		t.Fatalf("runWithRuntime() error = %v, want %v", err, stopServing)
	}
}

func TestObjectStorageRuntimeCloseOrder(t *testing.T) {
	resetShutdownConfigEnv(t)
	closed := &countingObjectStorageRuntime{}
	stopServing := errors.New("stop serving")
	wiring := objectStorageTestWiring(closed)
	var events []runtimeLifecycleStage
	wiring.lifecycle = func(stage runtimeLifecycleStage) { events = append(events, stage) }
	wiring.serve = func(context.Context, context.Context, serveRuntimeArgs) error { return stopServing }

	if err := runWithRuntime(nil, wiring); !errors.Is(err, stopServing) {
		t.Fatalf("runWithRuntime() error = %v, want %v", err, stopServing)
	}
	if closed.calls != 1 {
		t.Fatalf("object storage Close calls = %d, want 1", closed.calls)
	}
	want := []runtimeLifecycleStage{
		runtimeLifecycleMemoryPublished, runtimeLifecycleObjectStorageConstructed,
		runtimeLifecycleHTTPDrained, runtimeLifecycleBackgroundJoined,
		runtimeLifecycleObjectStorageClosed, runtimeLifecycleDependenciesClosed,
		runtimeLifecycleTelemetryFlushed,
	}
	if !slices.Equal(events, want) {
		t.Fatalf("startup lifecycle events = %v, want %v", events, want)
	}
}

func objectStorageTestWiring(runtime objectStorageRuntime) runtimeWiring {
	wiring := testRuntimeWiring()
	wiring.initObjectStorage = func(context.Context, config.ObjectStorageConfig) (objectStorageRuntime, error) { return runtime, nil }
	wiring.dependencies = func(context.Context, startupBootstrap) (runtimeDependencies, error) {
		return runtimeDependencies{}, nil
	}
	// profile:authn-oidc-jwt:start
	wiring.initAuthn = func(context.Context, config.Config, *telemetry.Metrics, *slog.Logger) (authnRuntime, error) {
		return fakeAuthnRuntime{}, nil
	}
	// profile:authn-oidc-jwt:end
	return wiring
}

type countingObjectStorageRuntime struct {
	noOpObjectStorageStore

	calls int
}

func (r *countingObjectStorageRuntime) Close() { r.calls++ }

type scriptedObjectStorageRuntime struct {
	countingObjectStorageRuntime

	outcomes []error
}

func (r *scriptedObjectStorageRuntime) Metadata(context.Context, string) (objectstorage.Metadata, error) {
	if len(r.outcomes) == 0 {
		return objectstorage.Metadata{}, nil
	}
	err := r.outcomes[0]
	r.outcomes = r.outcomes[1:]
	return objectstorage.Metadata{}, err
}

type noOpObjectStorageStore struct{}

func (noOpObjectStorageStore) Upload(context.Context, string, io.Reader, objectstorage.UploadOptions) error {
	return nil
}

func (noOpObjectStorageStore) Download(context.Context, string) (objectstorage.Object, error) {
	return objectstorage.Object{}, nil
}

func (noOpObjectStorageStore) Metadata(context.Context, string) (objectstorage.Metadata, error) {
	return objectstorage.Metadata{}, nil
}
func (noOpObjectStorageStore) Delete(context.Context, string) error { return nil }
func (noOpObjectStorageStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func validObjectStorageConfig() config.ObjectStorageConfig {
	return config.ObjectStorageConfig{
		Provider: "amazon_s3", Region: "us-east-1", Bucket: "examplebucket",
		ExpectedBucketOwner: "123456789012", CredentialSource: "aws_default",
		MaxObjectBytes: 10 << 20,
	}
}
