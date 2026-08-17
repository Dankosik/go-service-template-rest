package s3

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/samber/lo"

	"github.com/example/go-service-template-rest/internal/objectstorage"
	"golang.org/x/net/http/httpguts"
)

const (
	maximumCleanupListParts = 1_000
	maximumCleanupListPages = 10
	maximumCleanupCycles    = 3
	maximumContentTypeBytes = 1_024
)

// Upload streams exactly the declared bytes through one S3 write path.
//
//nolint:contextcheck // begin roots the span in ctx; finish only finalizes that outcome after the call.
func (c *Client) Upload(ctx context.Context, key string, source io.ReadCloser, options objectstorage.UploadOptions) (result objectstorage.UploadResult, err error) {
	call := c.telemetry.begin(ctx, telemetryOperationUpload)
	defer func() {
		if result.Cleanup != "" {
			call.cleanup = string(result.Cleanup)
		}
		call.finish(err, options.ContentLength)
	}()
	var closeSource func()
	if source != nil {
		var closeOnce sync.Once
		closeSource = func() { closeOnce.Do(func() { _ = source.Close() }) }
	}
	var release func()
	var stopClose func() bool
	defer func() {
		if stopClose != nil {
			_ = stopClose()
		}
		if closeSource != nil {
			closeSource()
		}
		if release != nil {
			release()
		}
	}()
	if err := c.validateUpload(key, source, options); err != nil {
		return result, err
	}
	if options.ContentLength <= c.config.MultipartChunkBytes {
		call.path = "single"
	} else {
		call.path = "multipart"
	}
	effective, admittedRelease, err := c.admit(ctx, call)
	if err != nil {
		return result, c.admissionError(err)
	}
	release = admittedRelease
	if err := effective.Err(); err != nil {
		return result, c.admissionError(err)
	}
	if closeSource != nil {
		stopClose = context.AfterFunc(effective, closeSource)
	}

	if options.ContentLength <= c.config.MultipartChunkBytes {
		return c.uploadSingle(effective, key, source, options)
	}
	return c.uploadMultipart(effective, key, source, options)
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func (c *Client) validateUpload(key string, source io.ReadCloser, options objectstorage.UploadOptions) error {
	if err := objectstorage.ValidateKey(key); err != nil {
		return err
	}
	if source == nil || options.ContentLength < 0 {
		return objectstorage.NewError(objectstorage.KindInvalid)
	}
	if options.ContentLength > c.config.MaxObjectBytes {
		return objectstorage.NewError(objectstorage.KindTooLarge)
	}
	if len(options.ContentType) > maximumContentTypeBytes || !httpguts.ValidHeaderFieldValue(options.ContentType) {
		return objectstorage.NewError(objectstorage.KindInvalid)
	}
	switch options.Intent {
	case objectstorage.UploadReplace:
	case objectstorage.UploadCreateOnly:
		if options.ContentLength > c.config.MultipartChunkBytes {
			return objectstorage.NewError(objectstorage.KindInvalid)
		}
	default:
		return objectstorage.NewError(objectstorage.KindInvalid)
	}
	return nil
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func (c *Client) uploadSingle(ctx context.Context, key string, source io.Reader, options objectstorage.UploadOptions) (objectstorage.UploadResult, error) {
	checksum := crc64NVME()
	body := newExactChecksumReader(source, options.ContentLength, checksum)
	send := &sendState{}
	ctx = withSendState(ctx, send)
	input := &awss3.PutObjectInput{
		Bucket:              aws.String(c.config.Bucket),
		ExpectedBucketOwner: c.expectedBucketOwner(),
		Key:                 aws.String(key),
		Body:                body,
		ContentLength:       aws.Int64(options.ContentLength),
		ChecksumAlgorithm:   types.ChecksumAlgorithmCrc64nvme,
	}
	if options.ContentType != "" {
		input.ContentType = aws.String(options.ContentType)
	}
	if options.Intent == objectstorage.UploadCreateOnly {
		input.IfNoneMatch = aws.String("*")
	}
	out, err := c.sdk.PutObject(ctx, input)
	if err != nil {
		return objectstorage.UploadResult{}, operationError(c.config.Provider, uploadOperation(options.Intent), err, send)
	}
	if err := body.complete(); err != nil {
		return objectstorage.UploadResult{}, operationError(c.config.Provider, uploadOperation(options.Intent), err, send)
	}
	if !matchingCRC64NVME(crc64NVMEBase64(checksum), out.ChecksumCRC64NVME) || out.ChecksumType != types.ChecksumTypeFullObject {
		return objectstorage.UploadResult{}, objectstorage.NewError(objectstorage.KindIntegrityFailed)
	}
	return objectstorage.UploadResult{Cleanup: objectstorage.CleanupNone}, nil
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func (c *Client) uploadMultipart(ctx context.Context, key string, source io.Reader, options objectstorage.UploadOptions) (result objectstorage.UploadResult, err error) {
	createSend := &sendState{}
	created, err := c.sdk.CreateMultipartUpload(withSendState(ctx, createSend), &awss3.CreateMultipartUploadInput{
		Bucket:              aws.String(c.config.Bucket),
		ExpectedBucketOwner: c.expectedBucketOwner(),
		Key:                 aws.String(key),
		ChecksumAlgorithm:   types.ChecksumAlgorithmCrc64nvme,
		ChecksumType:        types.ChecksumTypeFullObject,
		ContentType:         lo.EmptyableToPtr(options.ContentType),
	})
	if err != nil {
		if createSend.wroteHeaders {
			result.Cleanup = objectstorage.CleanupPending
		}
		return result, operationError(c.config.Provider, operationCreateMultipart, err, createSend)
	}
	if created == nil || created.UploadId == nil || *created.UploadId == "" {
		return objectstorage.UploadResult{Cleanup: objectstorage.CleanupPending}, objectstorage.NewError(objectstorage.KindInternal)
	}

	uploadID := *created.UploadId
	completed := false
	defer func() {
		if err != nil && !completed {
			result.Cleanup = objectstorage.CleanupPending
			c.cleanupMultipart(ctx, key, uploadID)
		}
	}()

	whole := crc64NVME()
	parts := make([]types.CompletedPart, 0, partCount(options.ContentLength, c.config.MultipartChunkBytes))
	remaining := options.ContentLength
	for partNumber := int32(1); remaining > 0; partNumber++ {
		length := min(remaining, c.config.MultipartChunkBytes)
		partChecksum := crc64NVME()
		partBody := newExactChecksumReader(source, length, io.MultiWriter(whole, partChecksum))
		send := &sendState{}
		part, partErr := c.sdk.UploadPart(withSendState(ctx, send), &awss3.UploadPartInput{
			Bucket:              aws.String(c.config.Bucket),
			ExpectedBucketOwner: c.expectedBucketOwner(),
			Key:                 aws.String(key),
			UploadId:            aws.String(uploadID),
			PartNumber:          aws.Int32(partNumber),
			Body:                partBody,
			ContentLength:       aws.Int64(length),
			ChecksumAlgorithm:   types.ChecksumAlgorithmCrc64nvme,
		})
		if partErr != nil {
			return result, operationError(c.config.Provider, operationMultipartStage, partErr, send)
		}
		if partErr = partBody.complete(); partErr != nil {
			return result, operationError(c.config.Provider, operationMultipartStage, partErr, send)
		}
		if part == nil || part.ETag == nil || !matchingCRC64NVME(crc64NVMEBase64(partChecksum), part.ChecksumCRC64NVME) {
			return result, objectstorage.NewError(objectstorage.KindIntegrityFailed)
		}
		parts = append(parts, types.CompletedPart{
			PartNumber:        aws.Int32(partNumber),
			ETag:              part.ETag,
			ChecksumCRC64NVME: part.ChecksumCRC64NVME,
		})
		remaining -= length
	}

	completeSend := &sendState{}
	complete, completeErr := c.sdk.CompleteMultipartUpload(withSendState(ctx, completeSend), &awss3.CompleteMultipartUploadInput{
		Bucket:              aws.String(c.config.Bucket),
		ExpectedBucketOwner: c.expectedBucketOwner(),
		Key:                 aws.String(key),
		UploadId:            aws.String(uploadID),
		ChecksumCRC64NVME:   aws.String(crc64NVMEBase64(whole)),
		ChecksumType:        types.ChecksumTypeFullObject,
		MpuObjectSize:       aws.Int64(options.ContentLength),
		MultipartUpload:     &types.CompletedMultipartUpload{Parts: parts},
	})
	if completeErr != nil {
		return result, operationError(c.config.Provider, operationComplete, completeErr, completeSend)
	}
	if complete == nil || !matchingCRC64NVME(crc64NVMEBase64(whole), complete.ChecksumCRC64NVME) || complete.ChecksumType != types.ChecksumTypeFullObject {
		return result, objectstorage.NewError(objectstorage.KindIntegrityFailed)
	}
	completed = true
	return objectstorage.UploadResult{Cleanup: objectstorage.CleanupNone}, nil
}

func (c *Client) cleanupMultipart(ctx context.Context, key, uploadID string) {
	if ctx.Err() != nil {
		return
	}
	if c.config.Provider != ProviderAmazonS3 {
		_ = c.abortMultipart(ctx, key, uploadID)
		return
	}
	for range maximumCleanupCycles {
		if !c.abortMultipart(ctx, key, uploadID) {
			return
		}
		empty, complete := c.multipartPartsEmpty(ctx, key, uploadID)
		if !complete {
			return
		}
		if empty {
			break
		}
	}
}

func (c *Client) abortMultipart(ctx context.Context, key, uploadID string) bool {
	response := &responseState{}
	_, err := c.sdk.AbortMultipartUpload(withResponseState(ctx, response), &awss3.AbortMultipartUploadInput{
		Bucket: aws.String(c.config.Bucket), ExpectedBucketOwner: c.expectedBucketOwner(), Key: aws.String(key), UploadId: aws.String(uploadID),
	})
	response.close()
	return err == nil && ctx.Err() == nil
}

func (c *Client) multipartPartsEmpty(ctx context.Context, key, uploadID string) (bool, bool) {
	var marker *string
	empty := true
	for range maximumCleanupListPages {
		response := &responseState{}
		parts, err := c.sdk.ListParts(withResponseState(ctx, response), &awss3.ListPartsInput{
			Bucket: aws.String(c.config.Bucket), ExpectedBucketOwner: c.expectedBucketOwner(), Key: aws.String(key), UploadId: aws.String(uploadID),
			MaxParts: aws.Int32(maximumCleanupListParts), PartNumberMarker: marker,
		})
		response.close()
		if err != nil || ctx.Err() != nil || parts == nil || parts.IsTruncated == nil {
			return false, false
		}
		empty = empty && len(parts.Parts) == 0
		if !*parts.IsTruncated {
			return empty, true
		}
		next := parts.NextPartNumberMarker
		if next == nil || *next == "" || marker != nil && *next == *marker {
			return false, false
		}
		marker = next
	}
	return false, false
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized feature result.
func (c *Client) admissionError(err error) error {
	if errors.Is(err, context.Canceled) {
		return objectstorage.NewError(objectstorage.KindCancelled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return objectstorage.NewError(objectstorage.KindDeadlineExceeded)
	}
	if err != nil && err.Error() == "S3 adapter admission is full" {
		return objectstorage.NewError(objectstorage.KindBusy)
	}
	return objectstorage.NewError(objectstorage.KindInvalid)
}

func uploadOperation(intent objectstorage.UploadIntent) operation {
	if intent == objectstorage.UploadCreateOnly {
		return operationPutCreateOnly
	}
	return operationPut
}
