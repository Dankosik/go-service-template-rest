package s3

import "time"

func validConfig(provider Provider) Config {
	cfg := Config{
		Provider:                provider,
		Region:                  "us-east-1",
		Bucket:                  "examplebucket",
		AccessKeyID:             "test-access-key",
		SecretAccessKey:         "test-secret-key",
		MaxObjectBytes:          10 << 20,
		MultipartChunkBytes:     5 << 20,
		MaxActiveOperations:     2,
		MaxOperationDuration:    time.Second,
		MaxPresignLifetime:      time.Minute,
		MaxResponseHeaderBytes:  1024,
		MaxControlResponseBytes: 1024,
	}
	if provider == ProviderCloudflare {
		cfg.Endpoint = "https://account.r2.cloudflarestorage.com"
		cfg.Region = "auto"
	} else {
		cfg.Endpoint = "https://s3.us-east-1.amazonaws.com"
	}
	required, ok := cfg.requiredMemory()
	if !ok {
		panic("valid S3 test configuration must have a memory envelope")
	}
	cfg.MaxWorkingMemoryBytes = required
	return cfg
}
