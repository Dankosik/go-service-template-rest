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
	Upload(ctx context.Context, key string, source io.Reader, options UploadOptions) error
	Download(ctx context.Context, key string) (Object, error)
	Metadata(ctx context.Context, key string) (Metadata, error)
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type UploadOptions struct {
	Size        int64
	ContentType string
	IfNotExists bool
}

type Object struct {
	Metadata

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
