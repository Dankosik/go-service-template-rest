package config

import (
	"errors"
	"strings"
	"testing"
)

//nolint:paralleltest // This test mutates process-global environment or working directory.
func TestObjectStorageConfigContract(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   string
		value string
		kind  error
	}{
		{"missing provider", "APP__OBJECT_STORAGE__PROVIDER", "", ErrValidate},
		{"missing endpoint", "APP__OBJECT_STORAGE__ENDPOINT", "", ErrValidate},
		{"missing bucket", "APP__OBJECT_STORAGE__BUCKET", "", ErrValidate},
		{"invalid provider", "APP__OBJECT_STORAGE__PROVIDER", "other", ErrValidate},
		{"missing access key", "APP__OBJECT_STORAGE__ACCESS_KEY_ID", "", ErrSecretPolicy},
		{"zero object limit", "APP__OBJECT_STORAGE__MAX_OBJECT_BYTES", "0", ErrValidate},
		{"zero operation limit", "APP__OBJECT_STORAGE__MAX_ACTIVE_OPERATIONS", "0", ErrValidate},
		{"zero operation duration", "APP__OBJECT_STORAGE__MAX_OPERATION_DURATION", "0s", ErrValidate},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(test.key, test.value)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, test.kind) {
				t.Fatalf("LoadDetailed() error = %v, want %v", err, test.kind)
			}
		})
	}

	resetConfigEnv(t)
	t.Setenv("APP__OBJECT_STORAGE__SESSION_TOKEN", "test-session-token")
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.ObjectStorage.SessionToken != "test-session-token" {
		t.Fatalf("SessionToken = %q, want environment value", cfg.ObjectStorage.SessionToken)
	}

	resetConfigEnv(t)
	t.Setenv("APP__AWS__REGION", "hostile-ambient-region")
	_, _, err = LoadDetailed(LoadOptions{})
	if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("LoadDetailed() with ambient AWS key error = %v, want ErrUnknownKey", err)
	}

	for _, key := range []string{
		"APP__OBJECT_STORAGE__CA_FILE",
		"APP__OBJECT_STORAGE__CA_BUNDLE",
		"APP__OBJECT_STORAGE__ROOT_CA",
	} {
		resetConfigEnv(t)
		t.Setenv(key, "/tmp/host-controlled-ca.pem")
		if _, _, err := LoadDetailed(LoadOptions{}); !errors.Is(err, ErrUnknownKey) {
			t.Fatalf("LoadDetailed() with %s error = %v, want ErrUnknownKey", key, err)
		}
	}
}

func TestStaticCredentialSourcePolicy(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"access_key_id", "secret_access_key", "session_token"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			resetConfigEnv(t)
			const canary = "object-storage-credential-canary"
			path := writeTempConfig(t, "object_storage:\n  "+key+": "+canary+"\n")
			_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
			if !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("LoadDetailed() error = %v, want ErrSecretPolicy", err)
			}
			if strings.Contains(err.Error(), canary) {
				t.Fatalf("LoadDetailed() leaked credential value: %v", err)
			}
		})
	}

	resetConfigEnv(t)
	path := writeTempConfig(t, "object_storage:\n  access_key_id: \"\"\n  secret_access_key: \"\"\n  session_token: \"\"\n")
	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err != nil {
		t.Fatalf("LoadDetailed() error = %v, want empty placeholders accepted", err)
	}
}
