package objectstorage

import (
	"context"
	"io"
	"time"
	"unicode/utf8"
)

// Store is the complete provider-neutral object-storage feature surface.
//
//nolint:iface // Consumers compose the provider-neutral port outside this package.
type Store interface {
	// Upload may return ErrOutcomeUnknown after the provider accepted the write;
	// establish the outcome before treating it as absent or retrying on that basis.
	Upload(ctx context.Context, key string, source io.Reader, options UploadOptions) error
	// Download returns a live streaming body. The caller keeps ctx valid while
	// reading and must close Body; Read can return an error after partial bytes.
	Download(ctx context.Context, key string) (Object, error)
	Metadata(ctx context.Context, key string) (Metadata, error)
	// Delete may return ErrOutcomeUnknown after the provider applied the delete;
	// establish the outcome before treating it as unapplied or retrying on that basis.
	Delete(ctx context.Context, key string) error
	// PresignGet accepts a whole-second ttl from one second through seven days;
	// an invalid ttl returns ErrInvalid.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type UploadOptions struct {
	// Size is the declared object length in bytes. Zero is an empty object;
	// negative and unknown lengths are unsupported.
	Size        int64
	ContentType string
	// IfNotExists requests create-only upload. The shipped S3 adapter supports
	// it only for single-request uploads of 8 MiB or less.
	IfNotExists bool
}

type Object struct {
	Metadata

	// Body is the stream returned by Download. Close releases resources but does
	// not validate unread data.
	Body io.ReadCloser
}

type Metadata struct {
	Size         int64
	ContentType  string
	LastModified time.Time
}

// ValidateKey enforces only the common provider boundary. Features own narrower
// namespaces and path conventions.
func ValidateKey(key string) error {
	if key == "" || len(key) > 1024 || !utf8.ValidString(key) {
		return ErrInvalid
	}
	return nil
}
