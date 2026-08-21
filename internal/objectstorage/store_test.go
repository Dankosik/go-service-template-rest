package objectstorage

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateKey(t *testing.T) {
	for _, key := range []string{"object", "path/to object", "café", strings.Repeat("x", 1024)} {
		if err := ValidateKey(key); err != nil {
			t.Fatalf("ValidateKey(%q) error = %v", key, err)
		}
	}
	for _, key := range []string{"", strings.Repeat("x", 1025), string([]byte{0xff})} {
		if err := ValidateKey(key); !errors.Is(err, ErrInvalid) {
			t.Fatalf("ValidateKey(%q) error = %v, want ErrInvalid", key, err)
		}
	}
}
