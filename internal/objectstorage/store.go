package objectstorage

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// Store is the complete provider-neutral object-storage feature surface.
//
//nolint:iface // Consumers compose the provider-neutral port outside this package.
type Store interface {
	// Upload takes ownership of source. Close must promptly unblock Read;
	// Upload closes source exactly once before it returns.
	Upload(ctx context.Context, key string, source io.ReadCloser, options UploadOptions) (UploadResult, error)
	Download(ctx context.Context, key string) (Download, error)
	Metadata(ctx context.Context, key string) (Metadata, error)
	Delete(ctx context.Context, key string) error
	PresignGET(ctx context.Context, key string, ttl time.Duration) (PresignedGET, error)
}

type UploadIntent string

const (
	UploadCreateOnly UploadIntent = "create_only"
	UploadReplace    UploadIntent = "replace"
)

type UploadOptions struct {
	ContentLength int64
	ContentType   string
	Intent        UploadIntent
}

type CleanupDisposition string

const (
	CleanupNone    CleanupDisposition = "none"
	CleanupPending CleanupDisposition = "pending"
)

type UploadResult struct {
	Cleanup CleanupDisposition
}

type Download struct {
	Body         io.ReadCloser
	Size         int64
	ContentType  string
	LastModified time.Time
}

type Metadata struct {
	Size         int64
	ContentType  string
	LastModified time.Time
}

type PresignedGET struct {
	Method             string
	URL                string
	Headers            http.Header
	SignatureExpiresAt time.Time
}

// ValidateKey rejects keys outside the common Amazon S3 and Cloudflare R2 domain.
func ValidateKey(key string) error {
	if key == "" || len(key) > 1024 || key == "soap" || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return NewError(KindInvalid)
	}

	segmentStart := 0
	for index := range len(key) {
		character := key[index]
		if !validKeyCharacter(character) {
			return NewError(KindInvalid)
		}
		if character != '/' {
			continue
		}
		if invalidKeySegment(key[segmentStart:index]) {
			return NewError(KindInvalid)
		}
		segmentStart = index + 1
	}
	if invalidKeySegment(key[segmentStart:]) {
		return NewError(KindInvalid)
	}
	return nil
}

func validKeyCharacter(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		character == '-' || character == '_' || character == '.' || character == '~' || character == '/'
}

func invalidKeySegment(segment string) bool {
	return segment == "" || segment == "." || segment == ".."
}
