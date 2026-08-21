package config

import (
	"fmt"
	"strings"
)

// ObjectStorageConfig is the immutable provider tuple. Transfer mechanics use
// adapter-owned safe defaults rather than deployment tuning.
type ObjectStorageConfig struct {
	Provider            string `koanf:"provider"`
	Endpoint            string `koanf:"endpoint"`
	Region              string `koanf:"region"`
	Bucket              string `koanf:"bucket"`
	ExpectedBucketOwner string `koanf:"expected_bucket_owner"`
	CredentialSource    string `koanf:"credential_source"`
	MaxObjectBytes      int64  `koanf:"max_object_bytes"`
}

func objectStorageDefaults() map[string]any {
	return map[string]any{
		"object_storage.provider":              "",
		"object_storage.endpoint":              "",
		"object_storage.region":                "",
		"object_storage.bucket":                "",
		"object_storage.expected_bucket_owner": "",
		"object_storage.credential_source":     "",
		"object_storage.max_object_bytes":      0,
	}
}

func validateObjectStorage(cfg *ObjectStorageConfig) error {
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.ExpectedBucketOwner = strings.TrimSpace(cfg.ExpectedBucketOwner)
	cfg.CredentialSource = strings.TrimSpace(cfg.CredentialSource)

	if cfg.Provider == "cloudflare_r2" && cfg.Region == "" {
		cfg.Region = "auto"
	}
	for _, value := range []struct {
		name  string
		value string
	}{
		{"object_storage.provider", cfg.Provider},
		{"object_storage.bucket", cfg.Bucket},
		{"object_storage.credential_source", cfg.CredentialSource},
	} {
		if value.value == "" {
			return fmt.Errorf("%w: %s is required", ErrValidate, value.name)
		}
	}
	if cfg.Provider != "amazon_s3" && cfg.Provider != "cloudflare_r2" {
		return fmt.Errorf("%w: object_storage.provider is invalid", ErrValidate)
	}
	if cfg.CredentialSource != "aws_default" && cfg.CredentialSource != "static" {
		return fmt.Errorf("%w: object_storage.credential_source is invalid", ErrValidate)
	}
	if cfg.MaxObjectBytes <= 0 {
		return fmt.Errorf("%w: object_storage.max_object_bytes must be positive", ErrValidate)
	}
	return nil
}
