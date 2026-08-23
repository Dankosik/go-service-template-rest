package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/example/go-service-template-rest/internal/objectstorage"
	"golang.org/x/net/http/httpguts"
)

func (c *Client) Upload(ctx context.Context, key string, source io.Reader, options objectstorage.UploadOptions) error {
	if err := c.validateUpload(ctx, key, source, options); err != nil {
		return err
	}
	release, err := c.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	if options.Size <= multipartPartBytes {
		input := &awss3.PutObjectInput{
			Bucket: aws.String(c.config.Bucket), Key: aws.String(key), Body: source,
			ContentLength: aws.Int64(options.Size), ExpectedBucketOwner: c.expectedBucketOwner(),
			ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
		}
		input.ContentType = optionalString(options.ContentType)
		if options.IfNotExists {
			input.IfNoneMatch = aws.String("*")
		}
		output, putErr := c.sdk.PutObject(ctx, input)
		if putErr != nil {
			return mutationError(ctx, putErr)
		}
		if output == nil || output.ChecksumCRC64NVME == nil || output.ChecksumType != types.ChecksumTypeFullObject {
			return objectstorage.ErrIntegrity
		}
		return nil
	}

	output, uploadErr := c.uploader.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(c.config.Bucket), Key: aws.String(key), Body: readerOnly{Reader: source},
		ContentLength: aws.Int64(options.Size), MpuObjectSize: aws.Int64(options.Size),
		ContentType: optionalString(options.ContentType), ExpectedBucketOwner: c.expectedBucketOwner(),
		ChecksumAlgorithm: tmtypes.ChecksumAlgorithm("CRC64NVME"), ChecksumType: tmtypes.ChecksumTypeFullObject,
	})
	if uploadErr != nil {
		return mutationError(ctx, uploadErr)
	}
	if output == nil || output.ChecksumCRC64NVME == nil || output.ChecksumType != tmtypes.ChecksumTypeFullObject {
		return objectstorage.ErrIntegrity
	}
	return nil
}

func (c *Client) Download(ctx context.Context, key string) (objectstorage.Object, error) {
	if err := validateCall(ctx, key); err != nil {
		return objectstorage.Object{}, err
	}
	release, err := c.acquire(ctx)
	if err != nil {
		return objectstorage.Object{}, err
	}

	output, err := c.sdk.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket), Key: aws.String(key),
		ExpectedBucketOwner: c.expectedBucketOwner(), ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		release()
		return objectstorage.Object{}, readError(ctx, err)
	}
	if output == nil || output.Body == nil || output.ContentRange != nil || output.ChecksumCRC64NVME == nil || output.ChecksumType != types.ChecksumTypeFullObject {
		if output != nil && output.Body != nil {
			_ = output.Body.Close()
		}
		release()
		return objectstorage.Object{}, objectstorage.ErrIntegrity
	}
	metadata, err := c.metadata(output.ContentLength, output.ContentType, output.LastModified)
	if err != nil {
		_ = output.Body.Close()
		release()
		return objectstorage.Object{}, err
	}
	return objectstorage.Object{
		Body:     &downloadBody{ctx: ctx, body: output.Body, release: release},
		Metadata: metadata,
	}, nil
}

func (c *Client) Metadata(ctx context.Context, key string) (objectstorage.Metadata, error) {
	if err := validateCall(ctx, key); err != nil {
		return objectstorage.Metadata{}, err
	}
	release, err := c.acquire(ctx)
	if err != nil {
		return objectstorage.Metadata{}, err
	}
	defer release()

	output, err := c.sdk.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(c.config.Bucket), Key: aws.String(key), ExpectedBucketOwner: c.expectedBucketOwner(),
	})
	if err != nil {
		return objectstorage.Metadata{}, readError(ctx, err)
	}
	if output == nil {
		return objectstorage.Metadata{}, errRequestFailed
	}
	return c.metadata(output.ContentLength, output.ContentType, output.LastModified)
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if err := validateCall(ctx, key); err != nil {
		return err
	}
	release, err := c.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = c.sdk.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(c.config.Bucket), Key: aws.String(key), ExpectedBucketOwner: c.expectedBucketOwner(),
	})
	if err != nil {
		return mutationError(ctx, err)
	}
	return nil
}

func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := validateCall(ctx, key); err != nil {
		return "", err
	}
	if ttl < time.Second || ttl > maximumPresignLifetime || ttl%time.Second != 0 {
		return "", objectstorage.ErrInvalid
	}
	output, err := c.presigner.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket), Key: aws.String(key), ExpectedBucketOwner: c.expectedBucketOwner(),
	}, func(options *awss3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("presign object: %w", ctxErr)
		}
		return "", errRequestFailed
	}
	if output == nil || output.URL == "" || output.Method != http.MethodGet {
		return "", errRequestFailed
	}
	return output.URL, nil
}

func (c *Client) validateUpload(ctx context.Context, key string, source io.Reader, options objectstorage.UploadOptions) error {
	if err := validateCall(ctx, key); err != nil {
		return err
	}
	if source == nil || options.Size < 0 {
		return objectstorage.ErrInvalid
	}
	if options.Size > c.config.MaxObjectBytes {
		return objectstorage.ErrTooLarge
	}
	if len(options.ContentType) > maximumContentTypeBytes || !httpguts.ValidHeaderFieldValue(options.ContentType) {
		return objectstorage.ErrInvalid
	}
	if options.IfNotExists && options.Size > multipartPartBytes {
		return objectstorage.ErrInvalid
	}
	return nil
}

func validateCall(ctx context.Context, key string) error {
	if ctx == nil {
		return objectstorage.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("object storage call: %w", err)
	}
	if err := objectstorage.ValidateKey(key); err != nil {
		return fmt.Errorf("validate object key: %w", err)
	}
	return nil
}

func (c *Client) acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("object storage admission: %w", err)
	}
	select {
	case c.tokens <- struct{}{}:
		return sync.OnceFunc(func() { <-c.tokens }), nil
	default:
		return nil, objectstorage.ErrBusy
	}
}

func (c *Client) metadata(size *int64, contentType *string, modified *time.Time) (objectstorage.Metadata, error) {
	if size == nil || *size < 0 || modified == nil {
		return objectstorage.Metadata{}, errRequestFailed
	}
	if *size > c.config.MaxObjectBytes {
		return objectstorage.Metadata{}, objectstorage.ErrTooLarge
	}
	return objectstorage.Metadata{
		Size: *size, ContentType: aws.ToString(contentType), LastModified: modified.UTC(),
	}, nil
}

func readError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("read object storage response: %w", ctxErr)
	}
	status, code := providerError(err)
	if status == http.StatusNotFound && (code == "NoSuchKey" || code == "NotFound") {
		return objectstorage.ErrNotFound
	}
	return errRequestFailed
}

func mutationError(ctx context.Context, err error) error {
	status, code := providerError(err)
	if status == http.StatusPreconditionFailed && code == "PreconditionFailed" {
		return objectstorage.ErrAlreadyExists
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return errors.Join(objectstorage.ErrOutcomeUnknown, fmt.Errorf("mutate object storage: %w", ctxErr))
	}
	return objectstorage.ErrOutcomeUnknown
}

func providerError(err error) (int, string) {
	status := 0
	if response, ok := errors.AsType[interface {
		error
		HTTPStatusCode() int
	}](err); ok {
		status = response.HTTPStatusCode()
	}
	if apiError, ok := errors.AsType[smithy.APIError](err); ok {
		return status, apiError.ErrorCode()
	}
	return status, ""
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return aws.String(value)
}

type readerOnly struct{ io.Reader }

type downloadBody struct {
	//nolint:containedctx // The returned body owns the request context until EOF or Close.
	ctx     context.Context
	body    io.ReadCloser
	release func()
	once    sync.Once
}

func (b *downloadBody) Read(buffer []byte) (int, error) {
	count, err := b.body.Read(buffer)
	if err == nil {
		return count, nil
	}
	if ctxErr := b.ctx.Err(); ctxErr != nil {
		err = ctxErr
	} else if !errors.Is(err, io.EOF) {
		err = objectstorage.ErrIntegrity
	}
	b.finish()
	return count, err
}

func (b *downloadBody) Close() error {
	b.finish()
	return nil
}

func (b *downloadBody) finish() {
	b.once.Do(func() {
		_ = b.body.Close()
		b.release()
	})
}
