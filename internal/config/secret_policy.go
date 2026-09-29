package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/knadh/koanf/v2"
)

// enforceSecretSourcePolicy refuses a config file that carries a non-empty
// secret-like value; secrets reach config only through the environment.
func enforceSecretSourcePolicy(k *koanf.Koanf, path string) error {
	keys := k.Keys()
	slices.Sort(keys)
	for _, key := range keys {
		if !isSecretLikeConfigKey(key) {
			continue
		}
		if hasNonEmptyConfigValue(k.Get(key)) {
			return fmt.Errorf("%w: secret-like key %q is not allowed in config file %q", ErrSecretPolicy, key, path)
		}
	}
	return nil
}

func isSecretLikeConfigKey(key string) bool {
	segments := configKeySegments(strings.ToLower(strings.TrimSpace(key)))
	for i, segment := range segments {
		switch segment {
		case "password", "secret", "secrets", "authorization", "dsn":
			return true
		case "token":
			// token_profile and token_url name a JWT profile and an endpoint, not
			// a credential.
			if i+1 == len(segments) || (segments[i+1] != "profile" && segments[i+1] != "url") {
				return true
			}
		case "key":
			// A bare key segment is usually a lookup key or key reference; only
			// api_key and private_key name the secret itself.
			if i > 0 && (segments[i-1] == "api" || segments[i-1] == "private") {
				return true
			}
		case "headers":
			// OTLP exporter headers carry the collector's authorization; other
			// header maps do not.
			if i > 0 && segments[i-1] == "otlp" {
				return true
			}
		}
	}
	return false
}

func configKeySegments(key string) []string {
	return strings.FieldsFunc(key, func(r rune) bool {
		switch r {
		case '.', '_', '-':
			return true
		}
		return false
	})
}

func hasNonEmptyConfigValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value)) != ""
	}
}
