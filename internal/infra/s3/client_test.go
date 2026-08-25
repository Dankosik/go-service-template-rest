package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

func TestConfigAndTransportPolicy(t *testing.T) {
	valid := testConfig()
	if err := valid.validate(); err != nil {
		t.Fatalf("valid Amazon config error = %v", err)
	}
	minimum := valid
	minimum.Bucket = "a1b"
	minimum.MaxObjectBytes = maximumObjectBytes
	if err := minimum.validate(); err != nil {
		t.Fatalf("minimum bucket and maximum object config error = %v", err)
	}
	r2 := valid
	r2.Provider = ProviderCloudflare
	r2.Endpoint = "https://0123456789abcdef0123456789abcdef.r2.cloudflarestorage.com"
	r2.Region = "auto"
	r2.ExpectedBucketOwner = ""
	if err := r2.validate(); err != nil {
		t.Fatalf("valid R2 config error = %v", err)
	}

	for _, mutate := range []func(*Config){
		func(cfg *Config) { cfg.Bucket = "" },
		func(cfg *Config) { cfg.Bucket = "x" },
		func(cfg *Config) { cfg.Bucket = "xx" },
		func(cfg *Config) { cfg.Bucket = strings.Repeat("x", 64) },
		func(cfg *Config) { cfg.MaxObjectBytes = maximumObjectBytes + 1 },
		func(cfg *Config) { cfg.CredentialSource = "ambient" },
		func(cfg *Config) { cfg.Endpoint = "https://override.example" },
	} {
		candidate := valid
		mutate(&candidate)
		if err := candidate.validate(); err == nil {
			t.Fatalf("invalid config accepted: %#v", candidate)
		}
	}

	client, transport := newHTTPClient()
	if transport.Proxy != nil || transport.MaxConnsPerHost != maximumActiveOperations ||
		transport.TLSClientConfig == nil || transport.TLSClientConfig.MinVersion != 0x0303 {
		t.Fatalf("transport policy = %#v", transport)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	if err := client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v, want ErrUseLastResponse", err)
	}
}

func TestSDKPolicyOptions(t *testing.T) {
	retryOptions := retry.StandardOptions{}
	configureRetry(&retryOptions)
	if retryOptions.MaxAttempts != 3 || retryOptions.MaxBackoff != time.Second {
		t.Fatalf("retry policy = attempts %d backoff %s", retryOptions.MaxAttempts, retryOptions.MaxBackoff)
	}

	transferOptions := transfermanager.Options{}
	configureTransfer(&transferOptions)
	if transferOptions.PartSizeBytes != multipartPartBytes ||
		transferOptions.MultipartUploadThreshold != multipartPartBytes ||
		transferOptions.Concurrency != 1 || transferOptions.FailTimeout != multipartFailureTimeout ||
		transferOptions.MaxUploadParts != maximumUploadParts ||
		transferOptions.ChecksumAlgorithm != tmtypes.ChecksumAlgorithm("CRC64NVME") ||
		transferOptions.RequestChecksumCalculation != aws.RequestChecksumCalculationWhenRequired {
		t.Fatalf("transfer policy = %#v", transferOptions)
	}
}

func TestUploadDelegatesSingleAndMultipart(t *testing.T) {
	var puts, multiparts int
	api := &fakeObjectAPI{put: func(_ context.Context, input *awss3.PutObjectInput) (*awss3.PutObjectOutput, error) {
		puts++
		if aws.ToString(input.ExpectedBucketOwner) != "123456789012" || input.ChecksumAlgorithm != types.ChecksumAlgorithmCrc64nvme {
			t.Fatalf("PutObject input = %#v", input)
		}
		return &awss3.PutObjectOutput{ChecksumCRC64NVME: new("checksum"), ChecksumType: types.ChecksumTypeFullObject}, nil
	}}
	uploader := fakeUploader(func(_ context.Context, input *transfermanager.UploadObjectInput) (*transfermanager.UploadObjectOutput, error) {
		multiparts++
		if input.IfNoneMatch != nil || input.ChecksumAlgorithm != tmtypes.ChecksumAlgorithm("CRC64NVME") ||
			input.ChecksumType != tmtypes.ChecksumTypeFullObject || aws.ToInt64(input.MpuObjectSize) != multipartPartBytes+1 {
			t.Fatalf("UploadObject input = %#v", input)
		}
		return &transfermanager.UploadObjectOutput{ChecksumCRC64NVME: new("checksum"), ChecksumType: tmtypes.ChecksumTypeFullObject}, nil
	})
	client := testClient(api, uploader)

	if err := client.Upload(t.Context(), "small", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1, IfNotExists: true}); err != nil {
		t.Fatalf("small Upload() error = %v", err)
	}
	if err := client.Upload(t.Context(), "large", bytes.NewReader(make([]byte, multipartPartBytes+1)), objectstorage.UploadOptions{Size: multipartPartBytes + 1}); err != nil {
		t.Fatalf("multipart Upload() error = %v", err)
	}
	if err := client.Upload(t.Context(), "create-large", bytes.NewReader(make([]byte, multipartPartBytes+1)), objectstorage.UploadOptions{Size: multipartPartBytes + 1, IfNotExists: true}); !errors.Is(err, objectstorage.ErrInvalid) {
		t.Fatalf("multipart create-only error = %v, want ErrInvalid", err)
	}
	if puts != 1 || multiparts != 1 {
		t.Fatalf("upload calls = put %d multipart %d, want 1/1", puts, multiparts)
	}
}

func TestPortableErrorsAndAdmission(t *testing.T) {
	api := &fakeObjectAPI{
		put: func(context.Context, *awss3.PutObjectInput) (*awss3.PutObjectOutput, error) {
			return nil, providerTestError{status: http.StatusPreconditionFailed, code: "PreconditionFailed"}
		},
		head: func(context.Context, *awss3.HeadObjectInput) (*awss3.HeadObjectOutput, error) {
			return nil, providerTestError{status: http.StatusNotFound, code: "NotFound"}
		},
	}
	client := testClient(api, nil)
	if err := client.Upload(t.Context(), "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1, IfNotExists: true}); !errors.Is(err, objectstorage.ErrAlreadyExists) {
		t.Fatalf("create-only error = %v, want ErrAlreadyExists", err)
	}
	if _, err := client.Metadata(t.Context(), "missing"); !errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("Metadata() error = %v, want ErrNotFound", err)
	}

	client.tokens = make(chan struct{}, 1)
	client.tokens <- struct{}{}
	if _, err := client.Metadata(t.Context(), "busy"); !errors.Is(err, objectstorage.ErrBusy) {
		t.Fatalf("busy Metadata() error = %v, want ErrBusy", err)
	}
}

func TestDownloadReleasesAtValidatedEOF(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader("payload")}
	api := &fakeObjectAPI{get: func(context.Context, *awss3.GetObjectInput) (*awss3.GetObjectOutput, error) {
		return &awss3.GetObjectOutput{
			Body: body, ContentLength: new(int64(7)), ContentType: new("text/plain"),
			LastModified: new(time.Unix(1, 0)), ChecksumCRC64NVME: new("checksum"),
			ChecksumType: types.ChecksumTypeFullObject,
		}, nil
	}}
	client := testClient(api, nil)
	client.tokens = make(chan struct{}, 1)

	object, err := client.Download(t.Context(), "object")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if len(client.tokens) != 1 {
		t.Fatalf("admission tokens = %d, want 1 while body is retained", len(client.tokens))
	}
	data, err := io.ReadAll(object.Body)
	if err != nil || string(data) != "payload" {
		t.Fatalf("ReadAll() = %q, %v", data, err)
	}
	if len(client.tokens) != 0 || body.closes.Load() != 1 {
		t.Fatalf("terminal release = tokens %d closes %d", len(client.tokens), body.closes.Load())
	}
}

func TestPresignGetReturnsSelfContainedLibraryURL(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	client, err := New(t.Context(), testConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(client.Close)

	got, err := client.PresignGet(t.Context(), "object", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet() error = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse presigned URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("x-amz-expected-bucket-owner") != "123456789012" || query.Get("X-Amz-SignedHeaders") != "host" {
		t.Fatalf("presigned URL query = %v", query)
	}
}

func TestNewBuildsLibraryClientWithoutProviderIO(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	client, err := New(t.Context(), testConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.sdk == nil || client.uploader == nil || client.presigner == nil || client.transport == nil {
		t.Fatal("New() returned an incomplete library client")
	}
	client.Close()
	client.Close()
}

func TestStoreRejectsInvalidCallsBeforeProviderIO(t *testing.T) {
	client := testClient(&fakeObjectAPI{}, nil)
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	for _, test := range []struct {
		name string
		run  func() error
		want error
	}{
		{"nil context", func() error {
			return client.Upload(nil, "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1}) //nolint:staticcheck // Nil is the trust-boundary input under test.
		}, objectstorage.ErrInvalid},
		{"canceled context", func() error {
			return client.Upload(canceled, "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1})
		}, context.Canceled},
		{"invalid key", func() error {
			return client.Upload(t.Context(), "", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1})
		}, objectstorage.ErrInvalid},
		{"nil source", func() error {
			return client.Upload(t.Context(), "object", nil, objectstorage.UploadOptions{Size: 1})
		}, objectstorage.ErrInvalid},
		{"negative size", func() error {
			return client.Upload(t.Context(), "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: -1})
		}, objectstorage.ErrInvalid},
		{"oversized object", func() error {
			return client.Upload(t.Context(), "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: client.config.MaxObjectBytes + 1})
		}, objectstorage.ErrTooLarge},
		{"invalid content type", func() error {
			return client.Upload(t.Context(), "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1, ContentType: "bad\nvalue"})
		}, objectstorage.ErrInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, test.want) {
				t.Fatalf("operation error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestMutationErrorPreservesOutcomeAfterCancellation(t *testing.T) {
	t.Run("definitive rejection", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		api := &fakeObjectAPI{put: func(context.Context, *awss3.PutObjectInput) (*awss3.PutObjectOutput, error) {
			cancel()
			return nil, providerTestError{status: http.StatusPreconditionFailed, code: "PreconditionFailed"}
		}}
		err := testClient(api, nil).Upload(ctx, "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1, IfNotExists: true})
		if !errors.Is(err, objectstorage.ErrAlreadyExists) {
			t.Fatalf("Upload() error = %v, want ErrAlreadyExists", err)
		}
	})

	t.Run("unknown outcome", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		api := &fakeObjectAPI{put: func(context.Context, *awss3.PutObjectInput) (*awss3.PutObjectOutput, error) {
			cancel()
			return nil, providerTestError{status: http.StatusInternalServerError, code: "InternalError"}
		}}
		err := testClient(api, nil).Upload(ctx, "object", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1})
		if !errors.Is(err, objectstorage.ErrOutcomeUnknown) || !errors.Is(err, context.Canceled) {
			t.Fatalf("Upload() error = %v, want ErrOutcomeUnknown and context.Canceled", err)
		}
	})
}

func TestStoreMapsOperationResults(t *testing.T) {
	modified := time.Unix(10, 0)
	api := &fakeObjectAPI{
		get: func(context.Context, *awss3.GetObjectInput) (*awss3.GetObjectOutput, error) {
			return nil, providerTestError{status: http.StatusNotFound, code: "NoSuchKey"}
		},
		head: func(context.Context, *awss3.HeadObjectInput) (*awss3.HeadObjectOutput, error) {
			return &awss3.HeadObjectOutput{ContentLength: new(int64(3)), ContentType: new("text/plain"), LastModified: &modified}, nil
		},
		delete: func(context.Context, *awss3.DeleteObjectInput) (*awss3.DeleteObjectOutput, error) {
			return &awss3.DeleteObjectOutput{}, nil
		},
	}
	client := testClient(api, nil)
	metadata, err := client.Metadata(t.Context(), "object")
	if err != nil || metadata.Size != 3 || metadata.ContentType != "text/plain" || !metadata.LastModified.Equal(modified.UTC()) {
		t.Fatalf("Metadata() = %#v, %v", metadata, err)
	}
	if err := client.Delete(t.Context(), "object"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := client.Download(t.Context(), "missing"); !errors.Is(err, objectstorage.ErrNotFound) {
		t.Fatalf("Download() error = %v, want ErrNotFound", err)
	}

	api.head = func(context.Context, *awss3.HeadObjectInput) (*awss3.HeadObjectOutput, error) {
		return nil, nil //nolint:nilnil // Incomplete SDK result under test.
	}
	if _, err := client.Metadata(t.Context(), "object"); !errors.Is(err, errRequestFailed) {
		t.Fatalf("Metadata(nil) error = %v, want request failure", err)
	}
	api.delete = func(context.Context, *awss3.DeleteObjectInput) (*awss3.DeleteObjectOutput, error) {
		return nil, providerTestError{status: http.StatusInternalServerError, code: "InternalError"}
	}
	if err := client.Delete(t.Context(), "object"); !errors.Is(err, objectstorage.ErrOutcomeUnknown) {
		t.Fatalf("Delete() error = %v, want ErrOutcomeUnknown", err)
	}
}

func TestStoreRejectsIncompleteLibraryResults(t *testing.T) {
	api := &fakeObjectAPI{
		put: func(context.Context, *awss3.PutObjectInput) (*awss3.PutObjectOutput, error) {
			return nil, nil //nolint:nilnil // Incomplete SDK result under test.
		},
		get: func(context.Context, *awss3.GetObjectInput) (*awss3.GetObjectOutput, error) {
			return nil, nil //nolint:nilnil // Incomplete SDK result under test.
		},
		head: func(context.Context, *awss3.HeadObjectInput) (*awss3.HeadObjectOutput, error) {
			return &awss3.HeadObjectOutput{ContentLength: new(int64(-1)), LastModified: new(time.Unix(1, 0))}, nil
		},
	}
	uploader := fakeUploader(func(context.Context, *transfermanager.UploadObjectInput) (*transfermanager.UploadObjectOutput, error) {
		return nil, nil //nolint:nilnil // Incomplete SDK result under test.
	})
	client := testClient(api, uploader)
	if err := client.Upload(t.Context(), "small", strings.NewReader("x"), objectstorage.UploadOptions{Size: 1}); !errors.Is(err, objectstorage.ErrIntegrity) {
		t.Fatalf("small Upload() error = %v, want ErrIntegrity", err)
	}
	if err := client.Upload(t.Context(), "large", strings.NewReader("x"), objectstorage.UploadOptions{Size: multipartPartBytes + 1}); !errors.Is(err, objectstorage.ErrIntegrity) {
		t.Fatalf("multipart Upload() error = %v, want ErrIntegrity", err)
	}
	if _, err := client.Download(t.Context(), "object"); !errors.Is(err, objectstorage.ErrIntegrity) {
		t.Fatalf("Download() error = %v, want ErrIntegrity", err)
	}
	if _, err := client.Metadata(t.Context(), "object"); !errors.Is(err, errRequestFailed) {
		t.Fatalf("Metadata() error = %v, want request failure", err)
	}

	client.presigner = fakePresigner(func(context.Context, *awss3.GetObjectInput) (*signer.PresignedHTTPRequest, error) {
		return nil, nil //nolint:nilnil // Incomplete SDK result under test.
	})
	if _, err := client.PresignGet(t.Context(), "object", time.Minute); !errors.Is(err, errRequestFailed) {
		t.Fatalf("PresignGet() error = %v, want request failure", err)
	}
	if _, err := client.PresignGet(t.Context(), "object", time.Millisecond); !errors.Is(err, objectstorage.ErrInvalid) {
		t.Fatalf("PresignGet(invalid TTL) error = %v, want ErrInvalid", err)
	}
}

func testConfig() Config {
	return Config{
		Provider: ProviderAmazonS3, Region: "us-east-1", Bucket: "examplebucket",
		ExpectedBucketOwner: "123456789012", CredentialSource: CredentialSourceStatic,
		MaxObjectBytes: 20 << 20,
	}
}

func testClient(api objectAPI, uploader uploadAPI) *Client {
	if uploader == nil {
		uploader = fakeUploader(func(context.Context, *transfermanager.UploadObjectInput) (*transfermanager.UploadObjectOutput, error) {
			return nil, errors.New("unexpected multipart upload")
		})
	}
	return &Client{config: testConfig(), sdk: api, uploader: uploader, tokens: make(chan struct{}, maximumActiveOperations)}
}

type fakeObjectAPI struct {
	put    func(context.Context, *awss3.PutObjectInput) (*awss3.PutObjectOutput, error)
	get    func(context.Context, *awss3.GetObjectInput) (*awss3.GetObjectOutput, error)
	head   func(context.Context, *awss3.HeadObjectInput) (*awss3.HeadObjectOutput, error)
	delete func(context.Context, *awss3.DeleteObjectInput) (*awss3.DeleteObjectOutput, error)
}

func (f *fakeObjectAPI) PutObject(ctx context.Context, input *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	return f.put(ctx, input)
}

func (f *fakeObjectAPI) GetObject(ctx context.Context, input *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	return f.get(ctx, input)
}

func (f *fakeObjectAPI) HeadObject(ctx context.Context, input *awss3.HeadObjectInput, _ ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	return f.head(ctx, input)
}

func (f *fakeObjectAPI) DeleteObject(ctx context.Context, input *awss3.DeleteObjectInput, _ ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	return f.delete(ctx, input)
}

type fakeUploader func(context.Context, *transfermanager.UploadObjectInput) (*transfermanager.UploadObjectOutput, error)

func (f fakeUploader) UploadObject(ctx context.Context, input *transfermanager.UploadObjectInput, _ ...func(*transfermanager.Options)) (*transfermanager.UploadObjectOutput, error) {
	return f(ctx, input)
}

type fakePresigner func(context.Context, *awss3.GetObjectInput) (*signer.PresignedHTTPRequest, error)

func (f fakePresigner) PresignGetObject(ctx context.Context, input *awss3.GetObjectInput, _ ...func(*awss3.PresignOptions)) (*signer.PresignedHTTPRequest, error) {
	return f(ctx, input)
}

type providerTestError struct {
	status int
	code   string
}

func (e providerTestError) Error() string                 { return "provider detail" }
func (e providerTestError) ErrorCode() string             { return e.code }
func (e providerTestError) ErrorMessage() string          { return "provider detail" }
func (e providerTestError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }
func (e providerTestError) HTTPStatusCode() int           { return e.status }

type trackedBody struct {
	io.Reader

	closes atomic.Int32
}

func (b *trackedBody) Close() error {
	b.closes.Add(1)
	return nil
}
