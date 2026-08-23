// Package postgresinboundwebhook implements durable Standard Webhooks ingress.
//
// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxEndpointManifestBytes   = 1 << 20
	maxSecretManifestBytes     = 1 << 20
	maxEndpointManifestEntries = 4096
	maxSecretManifestEntries   = 4096
	maxEndpointIDBytes         = 64
	maxKeyReferenceBytes       = 64
	minSecretBytes             = 32
	maxSecretBytes             = 64
)

// Endpoint is the non-secret trust identity for one inbound endpoint.
type Endpoint struct {
	ID                      string
	ActiveKeyReference      string
	PredecessorKeyReference string
}

type endpointSecrets struct {
	active      []byte
	predecessor []byte
}

// EndpointManifest is the immutable non-secret endpoint snapshot.
type EndpointManifest struct {
	endpoints map[string]Endpoint
}

// SecretManifest is the immutable endpoint/key-reference to secret bytes.
type SecretManifest struct {
	secrets map[string]map[string][]byte
}

type endpointDocument struct {
	Endpoints []endpointEntry `json:"endpoints"`
}

type endpointEntry struct {
	EndpointID              string `json:"endpoint_id"`
	ActiveKeyReference      string `json:"active_key_reference"`
	PredecessorKeyReference string `json:"predecessor_key_reference,omitempty"`
}

type secretDocument struct {
	Entries []secretEntry `json:"entries"`
}

type secretEntry struct {
	EndpointID   string `json:"endpoint_id"`
	KeyReference string `json:"key_reference"`
	Secret       string `json:"secret"`
}

// ParseEndpointManifest parses the non-secret endpoint document.
func ParseEndpointManifest(raw string) (*EndpointManifest, error) {
	if raw == "" {
		return &EndpointManifest{endpoints: map[string]Endpoint{}}, nil
	}
	if len(raw) > maxEndpointManifestBytes {
		return nil, errors.New("parse inbound webhook endpoints: document size is invalid")
	}
	var document endpointDocument
	if err := json.UnmarshalRead(strings.NewReader(raw), &document, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.New("parse inbound webhook endpoints: invalid JSON")
	}
	if len(document.Endpoints) == 0 || len(document.Endpoints) > maxEndpointManifestEntries {
		return nil, errors.New("parse inbound webhook endpoints: entries are required")
	}
	manifest := &EndpointManifest{endpoints: make(map[string]Endpoint, len(document.Endpoints))}
	for _, entry := range document.Endpoints {
		if err := validateEndpointID(entry.EndpointID); err != nil {
			return nil, errors.New("parse inbound webhook endpoints: invalid identifier")
		}
		if err := validateKeyReference(entry.ActiveKeyReference); err != nil {
			return nil, errors.New("parse inbound webhook endpoints: invalid identifier")
		}
		if entry.PredecessorKeyReference != "" {
			if err := validateKeyReference(entry.PredecessorKeyReference); err != nil {
				return nil, errors.New("parse inbound webhook endpoints: invalid predecessor key")
			}
			if entry.PredecessorKeyReference == entry.ActiveKeyReference {
				return nil, errors.New("parse inbound webhook endpoints: predecessor key must differ from active key")
			}
		}
		if _, exists := manifest.endpoints[entry.EndpointID]; exists {
			return nil, errors.New("parse inbound webhook endpoints: duplicate endpoint")
		}
		manifest.endpoints[entry.EndpointID] = Endpoint{
			ID:                      entry.EndpointID,
			ActiveKeyReference:      entry.ActiveKeyReference,
			PredecessorKeyReference: entry.PredecessorKeyReference,
		}
	}
	return manifest, nil
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
		if err := validateEndpointID(entry.EndpointID); err != nil {
			return nil, errors.New("parse inbound webhook secrets: invalid identifier")
		}
		if err := validateKeyReference(entry.KeyReference); err != nil {
			return nil, errors.New("parse inbound webhook secrets: invalid identifier")
		}
		encoded, ok := strings.CutPrefix(entry.Secret, "whsec_")
		if !ok {
			return nil, errors.New("parse inbound webhook secrets: secret encoding is invalid")
		}
		secret, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(secret) < minSecretBytes || len(secret) > maxSecretBytes {
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
	trust := &TrustManifest{
		endpoints: make(map[string]Endpoint, len(endpoints.endpoints)),
		secrets:   make(map[string]endpointSecrets, len(endpoints.endpoints)),
	}
	referenced := make(map[string]map[string]struct{})
	for id, endpoint := range endpoints.endpoints {
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
		trust.endpoints[id] = endpoint
		trust.secrets[id] = bound
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
	endpoints map[string]Endpoint
	secrets   map[string]endpointSecrets
}

// Lookup returns the named endpoint without exposing secret bytes.
func (m *TrustManifest) Lookup(endpointID string) (Endpoint, bool) {
	if m == nil {
		return Endpoint{}, false
	}
	endpoint, ok := m.endpoints[endpointID]
	return endpoint, ok
}

func (m *TrustManifest) secretsFor(endpointID string) (endpointSecrets, bool) {
	if m == nil {
		return endpointSecrets{}, false
	}
	secrets, ok := m.secrets[endpointID]
	return secrets, ok
}

// IDs returns configured endpoint identifiers.
func (m *EndpointManifest) IDs() []string {
	if m == nil {
		return nil
	}
	ids := make([]string, 0, len(m.endpoints))
	for id := range m.endpoints {
		ids = append(ids, id)
	}
	return ids
}

func validateEndpointID(value string) error {
	if value == "" || len(value) > maxEndpointIDBytes || !utf8.ValidString(value) {
		return errors.New("invalid endpoint id")
	}
	for _, r := range value {
		if r > unicode.MaxASCII || (r != '_' && r != '-' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9')) {
			return errors.New("invalid endpoint id")
		}
	}
	return nil
}

func validateKeyReference(value string) error {
	if value == "" || len(value) > maxKeyReferenceBytes || !utf8.ValidString(value) {
		return errors.New("invalid key reference")
	}
	if strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return errors.New("invalid key reference")
	}
	return nil
}

func validDeliveryID(value string) bool {
	if value == "" || len(value) > 256 || !utf8.ValidString(value) {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) < 0
}

// profile:inbound-webhooks-standard:end
