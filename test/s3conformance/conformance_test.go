//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	objectstore "github.com/example/go-service-template-rest/internal/infra/s3"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

const (
	conformancePartBytes   = int64(8 << 20)
	conformanceObjectBytes = int64(16 << 20)
)

var conformanceRunID = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// profile:object-storage:start
func TestS3ObjectStorageConformanceRequiresProviderCertification(t *testing.T) {
	if os.Getenv("REQUIRE_S3_CONFORMANCE") != "1" {
		t.Skip("provider conformance is not requested")
	}
	requireExactEnv(t, "S3_CONFORMANCE_MUTATION_AUTHORIZED", "1")
	requireExactEnv(t, "S3_CONFORMANCE_IDENTITY_POLICY_RECEIPT", "verified")
	requireExactEnv(t, "S3_CONFORMANCE_LIFECYCLE_RECEIPT", "verified")

	provider := objectstore.Provider(requiredEnv(t, "S3_CONFORMANCE_PROVIDER"))
	versioning := requiredEnv(t, "S3_CONFORMANCE_VERSIONING_RECEIPT")
	switch provider {
	case objectstore.ProviderAmazonS3:
		if versioning != "never_enabled" {
			t.Fatal("Amazon conformance requires never-enabled versioning evidence")
		}
	case objectstore.ProviderCloudflare:
		if versioning != "unsupported" {
			t.Fatal("R2 conformance requires its own unsupported-versioning evidence")
		}
	default:
		t.Fatal("provider conformance requires one exact provider selector")
	}

	runID := requiredEnv(t, "S3_CONFORMANCE_RUN_ID")
	if !conformanceRunID.MatchString(runID) {
		t.Fatal("S3_CONFORMANCE_RUN_ID must be a lowercase DNS label")
	}
	cfg := conformanceConfig(t, provider)
	primary := newConformanceClient(t, cfg, "S3_CONFORMANCE_")
	concealment := newConformanceClient(t, cfg, "S3_CONFORMANCE_CONCEALMENT_")

	prefix := "conformance/" + string(provider) + "/" + runID + "/"
	keys := []string{prefix + "single", prefix + "multipart", prefix + "cleanup"}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		for _, key := range keys {
			if err := primary.Delete(ctx, key); err != nil {
				t.Errorf("provider conformance cleanup delete failed: %v", err)
			}
		}
	})

	runConformance(t, primary, concealment, keys)
}

func conformanceConfig(t *testing.T, provider objectstore.Provider) objectstore.Config {
	t.Helper()
	region := requiredEnv(t, "S3_CONFORMANCE_REGION")
	endpoint := requiredEnv(t, "S3_CONFORMANCE_ENDPOINT")
	owner := os.Getenv("S3_CONFORMANCE_EXPECTED_BUCKET_OWNER")
	if provider == objectstore.ProviderAmazonS3 {
		if endpoint != "https://s3."+region+".amazonaws.com" {
			t.Fatal("Amazon conformance endpoint receipt is not the selected regional endpoint")
		}
		endpoint = ""
	}
	return objectstore.Config{
		Provider: provider, Endpoint: endpoint, Region: region,
		Bucket: requiredEnv(t, "S3_CONFORMANCE_BUCKET"), ExpectedBucketOwner: owner,
		CredentialSource: objectstore.CredentialSourceStatic, MaxObjectBytes: conformanceObjectBytes,
	}
}

func newConformanceClient(t *testing.T, cfg objectstore.Config, prefix string) *objectstore.Client {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", requiredEnv(t, prefix+"ACCESS_KEY_ID"))
	t.Setenv("AWS_SECRET_ACCESS_KEY", requiredEnv(t, prefix+"SECRET_ACCESS_KEY"))
	t.Setenv("AWS_SESSION_TOKEN", os.Getenv(prefix+"SESSION_TOKEN"))
	client, err := objectstore.New(t.Context(), cfg)
	if err != nil {
		t.Fatalf("construct provider conformance client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func runConformance(t *testing.T, primary, concealment *objectstore.Client, keys []string) {
	t.Helper()
	ctx := t.Context()
	if _, err := primary.Metadata(ctx, keys[0]+"-missing"); !errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("missing metadata error = %v, want ErrNotFound", err)
	}

	single := []byte("portable-object-storage-conformance")
	if err := primary.Upload(ctx, keys[0], bytes.NewReader(single), objectstorage.UploadOptions{
		Size: int64(len(single)), ContentType: "application/octet-stream", IfNotExists: true,
	}); err != nil {
		t.Fatalf("create-only upload error = %v", err)
	}
	if err := primary.Upload(ctx, keys[0], bytes.NewReader(single), objectstorage.UploadOptions{
		Size: int64(len(single)), IfNotExists: true,
	}); !errors.Is(err, objectstorage.ErrAlreadyExists) {
		t.Fatalf("create-only collision error = %v, want ErrAlreadyExists", err)
	}
	replacement := append([]byte(nil), single...)
	replacement[0] = 'P'
	if err := primary.Upload(ctx, keys[0], bytes.NewReader(replacement), objectstorage.UploadOptions{
		Size: int64(len(replacement)), ContentType: "application/octet-stream",
	}); err != nil {
		t.Fatalf("replacement upload error = %v", err)
	}
	assertObject(t, primary, keys[0], replacement, "application/octet-stream")
	if _, err := concealment.Metadata(ctx, keys[0]); err == nil || errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("concealment identity metadata error = %v, want opaque denial", err)
	}

	multipart := bytes.Repeat([]byte("m"), int(conformancePartBytes+257))
	if err := primary.Upload(ctx, keys[1], bytes.NewReader(multipart), objectstorage.UploadOptions{Size: int64(len(multipart))}); err != nil {
		t.Fatalf("multipart upload error = %v", err)
	}
	assertObject(t, primary, keys[1], multipart, "")
	if err := primary.Upload(ctx, keys[2], bytes.NewReader(multipart[:conformancePartBytes]), objectstorage.UploadOptions{Size: conformancePartBytes + 1}); err == nil {
		t.Fatal("short multipart source unexpectedly succeeded")
	}

	presigned, err := primary.PresignGet(ctx, keys[1], time.Minute)
	if err != nil {
		t.Fatalf("PresignGet() error = %v", err)
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if got := executePresigned(t, client, http.MethodGet, presigned, http.StatusOK); !bytes.Equal(got, multipart) {
		t.Fatal("presigned GET bytes changed")
	}
	mutated, parseErr := url.Parse(presigned)
	if parseErr != nil {
		t.Fatal("parse presigned URL")
	}
	query := mutated.Query()
	query.Set("X-Amz-Signature", strings.Repeat("0", 64))
	mutated.RawQuery = query.Encode()
	executePresigned(t, client, http.MethodGet, mutated.String(), http.StatusForbidden)

	if err := primary.Delete(ctx, keys[0]); err != nil {
		t.Fatalf("delete existing error = %v", err)
	}
	if _, err := primary.Metadata(ctx, keys[0]); !errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("metadata after delete error = %v, want ErrNotFound", err)
	}
	if err := primary.Delete(ctx, keys[0]); err != nil {
		t.Fatalf("delete absent error = %v", err)
	}
}

func assertObject(t *testing.T, client *objectstore.Client, key string, want []byte, contentType string) {
	t.Helper()
	metadata, err := client.Metadata(t.Context(), key)
	if err != nil || metadata.Size != int64(len(want)) || metadata.LastModified.IsZero() ||
		contentType != "" && metadata.ContentType != contentType {
		t.Fatalf("Metadata() = %#v, %v", metadata, err)
	}
	object, err := client.Download(t.Context(), key)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	got, readErr := io.ReadAll(object.Body)
	closeErr := object.Body.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(got, want) || object.Size != int64(len(want)) {
		t.Fatalf("Download() = %d bytes, size %d, read %v, close %v", len(got), object.Size, readErr, closeErr)
	}
}

func executePresigned(t *testing.T, client *http.Client, method, target string, wantStatus int) []byte {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, target, http.NoBody)
	if err != nil {
		t.Fatal("construct presigned request")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal("execute presigned request")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != wantStatus {
		t.Fatalf("presigned status = %d, want %d", response.StatusCode, wantStatus)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, conformanceObjectBytes+1))
	if err != nil || int64(len(data)) > conformanceObjectBytes {
		t.Fatal("read presigned response")
	}
	return data
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func requireExactEnv(t *testing.T, name, expected string) {
	t.Helper()
	if os.Getenv(name) != expected {
		t.Fatalf("%s must be %q", name, expected)
	}
}

// profile:object-storage:end
