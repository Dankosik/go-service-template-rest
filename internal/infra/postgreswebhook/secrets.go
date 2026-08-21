package postgreswebhook

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
)

const (
	MaxSecretManifestBytes   = 1 << 20
	MaxSecretManifestEntries = 4096
)

type secretTuple struct {
	owner     string
	receiver  string
	reference string
}

type SecretManifest struct {
	entries map[secretTuple][]byte
}

type secretDocument struct {
	Entries []secretEntry `json:"entries"`
}

type secretEntry struct {
	OwnerScope   string `json:"owner_scope"`
	ReceiverID   string `json:"receiver_id"`
	KeyReference string `json:"key_reference"`
	Secret       string `json:"secret"`
}

func ParseSecretManifest(raw string) (*SecretManifest, error) {
	if raw == "" || len(raw) > MaxSecretManifestBytes {
		return nil, errors.New("parse webhook secret manifest: document size is invalid")
	}
	var document secretDocument
	if err := json.UnmarshalRead(strings.NewReader(raw), &document, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.New("parse webhook secret manifest: invalid JSON")
	}
	if len(document.Entries) == 0 || len(document.Entries) > MaxSecretManifestEntries {
		return nil, errors.New("parse webhook secret manifest: entries are required")
	}
	manifest := &SecretManifest{entries: make(map[secretTuple][]byte, len(document.Entries))}
	bindings := make(map[[sha256.Size]byte]secretTuple)
	for _, entry := range document.Entries {
		for name, value := range map[string]string{
			ownerScopeField: entry.OwnerScope, receiverIDField: entry.ReceiverID, "key_reference": entry.KeyReference,
		} {
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
		tuple := secretTuple{owner: entry.OwnerScope, receiver: entry.ReceiverID, reference: entry.KeyReference}
		if _, exists := manifest.entries[tuple]; exists {
			return nil, errors.New("parse webhook secret manifest: duplicate binding")
		}
		digest := sha256.Sum256(secret)
		if previous, exists := bindings[digest]; exists && (previous.owner != tuple.owner || previous.receiver != tuple.receiver) {
			return nil, errors.New("parse webhook secret manifest: key is cross-bound")
		}
		bindings[digest] = tuple
		manifest.entries[tuple] = bytes.Clone(secret)
	}
	return manifest, nil
}

func (m *SecretManifest) Resolve(owner, receiver, reference string) (SigningKey, error) {
	if m == nil {
		return SigningKey{}, fmt.Errorf("%w: secret manifest is required", ErrConfig)
	}
	value, ok := m.entries[secretTuple{owner: owner, receiver: receiver, reference: reference}]
	if !ok {
		return SigningKey{}, fmt.Errorf("%w: binding not found", ErrSecretUnavailable)
	}
	return SigningKey{Reference: reference, Bytes: bytes.Clone(value)}, nil
}
