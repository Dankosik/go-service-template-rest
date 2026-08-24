package postgreswebhook

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
)

const (
	maxEndpointManifestBytes   = 1 << 20
	maxEndpointManifestEntries = 4096
)

type endpointKey struct {
	owner    string
	receiver string
}

type endpoint struct {
	OwnerScope              string
	ReceiverID              string
	Generation              int64
	URL                     string
	ActiveKeyReference      string
	PredecessorKeyReference string
}

type EndpointManifest struct {
	entries map[endpointKey]endpoint
}

type endpointDocument struct {
	Endpoints []endpointEntry `json:"endpoints"`
}

type endpointEntry struct {
	OwnerScope              string `json:"owner_scope"`
	ReceiverID              string `json:"receiver_id"`
	Generation              int64  `json:"generation"`
	URL                     string `json:"url"`
	ActiveKeyReference      string `json:"active_key_reference"`
	PredecessorKeyReference string `json:"predecessor_key_reference,omitempty"`
}

func ParseEndpointManifest(raw string) (*EndpointManifest, error) {
	if raw == "" || len(raw) > maxEndpointManifestBytes {
		return nil, errors.New("parse webhook endpoints: document size is invalid")
	}
	var document endpointDocument
	if err := json.UnmarshalRead(strings.NewReader(raw), &document, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.New("parse webhook endpoints: invalid JSON")
	}
	if len(document.Endpoints) == 0 || len(document.Endpoints) > maxEndpointManifestEntries {
		return nil, errors.New("parse webhook endpoints: entries are required")
	}
	manifest := &EndpointManifest{entries: make(map[endpointKey]endpoint, len(document.Endpoints))}
	for _, entry := range document.Endpoints {
		for name, value := range map[string]string{
			ownerScopeField: entry.OwnerScope, receiverIDField: entry.ReceiverID,
			"active_key_reference": entry.ActiveKeyReference,
		} {
			if err := validateToken(name, value); err != nil {
				return nil, errors.New("parse webhook endpoints: invalid identifier")
			}
		}
		if entry.PredecessorKeyReference != "" {
			if err := validateToken("predecessor_key_reference", entry.PredecessorKeyReference); err != nil {
				return nil, errors.New("parse webhook endpoints: invalid predecessor key")
			}
			if entry.PredecessorKeyReference == entry.ActiveKeyReference {
				return nil, errors.New("parse webhook endpoints: predecessor key must differ from active key")
			}
		}
		if entry.Generation <= 0 {
			return nil, errors.New("parse webhook endpoints: positive generation is required")
		}
		parsed, err := parseWebhookURL(entry.URL)
		if err != nil {
			return nil, err
		}
		key := endpointKey{owner: entry.OwnerScope, receiver: entry.ReceiverID}
		if _, exists := manifest.entries[key]; exists {
			return nil, errors.New("parse webhook endpoints: duplicate receiver")
		}
		manifest.entries[key] = endpoint{
			OwnerScope: entry.OwnerScope, ReceiverID: entry.ReceiverID, Generation: entry.Generation,
			URL: parsed.String(), ActiveKeyReference: entry.ActiveKeyReference,
			PredecessorKeyReference: entry.PredecessorKeyReference,
		}
	}
	return manifest, nil
}

func (m *EndpointManifest) resolve(owner, receiver string) (endpoint, error) {
	if m == nil {
		return endpoint{}, fmt.Errorf("%w: endpoint manifest is required", ErrConfig)
	}
	resolved, ok := m.entries[endpointKey{owner: owner, receiver: receiver}]
	if !ok {
		return endpoint{}, fmt.Errorf("%w: receiver is not registered", ErrNotFound)
	}
	return resolved, nil
}
