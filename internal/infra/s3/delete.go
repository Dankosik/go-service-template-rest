package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

// Delete performs one unversioned delete operation; success does not imply erasure.
//
//nolint:contextcheck,wrapcheck // Preserve the span rooted in ctx and the closed object-storage errors.
func (c *Client) Delete(ctx context.Context, key string) (err error) {
	call := c.telemetry.begin(ctx, telemetryOperationDelete)
	defer func() { call.finish(err, 0) }()
	if err := objectstorage.ValidateKey(key); err != nil {
		return err
	}
	effective, release, err := c.admit(ctx, call)
	if err != nil {
		return c.admissionError(err)
	}
	defer release()
	if err := effective.Err(); err != nil {
		return c.admissionError(err)
	}

	send := &sendState{}
	_, err = c.sdk.DeleteObject(withSendState(effective, send), &awss3.DeleteObjectInput{
		Bucket: aws.String(c.config.Bucket), ExpectedBucketOwner: c.expectedBucketOwner(), Key: aws.String(key),
	})
	if err != nil {
		return operationError(c.config.Provider, operationDelete, err, send)
	}
	return nil
}
