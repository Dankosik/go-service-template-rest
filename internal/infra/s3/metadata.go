package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

// Metadata projects one HeadObject response into the provider-neutral port.
//
//nolint:contextcheck,wrapcheck // Preserve the span rooted in ctx and the closed object-storage errors.
func (c *Client) Metadata(ctx context.Context, key string) (result objectstorage.Metadata, err error) {
	call := c.telemetry.begin(ctx, telemetryOperationMetadata)
	defer func() { call.finish(err, 0) }()
	if err := objectstorage.ValidateKey(key); err != nil {
		return result, err
	}
	effective, release, err := c.admit(ctx, call)
	if err != nil {
		return result, c.admissionError(err)
	}
	defer release()
	if err := effective.Err(); err != nil {
		return result, c.admissionError(err)
	}

	out, err := c.sdk.HeadObject(effective, &awss3.HeadObjectInput{Bucket: aws.String(c.config.Bucket), Key: aws.String(key)})
	if err != nil {
		return result, operationError(operationMetadata, err, false)
	}
	if out == nil || out.ContentLength == nil || *out.ContentLength < 0 || out.LastModified == nil {
		return result, objectstorage.NewError(objectstorage.KindInternal)
	}
	return objectstorage.Metadata{
		Size:         *out.ContentLength,
		ContentType:  aws.ToString(out.ContentType),
		LastModified: aws.ToTime(out.LastModified).UTC(),
	}, nil
}
