package s3

import (
	"io"
	"time"
)

func uploadSource(reader io.Reader) io.ReadCloser { return io.NopCloser(reader) }

func validConfig(provider Provider) Config {
	cfg := Config{
		Provider:                provider,
		Region:                  "us-east-1",
		Bucket:                  "examplebucket",
		AccessKeyID:             "test-access-key",
		SecretAccessKey:         "test-secret-key",
		SessionToken:            "test-session-token",
		MaxObjectBytes:          10 << 20,
		MultipartChunkBytes:     5 << 20,
		MaxActiveOperations:     2,
		MaxOperationDuration:    time.Second,
		MaxPresignLifetime:      time.Minute,
		MaxResponseHeaderBytes:  1024,
		MaxControlResponseBytes: 1024,
	}
	if provider == ProviderCloudflare {
		cfg.Endpoint = "https://0123456789abcdef0123456789abcdef.r2.cloudflarestorage.com"
		cfg.Region = "auto"
	} else {
		cfg.Endpoint = "https://s3.us-east-1.amazonaws.com"
		cfg.ExpectedBucketOwner = "123456789012"
	}
	required, ok := cfg.requiredMemory()
	if !ok {
		panic("valid S3 test configuration must have a memory envelope")
	}
	cfg.MaxWorkingMemoryBytes = required
	return cfg
}
