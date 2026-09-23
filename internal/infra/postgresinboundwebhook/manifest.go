// Package postgresinboundwebhook implements durable Standard Webhooks ingress.
//
// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"bytes"
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	inboundmanifest "github.com/example/go-service-template-rest/internal/inboundwebhook/manifest"
	"github.com/example/go-service-template-rest/internal/webhooksecret"
)

const (
	maxSecretManifestBytes   = 1 << 20
	maxSecretManifestEntries = 4096
)

// Endpoint is the non-secret trust identity for one inbound endpoint.
type Endpoint = inboundmanifest.Endpoint

type endpointSecrets struct {
	active      []byte
	predecessor []byte
}

// EndpointManifest is the immutable non-secret endpoint snapshot.
type EndpointManifest = inboundmanifest.Endpoints

// SecretManifest is the immutable endpoint/key-reference to secret bytes.
type SecretManifest struct {
	secrets map[string]map[string][]byte
}

type secretDocument struct {
	Entries []secretEntry `json:"entries"`
}

type secretEntry struct {
	EndpointID   string `json:"endpoint_id"`
	KeyReference string `json:"key_reference"`
	Secret       string `json:"secret"`
}

// ParseSecretManifest parses environment-only secret material.
func ParseSecretManifest(raw string) (*SecretManifest, error) {
	if raw == "" {
		return &SecretManifest{secrets: map[string]map[string][]byte{}}, nil
	}
	if len(raw) > maxSecretManifestBytes {
		return nil, errors.New("parse inbound webhook secrets: document size is invalid")
	}
	var document secretDocument
	if err := json.UnmarshalRead(strings.NewReader(raw), &document, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.New("parse inbound webhook secrets: invalid JSON")
	}
	if len(document.Entries) == 0 || len(document.Entries) > maxSecretManifestEntries {
		return nil, errors.New("parse inbound webhook secrets: entries are required")
	}
	manifest := &SecretManifest{secrets: make(map[string]map[string][]byte)}
	bindings := make(map[[sha256.Size]byte]string)
	for _, entry := range document.Entries {
		if !inboundmanifest.ValidEndpointID(entry.EndpointID) {
			return nil, errors.New("parse inbound webhook secrets: invalid identifier")
		}
		if !inboundmanifest.ValidKeyReference(entry.KeyReference) {
			return nil, errors.New("parse inbound webhook secrets: invalid identifier")
		}
		secret, ok := webhooksecret.Decode(entry.Secret)
		if !ok {
			return nil, errors.New("parse inbound webhook secrets: secret encoding is invalid")
		}
		if _, exists := manifest.secrets[entry.EndpointID][entry.KeyReference]; exists {
			return nil, errors.New("parse inbound webhook secrets: duplicate binding")
		}
		digest := sha256.Sum256(secret)
		if previous, exists := bindings[digest]; exists && previous != entry.EndpointID {
			return nil, errors.New("parse inbound webhook secrets: key is cross-bound")
		}
		bindings[digest] = entry.EndpointID
		if manifest.secrets[entry.EndpointID] == nil {
			manifest.secrets[entry.EndpointID] = make(map[string][]byte)
		}
		manifest.secrets[entry.EndpointID][entry.KeyReference] = bytes.Clone(secret)
	}
	return manifest, nil
}

// BindSecrets attaches secret bytes to a non-secret endpoint snapshot.
func BindSecrets(endpoints *EndpointManifest, secrets *SecretManifest) (*TrustManifest, error) {
	if endpoints == nil || secrets == nil {
		return nil, errors.New("inbound webhook manifests are required")
	}
	endpointIDs := endpoints.IDs()
	trust := &TrustManifest{
		bindings: make(map[string]endpointBinding, len(endpointIDs)),
	}
	referenced := make(map[string]map[string]struct{})
	for _, id := range endpointIDs {
		endpoint, _ := endpoints.Lookup(id)
		keys, ok := secrets.secrets[id]
		if !ok {
			return nil, errors.New("parse inbound webhook secrets: missing referenced key")
		}
		active, ok := keys[endpoint.ActiveKeyReference]
		if !ok {
			return nil, errors.New("parse inbound webhook secrets: missing referenced key")
		}
		bound := endpointSecrets{active: bytes.Clone(active)}
		if referenced[id] == nil {
			referenced[id] = make(map[string]struct{})
		}
		referenced[id][endpoint.ActiveKeyReference] = struct{}{}
		if endpoint.PredecessorKeyReference != "" {
			predecessor, ok := keys[endpoint.PredecessorKeyReference]
			if !ok {
				return nil, errors.New("parse inbound webhook secrets: missing referenced key")
			}
			bound.predecessor = bytes.Clone(predecessor)
			referenced[id][endpoint.PredecessorKeyReference] = struct{}{}
		}
		trust.bindings[id] = endpointBinding{endpoint: endpoint, secrets: bound}
	}
	for endpointID, keys := range secrets.secrets {
		used := referenced[endpointID]
		if used == nil {
			return nil, errors.New("parse inbound webhook secrets: unused key")
		}
		for reference := range keys {
			if _, ok := used[reference]; !ok {
				return nil, errors.New("parse inbound webhook secrets: unused key")
			}
		}
	}
	return trust, nil
}

// TrustManifest is the immutable verification snapshot.
type TrustManifest struct {
	bindings map[string]endpointBinding
}

type endpointBinding struct {
	endpoint Endpoint
	secrets  endpointSecrets
}

// Lookup returns the named endpoint without exposing secret bytes.
func (m *TrustManifest) Lookup(endpointID string) (Endpoint, bool) {
	if m == nil {
		return Endpoint{}, false
	}
	binding, ok := m.bindings[endpointID]
	return binding.endpoint, ok
}

func (m *TrustManifest) secretsFor(endpointID string) (endpointSecrets, bool) {
	if m == nil {
		return endpointSecrets{}, false
	}
	binding, ok := m.bindings[endpointID]
	return binding.secrets, ok
}

func validDeliveryID(value string) bool {
	if value == "" || len(value) > 256 || !utf8.ValidString(value) {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) < 0
}

// profile:inbound-webhooks-standard:end
