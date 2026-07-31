package oidcjwt

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	josejson "github.com/go-jose/go-jose/v4/json"
)

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
