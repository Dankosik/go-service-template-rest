package s3

import (
	"context"
	"encoding/base64"
	"errors"
	"hash"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/example/go-service-template-rest/internal/objectstorage"
)

// Download returns a bounded body whose successful EOF proves its checksum.
//
//nolint:contextcheck,wrapcheck // The retained body owns ctx until terminal release, and object-storage errors are the closed, sanitized result.
func (c *Client) Download(ctx context.Context, key string) (result objectstorage.Download, err error) {
	call := c.telemetry.begin(ctx, telemetryOperationDownload)
	if err := objectstorage.ValidateKey(key); err != nil {
		call.finish(err, 0)
		return result, err
	}
	effective, release, err := c.admit(ctx, call)
	if err != nil {
		err = c.admissionError(err)
		call.finish(err, 0)
		return result, err
	}
	if err := effective.Err(); err != nil {
		release()
		err = c.admissionError(err)
		call.finish(err, 0)
		return result, err
	}

	out, err := c.sdk.GetObject(effective, &awss3.GetObjectInput{
		Bucket:       aws.String(c.config.Bucket),
		Key:          aws.String(key),
		ChecksumMode: types.ChecksumModeEnabled,
	})
	if err != nil {
		release()
		err = c.downloadError(err)
		call.finish(err, 0)
		return result, err
	}
	if out == nil || out.Body == nil || out.ContentLength != nil && *out.ContentLength > c.config.MaxObjectBytes ||
		!validCRC64NVME(out.ChecksumCRC64NVME) || out.ChecksumType != types.ChecksumTypeFullObject {
		if out != nil && out.Body != nil {
			_ = out.Body.Close()
		}
		release()
		if out != nil && out.ContentLength != nil && *out.ContentLength > c.config.MaxObjectBytes {
			err = objectstorage.NewError(objectstorage.KindTooLarge)
			call.finish(err, 0)
			return result, err
		}
		err = objectstorage.NewError(objectstorage.KindIntegrityFailed)
		call.finish(err, 0)
		return result, err
	}

	size := int64(0)
	if out.ContentLength != nil && *out.ContentLength >= 0 {
		size = *out.ContentLength
	}
	return objectstorage.Download{
		Body: newDownloadBody(effective, out.Body, c.config.MaxObjectBytes, *out.ChecksumCRC64NVME, release, call),
		Size: size, ContentType: aws.ToString(out.ContentType), LastModified: aws.ToTime(out.LastModified),
	}, nil
}

func validCRC64NVME(value *string) bool {
	if value == nil || *value == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(*value)
	return err == nil && len(decoded) == 8
}

type downloadBody struct {
	//nolint:containedctx // The body owns ctx until EOF, error, or Close releases the admission token.
	ctx       context.Context
	body      io.ReadCloser
	remaining int64
	expected  string
	checksum  hash.Hash64
	release   func()
	call      *operationCall
	bytes     int64

	readMu sync.Mutex
	mu     sync.Mutex
	done   bool
}

func newDownloadBody(ctx context.Context, body io.ReadCloser, limit int64, expected string, release func(), call *operationCall) *downloadBody {
	return &downloadBody{ctx: ctx, body: body, remaining: limit, expected: expected, checksum: crc64NVME(), release: release, call: call}
}

func (b *downloadBody) Read(p []byte) (int, error) {
	b.readMu.Lock()
	defer b.readMu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	b.mu.Lock()
	if b.done {
		b.mu.Unlock()
		return 0, io.EOF
	}
	if err := b.ctx.Err(); err != nil {
		b.mu.Unlock()
		mapped := downloadContextError(err)
		b.finish(true, mapped)
		return 0, mapped
	}
	remaining := b.remaining
	b.mu.Unlock()
	if remaining == 0 {
		var probe [1]byte
		n, err := b.body.Read(probe[:])
		if n > 0 {
			mapped := objectstorage.NewError(objectstorage.KindTooLarge)
			b.finish(true, mapped)
			return 0, mapped //nolint:wrapcheck // Reader callers receive the closed, sanitized size error.
		}
		return b.terminal(0, err)
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := b.body.Read(p)
	b.mu.Lock()
	if b.done {
		b.mu.Unlock()
		return 0, io.EOF
	}
	if n > 0 {
		b.remaining -= int64(n)
		b.bytes += int64(n)
		_, _ = b.checksum.Write(p[:n])
	}
	if n == 0 && err == nil {
		err = io.ErrNoProgress
	}
	if err == nil {
		b.mu.Unlock()
		return n, nil
	}
	b.mu.Unlock()
	return b.terminal(n, err)
}

//nolint:wrapcheck // Object-storage errors preserve the closed, sanitized read result.
func (b *downloadBody) terminal(n int, err error) (int, error) {
	b.mu.Lock()
	actual := crc64NVMEBase64(b.checksum)
	b.mu.Unlock()
	if errors.Is(err, io.EOF) && actual == b.expected {
		b.finish(false, nil)
		return n, io.EOF
	}
	if actual != b.expected && (errors.Is(err, io.EOF) || deferredChecksumError(err)) {
		mapped := objectstorage.NewError(objectstorage.KindIntegrityFailed)
		b.finish(false, mapped)
		return n, mapped
	}
	mapped := b.readError(err)
	b.finish(true, mapped)
	return n, mapped
}

func (b *downloadBody) readError(err error) error {
	if b.ctx.Err() != nil {
		return downloadContextError(b.ctx.Err())
	}
	return stableError(err)
}

// The pinned SDK exposes its deferred checksum failure as an unexported error.
func deferredChecksumError(err error) bool {
	return strings.Contains(err.Error(), "checksum did not match: algorithm CRC64NVME")
}

func (b *downloadBody) Close() error {
	b.finish(true, objectstorage.NewError(objectstorage.KindCancelled))
	return nil
}

func (b *downloadBody) finish(closeBody bool, err error) {
	b.mu.Lock()
	if b.done {
		b.mu.Unlock()
		return
	}
	b.done = true
	bytes := b.bytes
	b.mu.Unlock()
	if closeBody {
		_ = b.body.Close()
	}
	b.release()
	b.call.finish(err, bytes)
}

func (c *Client) downloadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return downloadContextError(err)
	}
	return stableError(err)
}

//nolint:wrapcheck // Object-storage errors are the closed, sanitized context result.
func downloadContextError(err error) error {
	if errors.Is(err, context.Canceled) {
		return objectstorage.NewError(objectstorage.KindCancelled)
	}
	return objectstorage.NewError(objectstorage.KindDeadlineExceeded)
}
