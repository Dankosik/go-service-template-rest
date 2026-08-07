package oidcjwt

// The two decoders every parse in this package goes through: unpadded base64url
// that must be the only encoding of its bytes, and JSON that refuses a repeated
// member. They live together because they answer the same question — whether the
// input has exactly one legal reading — and neither is safe to replace with its
// lenient standard-library equivalent, which takes the last value for a repeated
// key instead of rejecting it.
//
// Unknown members are accepted, and have to be: a provider may put any claim it
// likes in an access token. So a member this package must refuse is one it
// declares and tests for presence — that is why parseProtectedHeader names crit,
// b64, jku, x5u, and jwk rather than relying on the decoder to reject them.

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	josejson "github.com/go-jose/go-jose/v4/json"
)

// decodeCanonicalSegment decodes one unpadded base64url segment and requires it
// to be the only encoding of its bytes.
//
// Re-encoding and comparing is the check that matters: base64 leaves the unused
// bits of a final character free, so several distinct segments decode to the
// same bytes. Accepting all of them would let one token be rewritten into
// several forms that verify alike, which defeats any downstream identity or
// replay decision keyed on the compact text.
func decodeCanonicalSegment(segment string) ([]byte, error) {
	if segment == "" || strings.Contains(segment, "=") {
		return nil, errors.New("invalid compact segment")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != segment {
		return nil, errors.New("invalid compact segment")
	}
	return decoded, nil
}

func strictUnmarshal(data []byte, destination any) error {
	if len(data) == 0 {
		return errors.New("empty JSON")
	}
	if err := josejson.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("decode strict JSON: %w", err)
	}
	return nil
}
