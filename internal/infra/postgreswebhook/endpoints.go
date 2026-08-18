package postgreswebhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	MaxEndpointManifestBytes   = 1 << 20
	MaxEndpointManifestEntries = 4096
)

type endpointKey struct {
	owner    string
	receiver string
}

type Endpoint struct {
	OwnerScope              string
	ReceiverID              string
	Generation              int64
	URL                     string
	ActiveKeyReference      string
	PredecessorKeyReference string
}

type EndpointManifest struct {
	entries map[endpointKey]Endpoint
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
	if raw == "" || len(raw) > MaxEndpointManifestBytes {
		return nil, errors.New("parse webhook endpoints: document size is invalid")
	}
	if err := rejectDuplicateJSONFields(raw); err != nil {
		return nil, errors.New("parse webhook endpoints: duplicate field")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document endpointDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, errors.New("parse webhook endpoints: invalid JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("parse webhook endpoints: trailing data")
	}
	if len(document.Endpoints) == 0 || len(document.Endpoints) > MaxEndpointManifestEntries {
		return nil, errors.New("parse webhook endpoints: entries are required")
	}
	manifest := &EndpointManifest{entries: make(map[endpointKey]Endpoint, len(document.Endpoints))}
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
		manifest.entries[key] = Endpoint{
			OwnerScope: entry.OwnerScope, ReceiverID: entry.ReceiverID, Generation: entry.Generation,
			URL: parsed.String(), ActiveKeyReference: entry.ActiveKeyReference,
			PredecessorKeyReference: entry.PredecessorKeyReference,
		}
	}
	return manifest, nil
}

func (m *EndpointManifest) Resolve(owner, receiver string) (Endpoint, error) {
	if m == nil {
		return Endpoint{}, fmt.Errorf("%w: endpoint manifest is required", ErrConfig)
	}
	endpoint, ok := m.entries[endpointKey{owner: owner, receiver: receiver}]
	if !ok {
		return Endpoint{}, fmt.Errorf("%w: receiver is not registered", ErrNotFound)
	}
	return endpoint, nil
}
