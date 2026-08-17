//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	boundedhttp "github.com/example/go-service-template-rest/internal/infra/httpclient"
	objectstore "github.com/example/go-service-template-rest/internal/infra/s3"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

const (
	conformanceRootBundle  = "/etc/ssl/certs/ca-certificates.crt"
	conformanceChunkBytes  = int64(5 << 20)
	conformanceObjectBytes = int64(16 << 20)
)

var (
	conformanceRunID = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	imageDigest      = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// profile:object-storage:start
func TestS3ObjectStorageConformanceRequiresProviderCertification(t *testing.T) {
	t.Parallel()
	if os.Getenv("REQUIRE_S3_CONFORMANCE") != "1" {
		t.Skip("provider conformance is not requested")
	}

	env := loadS3ConformanceEnv(t)
	roots := verifyS3ConformanceRootBundle(t, env)
	primary := newS3ConformanceClient(t, env.primary)
	concealment := newS3ConformanceClient(t, env.concealment)

	prefix := "conformance/" + string(env.primary.Provider) + "/" + env.runID + "/"
	keys := []string{prefix + "single", prefix + "multipart", prefix + "cleanup"}
	registerS3ConformanceCleanup(t, primary, env.primary, roots, prefix, keys)
	runS3Conformance(t, primary, concealment, roots, keys)
}

type s3ConformanceEnv struct {
	primary       objectstore.Config
	concealment   objectstore.Config
	runID         string
	imagePlatform string
	bundleSHA256  string
	bundleBytes   int64
	bundleRoots   int
}

func loadS3ConformanceEnv(t *testing.T) s3ConformanceEnv {
	t.Helper()
	requireExactEnv(t, "S3_CONFORMANCE_MUTATION_AUTHORIZED", "1")
	requireExactEnv(t, "S3_CONFORMANCE_IDENTITY_POLICY_RECEIPT", "verified")
	requireExactEnv(t, "S3_CONFORMANCE_LIFECYCLE_RECEIPT", "verified")
	requireExactEnv(t, "S3_CONFORMANCE_NO_ROOT_OVERRIDE", "1")
	if os.Getenv("SSL_CERT_FILE") != "" || os.Getenv("SSL_CERT_DIR") != "" {
		t.Fatal("provider conformance forbids ambient root overrides")
	}

	provider := objectstore.Provider(os.Getenv("S3_CONFORMANCE_PROVIDER"))
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
	platform := requiredEnv(t, "S3_CONFORMANCE_IMAGE_PLATFORM")
	if platform != runtime.GOOS+"/"+runtime.GOARCH || platform != "linux/amd64" && platform != "linux/arm64" {
		t.Fatal("provider conformance must run on its attested Linux image architecture")
	}
	if !imageDigest.MatchString(requiredEnv(t, "S3_CONFORMANCE_IMAGE_DIGEST")) {
		t.Fatal("S3_CONFORMANCE_IMAGE_DIGEST must be an immutable sha256 digest")
	}
	bundleBytes := requiredPositiveInt64(t, "S3_CONFORMANCE_ROOT_BUNDLE_BYTES")
	bundleRoots64 := requiredPositiveInt64(t, "S3_CONFORMANCE_ROOT_BUNDLE_ROOTS")
	if bundleRoots64 > int64(^uint(0)>>1) {
		t.Fatal("S3_CONFORMANCE_ROOT_BUNDLE_ROOTS exceeds the current architecture")
	}
	bundleSHA256 := requiredEnv(t, "S3_CONFORMANCE_ROOT_BUNDLE_SHA256")
	if decoded, err := hex.DecodeString(bundleSHA256); err != nil || len(decoded) != sha256.Size {
		t.Fatal("S3_CONFORMANCE_ROOT_BUNDLE_SHA256 must be one SHA-256 value")
	}

	primary := s3ConformanceConfig(t, provider, "S3_CONFORMANCE_")
	concealment := primary
	concealment.AccessKeyID = requiredEnv(t, "S3_CONFORMANCE_CONCEALMENT_ACCESS_KEY_ID")
	concealment.SecretAccessKey = requiredEnv(t, "S3_CONFORMANCE_CONCEALMENT_SECRET_ACCESS_KEY")
	concealment.SessionToken = os.Getenv("S3_CONFORMANCE_CONCEALMENT_SESSION_TOKEN")
	if provider == objectstore.ProviderAmazonS3 && concealment.SessionToken == "" {
		t.Fatal("S3_CONFORMANCE_CONCEALMENT_SESSION_TOKEN is required for Amazon conformance")
	}

	return s3ConformanceEnv{
		primary: primary, concealment: concealment, runID: runID, imagePlatform: platform,
		bundleSHA256: bundleSHA256, bundleBytes: bundleBytes, bundleRoots: int(bundleRoots64),
	}
}

func s3ConformanceConfig(t *testing.T, provider objectstore.Provider, prefix string) objectstore.Config {
	t.Helper()
	cfg := objectstore.Config{
		Provider:                provider,
		Endpoint:                requiredEnv(t, prefix+"ENDPOINT"),
		Region:                  requiredEnv(t, prefix+"REGION"),
		Bucket:                  requiredEnv(t, prefix+"BUCKET"),
		AccessKeyID:             requiredEnv(t, prefix+"ACCESS_KEY_ID"),
		SecretAccessKey:         requiredEnv(t, prefix+"SECRET_ACCESS_KEY"),
		SessionToken:            os.Getenv(prefix + "SESSION_TOKEN"),
		ExpectedBucketOwner:     os.Getenv(prefix + "EXPECTED_BUCKET_OWNER"),
		MaxObjectBytes:          conformanceObjectBytes,
		MultipartChunkBytes:     conformanceChunkBytes,
		MaxActiveOperations:     2,
		MaxOperationDuration:    30 * time.Second,
		MaxPresignLifetime:      5 * time.Minute,
		MaxResponseHeaderBytes:  64 << 10,
		MaxControlResponseBytes: 1 << 20,
		MaxWorkingMemoryBytes:   512 << 20,
	}
	if provider == objectstore.ProviderAmazonS3 && cfg.SessionToken == "" {
		t.Fatal(prefix + "SESSION_TOKEN is required for Amazon conformance")
	}
	return cfg
}

func verifyS3ConformanceRootBundle(t *testing.T, env s3ConformanceEnv) *x509.CertPool {
	t.Helper()
	info, err := os.Stat(conformanceRootBundle)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o555 {
		t.Fatal("provider conformance requires the regular read-only production root bundle")
	}
	data, err := os.ReadFile(conformanceRootBundle)
	if err != nil {
		t.Fatal("provider conformance could not read the production root bundle")
	}
	identity := sha256.Sum256(data)
	if int64(len(data)) != env.bundleBytes || hex.EncodeToString(identity[:]) != env.bundleSHA256 || len(data) > (448<<10)/2 {
		t.Fatal("provider conformance root-bundle byte identity or headroom differs from its receipt")
	}

	pool := x509.NewCertPool()
	seen := make(map[[sha256.Size]byte]struct{})
	rest := data
	for len(bytes.TrimSpace(rest)) > 0 {
		rest = bytes.TrimLeft(rest, " \t\r\n")
		block, remaining := pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			t.Fatal("provider conformance root bundle contains non-certificate data")
		}
		certificate, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil || !certificate.BasicConstraintsValid || !certificate.IsCA {
			t.Fatal("provider conformance root bundle contains an invalid CA")
		}
		certificateID := sha256.Sum256(certificate.Raw)
		if _, duplicate := seen[certificateID]; duplicate {
			t.Fatal("provider conformance root bundle contains a duplicate CA")
		}
		seen[certificateID] = struct{}{}
		pool.AddCert(certificate)
		rest = remaining
	}
	if len(seen) != env.bundleRoots || len(seen) > 288/2 {
		t.Fatal("provider conformance root count or headroom differs from its receipt")
	}
	return pool
}

func newS3ConformanceClient(t *testing.T, cfg objectstore.Config) *objectstore.Client {
	t.Helper()
	client, err := objectstore.New(cfg)
	if err != nil {
		t.Fatalf("construct provider conformance client: %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func registerS3ConformanceCleanup(t *testing.T, client *objectstore.Client, cfg objectstore.Config, roots *x509.CertPool, prefix string, keys []string) {
	t.Helper()
	direct := newS3ConformanceCleanupClient(t, cfg, roots)
	t.Cleanup(func() {
		defer direct.close()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		for _, key := range keys {
			if err := client.Delete(ctx, key); err != nil {
				t.Errorf("provider conformance cleanup delete failed with %q", objectstorage.Kind(err))
				continue
			}
			if _, err := client.Metadata(ctx, key); objectstorage.Kind(err) != objectstorage.KindNotFound {
				t.Errorf("provider conformance cleanup readback = %q, want not_found", objectstorage.Kind(err))
			}
		}
		if err := direct.abortPrefix(ctx, prefix); err != nil {
			t.Error("provider conformance multipart cleanup did not reach an empty prefix")
		}
	})
}

type s3ConformanceCleanupClient struct {
	sdk    *awss3.Client
	client *boundedhttp.Client
	bucket string
	owner  *string
}

func newS3ConformanceCleanupClient(t *testing.T, cfg objectstore.Config, roots *x509.CertPool) *s3ConformanceCleanupClient {
	t.Helper()
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil {
		t.Fatal("parse provider conformance cleanup endpoint")
	}
	endpoint.Host = cfg.Bucket + "." + endpoint.Host
	client, err := boundedhttp.New(boundedhttp.Config{
		DependencyName: "object-storage-conformance-cleanup", BaseURL: endpoint.String(),
		TargetClass: boundedhttp.ExternalHTTPS, OneAttempt: true, RootCAs: roots,
		DisableInstrumentation: true, RequestTimeout: 30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second, MaxResponseHeaderBytes: 64 << 10,
		MaxResponseBodyBytes: 1 << 20, MaxConnsPerHost: 1,
	}, nil)
	if err != nil {
		t.Fatal("construct provider conformance cleanup transport")
	}
	options := awss3.Options{
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
		Region:      cfg.Region, Retryer: aws.NopRetryer{}, HTTPClient: client,
		EndpointResolverV2:         s3ConformanceResolver{endpoint: *endpoint, region: cfg.Region, bucket: cfg.Bucket},
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		DisableClockSkewCorrection: true, DisableMultiRegionAccessPoints: true,
		DisableS3ExpressSessionAuth: aws.Bool(true), UseARNRegion: false, UseAccelerate: false,
		UseDualstack: false, UsePathStyle: false,
	}
	var owner *string
	if cfg.Provider == objectstore.ProviderAmazonS3 {
		owner = aws.String(cfg.ExpectedBucketOwner)
	}
	return &s3ConformanceCleanupClient{sdk: awss3.New(options), client: client, bucket: cfg.Bucket, owner: owner}
}

func (c *s3ConformanceCleanupClient) close() { c.client.CloseIdleConnections() }

func (c *s3ConformanceCleanupClient) abortPrefix(ctx context.Context, prefix string) error {
	for range 3 {
		uploads, err := c.multipartUploads(ctx, prefix)
		if err != nil {
			return err
		}
		if len(uploads) == 0 {
			return nil
		}
		for _, upload := range uploads {
			if _, err := c.sdk.AbortMultipartUpload(ctx, &awss3.AbortMultipartUploadInput{
				Bucket: aws.String(c.bucket), ExpectedBucketOwner: c.owner,
				Key: aws.String(upload.key), UploadId: aws.String(upload.id),
			}); err != nil {
				return errors.New("abort provider conformance multipart upload")
			}
		}
	}
	uploads, err := c.multipartUploads(ctx, prefix)
	if err != nil {
		return err
	}
	if len(uploads) == 0 {
		return nil
	}
	return errors.New("provider conformance multipart prefix remains non-empty")
}

func (c *s3ConformanceCleanupClient) multipartUploads(ctx context.Context, prefix string) ([]s3ConformanceUpload, error) {
	var uploads []s3ConformanceUpload
	var keyMarker, uploadMarker *string
	for range 10 {
		out, err := c.sdk.ListMultipartUploads(ctx, &awss3.ListMultipartUploadsInput{
			Bucket: aws.String(c.bucket), ExpectedBucketOwner: c.owner, Prefix: aws.String(prefix),
			KeyMarker: keyMarker, UploadIdMarker: uploadMarker, MaxUploads: aws.Int32(1_000),
		})
		if err != nil || out == nil || out.IsTruncated == nil {
			return nil, errors.New("list provider conformance multipart uploads")
		}
		for _, upload := range out.Uploads {
			if upload.Key == nil || upload.UploadId == nil || !strings.HasPrefix(*upload.Key, prefix) {
				return nil, errors.New("provider conformance multipart listing escaped its prefix")
			}
			uploads = append(uploads, s3ConformanceUpload{key: *upload.Key, id: *upload.UploadId})
		}
		if !*out.IsTruncated {
			return uploads, nil
		}
		if out.NextKeyMarker == nil || out.NextUploadIdMarker == nil || *out.NextKeyMarker == "" || *out.NextUploadIdMarker == "" ||
			keyMarker != nil && uploadMarker != nil && *out.NextKeyMarker == *keyMarker && *out.NextUploadIdMarker == *uploadMarker {
			return nil, errors.New("provider conformance multipart pagination did not advance")
		}
		keyMarker, uploadMarker = out.NextKeyMarker, out.NextUploadIdMarker
	}
	return nil, errors.New("provider conformance multipart listing exceeded its page bound")
}

type s3ConformanceUpload struct{ key, id string }

type s3ConformanceResolver struct {
	endpoint url.URL
	region   string
	bucket   string
}

func (r s3ConformanceResolver) ResolveEndpoint(_ context.Context, params awss3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	if params.Region == nil || *params.Region != r.region || params.Bucket == nil || *params.Bucket != r.bucket ||
		params.ForcePathStyle != nil && *params.ForcePathStyle || params.Accelerate != nil && *params.Accelerate ||
		params.UseDualStack != nil && *params.UseDualStack || params.UseFIPS != nil && *params.UseFIPS ||
		params.UseArnRegion != nil && *params.UseArnRegion {
		return smithyendpoints.Endpoint{}, errors.New("alternate provider conformance cleanup authority")
	}
	return smithyendpoints.Endpoint{URI: r.endpoint}, nil
}

func runS3Conformance(t *testing.T, primary, concealment *objectstore.Client, roots *x509.CertPool, keys []string) {
	t.Helper()
	ctx := t.Context()
	missing := keys[0] + "-missing"
	if _, err := primary.Metadata(ctx, missing); objectstorage.Kind(err) != objectstorage.KindNotFound {
		t.Fatalf("primary missing metadata = %q, want not_found", objectstorage.Kind(err))
	}

	single := []byte("portable-object-storage-conformance")
	result, err := primary.Upload(ctx, keys[0], io.NopCloser(bytes.NewReader(single)), objectstorage.UploadOptions{
		ContentLength: int64(len(single)), ContentType: "application/octet-stream", Intent: objectstorage.UploadCreateOnly,
	})
	if err != nil || result.Cleanup != objectstorage.CleanupNone {
		t.Fatalf("create-only upload = %q, cleanup %q", objectstorage.Kind(err), result.Cleanup)
	}
	if _, err := primary.Upload(ctx, keys[0], io.NopCloser(bytes.NewReader(single)), objectstorage.UploadOptions{ContentLength: int64(len(single)), Intent: objectstorage.UploadCreateOnly}); objectstorage.Kind(err) != objectstorage.KindPreconditionFailed {
		t.Fatalf("create-only collision = %q, want precondition_failed", objectstorage.Kind(err))
	}
	replacement := append([]byte(nil), single...)
	replacement[0] = 'P'
	if result, err = primary.Upload(ctx, keys[0], io.NopCloser(bytes.NewReader(replacement)), objectstorage.UploadOptions{ContentLength: int64(len(replacement)), ContentType: "application/octet-stream", Intent: objectstorage.UploadReplace}); err != nil || result.Cleanup != objectstorage.CleanupNone {
		t.Fatalf("replacement upload = %q, cleanup %q", objectstorage.Kind(err), result.Cleanup)
	}
	assertS3ConformanceObject(t, primary, keys[0], replacement, "application/octet-stream")
	if _, err := concealment.Metadata(ctx, keys[0]); objectstorage.Kind(err) != objectstorage.KindDenied {
		t.Fatalf("concealment identity metadata = %q, want denied", objectstorage.Kind(err))
	}

	multipart := bytes.Repeat([]byte("m"), int(conformanceChunkBytes+257))
	if result, err = primary.Upload(ctx, keys[1], io.NopCloser(bytes.NewReader(multipart)), objectstorage.UploadOptions{ContentLength: int64(len(multipart)), Intent: objectstorage.UploadReplace}); err != nil || result.Cleanup != objectstorage.CleanupNone {
		t.Fatalf("multipart upload = %q, cleanup %q", objectstorage.Kind(err), result.Cleanup)
	}
	assertS3ConformanceObject(t, primary, keys[1], multipart, "")

	result, err = primary.Upload(ctx, keys[2], io.NopCloser(bytes.NewReader(multipart[:conformanceChunkBytes])), objectstorage.UploadOptions{ContentLength: conformanceChunkBytes + 1, Intent: objectstorage.UploadReplace})
	if err == nil {
		t.Fatal("short multipart source unexpectedly succeeded")
	}
	if result.Cleanup != objectstorage.CleanupPending {
		t.Fatalf("forced multipart cleanup = %q", result.Cleanup)
	}
	t.Logf("forced multipart cleanup disposition: %s", result.Cleanup)

	presignedClient := newS3PresignedHTTPClient(roots)
	t.Cleanup(presignedClient.CloseIdleConnections)
	for range 2 {
		presigned, presignErr := primary.PresignGET(ctx, keys[1], time.Minute)
		if presignErr != nil {
			t.Fatalf("presign GET = %q", objectstorage.Kind(presignErr))
		}
		if got := executeS3Presigned(t, presignedClient, presigned, "", http.StatusOK); !bytes.Equal(got, multipart) {
			t.Fatal("presigned GET bytes changed")
		}
		assertS3PresignedMutationsRejected(t, presignedClient, presigned)
		rangeHeaders := presigned.Headers.Clone()
		rangeHeaders.Set("Range", "bytes=0-2")
		presigned.Headers = rangeHeaders
		if got := executeS3Presigned(t, presignedClient, presigned, "", http.StatusPartialContent); !bytes.Equal(got, multipart[:3]) {
			t.Fatal("presigned unsigned Range bytes changed")
		}
	}

	if err := primary.Delete(ctx, keys[0]); err != nil {
		t.Fatalf("delete existing = %q", objectstorage.Kind(err))
	}
	if _, err := primary.Metadata(ctx, keys[0]); objectstorage.Kind(err) != objectstorage.KindNotFound {
		t.Fatalf("metadata after delete = %q, want not_found", objectstorage.Kind(err))
	}
	if err := primary.Delete(ctx, keys[0]); err != nil {
		t.Fatalf("delete absent = %q", objectstorage.Kind(err))
	}
}

func assertS3ConformanceObject(t *testing.T, client *objectstore.Client, key string, want []byte, contentType string) {
	t.Helper()
	metadata, err := client.Metadata(t.Context(), key)
	if err != nil || metadata.Size != int64(len(want)) || metadata.LastModified.IsZero() || metadata.LastModified.Location() != time.UTC ||
		contentType != "" && metadata.ContentType != contentType {
		t.Fatalf("metadata = size %d, modified %t, error %q", metadata.Size, !metadata.LastModified.IsZero(), objectstorage.Kind(err))
	}
	download, err := client.Download(t.Context(), key)
	if err != nil {
		t.Fatalf("download acquisition = %q", objectstorage.Kind(err))
	}
	got, err := io.ReadAll(download.Body)
	closeErr := download.Body.Close()
	if err != nil || closeErr != nil || !bytes.Equal(got, want) || download.Size != int64(len(want)) ||
		download.LastModified.IsZero() || download.LastModified.Location() != time.UTC || contentType != "" && download.ContentType != contentType {
		t.Fatalf("download = bytes %d, size %d, read error %q", len(got), download.Size, objectstorage.Kind(err))
	}
}

func newS3PresignedHTTPClient(roots *x509.CertPool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                  nil,
			TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots},
			ResponseHeaderTimeout:  30 * time.Second,
			MaxResponseHeaderBytes: 64 << 10,
			MaxConnsPerHost:        1,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Timeout:       30 * time.Second,
	}
}

func executeS3Presigned(t *testing.T, client *http.Client, presigned objectstorage.PresignedGET, host string, wantStatus int) []byte {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), presigned.Method, presigned.URL, http.NoBody)
	if err != nil {
		t.Fatal("construct presigned request")
	}
	for name, values := range presigned.Headers {
		if strings.EqualFold(name, "host") {
			if len(values) == 1 {
				request.Host = values[0]
			}
			continue
		}
		request.Header[name] = append([]string(nil), values...)
	}
	if host != "" {
		request.Host = host
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

func assertS3PresignedMutationsRejected(t *testing.T, client *http.Client, presigned objectstorage.PresignedGET) {
	t.Helper()
	for _, mutation := range []struct {
		name   string
		method string
		host   string
		url    func(string) string
	}{
		{name: "method", method: http.MethodHead},
		{name: "host", host: "invalid.example"},
		{name: "key", url: func(raw string) string {
			target, _ := url.Parse(raw)
			target.Path += "-mutated"
			target.RawPath = ""
			return target.String()
		}},
		{name: "query", url: func(raw string) string {
			target, _ := url.Parse(raw)
			query := target.Query()
			query.Set("X-Amz-Signature", strings.Repeat("0", 64))
			target.RawQuery = query.Encode()
			return target.String()
		}},
	} {
		t.Run("presigned rejects "+mutation.name, func(t *testing.T) {
			t.Parallel()
			candidate := presigned
			if mutation.method != "" {
				candidate.Method = mutation.method
			}
			if mutation.url != nil {
				candidate.URL = mutation.url(candidate.URL)
			}
			request, err := http.NewRequestWithContext(t.Context(), candidate.Method, candidate.URL, http.NoBody)
			if err != nil {
				t.Fatal("construct mutated presigned request")
			}
			for name, values := range candidate.Headers {
				if strings.EqualFold(name, "host") {
					if len(values) == 1 {
						request.Host = values[0]
					}
					continue
				}
				request.Header[name] = append([]string(nil), values...)
			}
			if mutation.host != "" {
				request.Host = mutation.host
			}
			response, err := client.Do(request)
			if err != nil {
				t.Fatal("execute mutated presigned request")
			}
			_ = response.Body.Close()
			if response.StatusCode < 400 || response.StatusCode >= 500 {
				t.Fatalf("mutated presigned status = %d, want 4xx", response.StatusCode)
			}
		})
	}
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

func requiredPositiveInt64(t *testing.T, name string) int64 {
	t.Helper()
	value, err := strconv.ParseInt(requiredEnv(t, name), 10, 64)
	if err != nil || value <= 0 {
		t.Fatalf("%s must be a positive integer", name)
	}
	return value
}

// profile:object-storage:end
