package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func TestUploadDelegatesSingleAndMultipart(t *testing.T) {
	var puts, multiparts int
	api := &fakeObjectAPI{put: func(_ context.Context, input *awss3.PutObjectInput) (*awss3.PutObjectOutput, error) {
		puts++
		if aws.ToString(input.ExpectedBucketOwner) != "123456789012" || input.ChecksumAlgorithm != types.ChecksumAlgorithmCrc64nvme {
			t.Fatalf("PutObject input = %#v", input)
		}
		return &awss3.PutObjectOutput{ChecksumCRC64NVME: aws.String("checksum"), ChecksumType: types.ChecksumTypeFullObject}, nil
	}}
	uploader := fakeUploader(func(_ context.Context, input *transfermanager.UploadObjectInput) (*transfermanager.UploadObjectOutput, error) {
		multiparts++
		if input.IfNoneMatch != nil || input.ChecksumAlgorithm != tmtypes.ChecksumAlgorithm("CRC64NVME") ||
			input.ChecksumType != tmtypes.ChecksumTypeFullObject || aws.ToInt64(input.MpuObjectSize) != multipartPartBytes+1 {
			t.Fatalf("UploadObject input = %#v", input)
		}
		return &transfermanager.UploadObjectOutput{ChecksumCRC64NVME: aws.String("checksum"), ChecksumType: tmtypes.ChecksumTypeFullObject}, nil
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
			Body: body, ContentLength: aws.Int64(7), ContentType: aws.String("text/plain"),
			LastModified: aws.Time(time.Unix(1, 0)), ChecksumCRC64NVME: aws.String("checksum"),
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

func TestPresignGetReturnsLibraryURL(t *testing.T) {
	client := testClient(&fakeObjectAPI{}, nil)
	client.presigner = fakePresigner(func(context.Context, *awss3.GetObjectInput) (*signer.PresignedHTTPRequest, error) {
		return &signer.PresignedHTTPRequest{URL: "https://example.com/object?signature", Method: http.MethodGet}, nil
	})
	got, err := client.PresignGet(t.Context(), "object", time.Minute)
	if err != nil || got != "https://example.com/object?signature" {
		t.Fatalf("PresignGet() = %q, %v", got, err)
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
