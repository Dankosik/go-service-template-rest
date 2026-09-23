package postgreswebhook

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/webhooksecret"
)

type secretTuple struct {
	owner     string
	receiver  string
	reference string
}

// secretPrincipal is the receiver identity one secret may sign for.
type secretPrincipal struct {
	owner    string
	receiver string
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
	if raw == "" || len(raw) > webhooksecret.MaxManifestBytes {
		return nil, errors.New("parse webhook secret manifest: document size is invalid")
	}
	var document secretDocument
	if err := json.UnmarshalRead(strings.NewReader(raw), &document, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.New("parse webhook secret manifest: invalid JSON")
	}
	if len(document.Entries) == 0 || len(document.Entries) > webhooksecret.MaxManifestEntries {
		return nil, errors.New("parse webhook secret manifest: entries are required")
	}
	manifest := &SecretManifest{entries: make(map[secretTuple][]byte, len(document.Entries))}
	var bindings webhooksecret.Bindings[secretPrincipal]
	for _, entry := range document.Entries {
		for name, value := range map[string]string{
			ownerScopeField: entry.OwnerScope, receiverIDField: entry.ReceiverID, "key_reference": entry.KeyReference,
		} {
			if err := validateToken(name, value); err != nil {
				return nil, errors.New("parse webhook secret manifest: invalid identifier")
			}
		}
		secret, ok := webhooksecret.Decode(entry.Secret)
		if !ok {
			return nil, errors.New("parse webhook secret manifest: secret encoding is invalid")
		}
		tuple := secretTuple{owner: entry.OwnerScope, receiver: entry.ReceiverID, reference: entry.KeyReference}
		if _, exists := manifest.entries[tuple]; exists {
			return nil, errors.New("parse webhook secret manifest: duplicate binding")
		}
		if !bindings.Bind(secret, secretPrincipal{owner: tuple.owner, receiver: tuple.receiver}) {
			return nil, errors.New("parse webhook secret manifest: key is cross-bound")
		}
		manifest.entries[tuple] = bytes.Clone(secret)
	}
	return manifest, nil
}

func (m *SecretManifest) resolve(owner, receiver, reference string) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("%w: secret manifest is required", ErrConfig)
	}
	value, ok := m.entries[secretTuple{owner: owner, receiver: receiver, reference: reference}]
	if !ok {
		return nil, fmt.Errorf("%w: binding not found", errSecretUnavailable)
	}
	return bytes.Clone(value), nil
}
