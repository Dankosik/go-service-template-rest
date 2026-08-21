package config

import (
	"errors"
	"testing"
)

//nolint:paralleltest // This test mutates process-global environment.
func TestObjectStorageConfigContract(t *testing.T) {
	for _, test := range []struct {
		name  string
		key   string
		value string
	}{
		{"missing provider", "APP__OBJECT_STORAGE__PROVIDER", ""},
		{"missing bucket", "APP__OBJECT_STORAGE__BUCKET", ""},
		{"invalid provider", "APP__OBJECT_STORAGE__PROVIDER", "other"},
		{"missing credential source", "APP__OBJECT_STORAGE__CREDENTIAL_SOURCE", ""},
		{"invalid credential source", "APP__OBJECT_STORAGE__CREDENTIAL_SOURCE", "other"},
		{"zero object limit", "APP__OBJECT_STORAGE__MAX_OBJECT_BYTES", "0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(test.key, test.value)
			_, _, err := LoadDetailed(LoadOptions{})
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("LoadDetailed() error = %v, want ErrValidate", err)
			}
		})
	}

	resetConfigEnv(t)
	t.Setenv("APP__AWS__REGION", "hostile-ambient-region")
	if _, _, err := LoadDetailed(LoadOptions{}); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("LoadDetailed() with ambient APP__AWS key error = %v, want ErrUnknownKey", err)
	}

	resetConfigEnv(t)
	t.Setenv("APP__OBJECT_STORAGE__ACCESS_KEY_ID", "removed-key")
	if _, _, err := LoadDetailed(LoadOptions{}); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("LoadDetailed() with retired credential key error = %v, want ErrUnknownKey", err)
	}
}
