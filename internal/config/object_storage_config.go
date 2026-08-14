package config

import (
	"fmt"
	"strings"
	"time"
)

// ObjectStorageConfig is the static, immutable S3-compatible adapter tuple.
// It has no executable defaults: selecting the profile requires deployment to
// provide one complete authority, credential, and resource envelope.
type ObjectStorageConfig struct {
	Provider        string `koanf:"provider"`
	Endpoint        string `koanf:"endpoint"`
	Region          string `koanf:"region"`
	Bucket          string `koanf:"bucket"`
	AccessKeyID     string `koanf:"access_key_id"`
	SecretAccessKey string `koanf:"secret_access_key"`
	SessionToken    string `koanf:"session_token"`

	MaxObjectBytes          int64         `koanf:"max_object_bytes"`
	MultipartChunkBytes     int64         `koanf:"multipart_chunk_bytes"`
	MaxActiveOperations     int           `koanf:"max_active_operations"`
	MaxOperationDuration    time.Duration `koanf:"max_operation_duration"`
	MaxPresignLifetime      time.Duration `koanf:"max_presign_lifetime"`
	MaxResponseHeaderBytes  int64         `koanf:"max_response_header_bytes"`
	MaxControlResponseBytes int64         `koanf:"max_control_response_bytes"`
	MaxWorkingMemoryBytes   int64         `koanf:"max_working_memory_bytes"`
}

func objectStorageDefaults() map[string]any {
	return map[string]any{
		"object_storage.provider":                   "",
		"object_storage.endpoint":                   "",
		"object_storage.region":                     "",
		"object_storage.bucket":                     "",
		"object_storage.access_key_id":              "",
		"object_storage.secret_access_key":          "",
		"object_storage.session_token":              "",
		"object_storage.max_object_bytes":           0,
		"object_storage.multipart_chunk_bytes":      0,
		"object_storage.max_active_operations":      0,
		"object_storage.max_operation_duration":     "0s",
		"object_storage.max_presign_lifetime":       "0s",
		"object_storage.max_response_header_bytes":  0,
		"object_storage.max_control_response_bytes": 0,
		"object_storage.max_working_memory_bytes":   0,
	}
}

func validateObjectStorage(cfg *ObjectStorageConfig) error {
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.AccessKeyID = strings.TrimSpace(cfg.AccessKeyID)
	cfg.SecretAccessKey = strings.TrimSpace(cfg.SecretAccessKey)
	cfg.SessionToken = strings.TrimSpace(cfg.SessionToken)

	for _, value := range []struct {
		name  string
		value string
	}{
		{"object_storage.provider", cfg.Provider},
		{"object_storage.endpoint", cfg.Endpoint},
		{"object_storage.region", cfg.Region},
		{"object_storage.bucket", cfg.Bucket},
	} {
		if value.value == "" {
			return fmt.Errorf("%w: %s is required", ErrValidate, value.name)
		}
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return fmt.Errorf("%w: object_storage static credentials are required", ErrSecretPolicy)
	}
	if cfg.Provider != "amazon_s3" && cfg.Provider != "cloudflare_r2" {
		return fmt.Errorf("%w: object_storage.provider is invalid", ErrValidate)
	}
	for _, value := range []struct {
		name  string
		value int64
	}{
		{"object_storage.max_object_bytes", cfg.MaxObjectBytes},
		{"object_storage.multipart_chunk_bytes", cfg.MultipartChunkBytes},
		{"object_storage.max_response_header_bytes", cfg.MaxResponseHeaderBytes},
		{"object_storage.max_control_response_bytes", cfg.MaxControlResponseBytes},
		{"object_storage.max_working_memory_bytes", cfg.MaxWorkingMemoryBytes},
	} {
		if value.value <= 0 {
			return fmt.Errorf("%w: %s must be positive", ErrValidate, value.name)
		}
	}
	if cfg.MaxActiveOperations <= 0 || cfg.MaxOperationDuration <= 0 || cfg.MaxPresignLifetime <= 0 {
		return fmt.Errorf("%w: object_storage bounds must be positive", ErrValidate)
	}
	return nil
}
