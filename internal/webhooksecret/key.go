// Package webhooksecret owns the shared Standard Webhooks secret format used by
// the independently removable inbound and outbound webhook adapters.
package webhooksecret

import (
	"encoding/base64"
	"strings"
)

const (
	minKeyBytes = 32
	maxKeyBytes = 64
)

// Decode accepts a whsec_ prefixed, padded standard-base64 secret.
func Decode(value string) ([]byte, bool) {
	encoded, ok := strings.CutPrefix(value, "whsec_")
	if !ok {
		return nil, false
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || !ValidKey(key) {
		return nil, false
	}
	return key, true
}

// ValidKey reports whether raw signing key bytes meet the shared length rule.
func ValidKey(key []byte) bool {
	return len(key) >= minKeyBytes && len(key) <= maxKeyBytes
}
