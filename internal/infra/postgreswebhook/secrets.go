package postgreswebhook

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type (
	secretTuple    struct{ owner, destination, reference string }
	SecretManifest struct {
		revision int64
		entries  map[secretTuple][]byte
	}
)

type manifestDocument struct {
	Revision int64           `json:"revision"`
	Entries  []manifestEntry `json:"entries"`
}
type manifestEntry struct {
	OwnerScope    string `json:"owner_scope"`
	DestinationID string `json:"destination_id"`
	KeyReference  string `json:"key_reference"`
	Secret        string `json:"secret"`
}

func ParseSecretManifest(raw string) (*SecretManifest, error) {
	if raw == "" || len(raw) > MaxSecretManifestBytes {
		return nil, errors.New("parse webhook secret manifest: document size is invalid")
	}
	if err := rejectDuplicateJSONFields(raw); err != nil {
		return nil, errors.New("parse webhook secret manifest: duplicate field")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document manifestDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, errors.New("parse webhook secret manifest: invalid JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("parse webhook secret manifest: trailing data")
	}
	if document.Revision <= 0 || len(document.Entries) == 0 || len(document.Entries) > MaxSecretManifestEntries {
		return nil, errors.New("parse webhook secret manifest: positive revision and entries are required")
	}
	manifest := &SecretManifest{revision: document.Revision, entries: make(map[secretTuple][]byte, len(document.Entries))}
	bindings := make(map[string]secretTuple)
	for _, entry := range document.Entries {
		for name, value := range map[string]string{"owner_scope": entry.OwnerScope, "destination_id": entry.DestinationID, "key_reference": entry.KeyReference} {
			if err := validateToken(name, value); err != nil {
				return nil, errors.New("parse webhook secret manifest: invalid identifier")
			}
		}
		encoded, ok := strings.CutPrefix(entry.Secret, "whsec_")
		if !ok {
			return nil, errors.New("parse webhook secret manifest: secret encoding is invalid")
		}
		secret, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(secret) < 32 || len(secret) > 64 {
			return nil, errors.New("parse webhook secret manifest: secret encoding is invalid")
		}
		tuple := secretTuple{entry.OwnerScope, entry.DestinationID, entry.KeyReference}
		if _, exists := manifest.entries[tuple]; exists {
			return nil, errors.New("parse webhook secret manifest: duplicate binding")
		}
		digest := canonicalDigest(secret)
		fingerprint := string(digest[:])
		if previous, exists := bindings[fingerprint]; exists && (previous.owner != tuple.owner || previous.destination != tuple.destination) {
			return nil, errors.New("parse webhook secret manifest: key is cross-bound")
		}
		bindings[fingerprint] = tuple
		manifest.entries[tuple] = bytes.Clone(secret)
	}
	return manifest, nil
}

func rejectDuplicateJSONFields(raw string) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := inspectJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("read JSON token: %w", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			field, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("read JSON object field: %w", err)
			}
			name, ok := field.(string)
			if !ok {
				return errors.New("invalid object field")
			}
			if _, exists := seen[name]; exists {
				return errors.New("duplicate object field")
			}
			seen[name] = struct{}{}
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	_, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("close JSON value: %w", err)
	}
	return nil
}

func (m *SecretManifest) Revision() int64 {
	if m == nil {
		return 0
	}
	return m.revision
}

func (m *SecretManifest) Resolve(owner, destination, reference string) (SigningKey, error) {
	if m == nil {
		return SigningKey{}, fmt.Errorf("%w: secret manifest is required", ErrConfig)
	}
	value, ok := m.entries[secretTuple{owner, destination, reference}]
	if !ok {
		return SigningKey{}, fmt.Errorf("%w: binding not found", ErrSecretUnavailable)
	}
	return SigningKey{Reference: reference, Bytes: bytes.Clone(value)}, nil
}
