// Package manifest owns the pure inbound-webhook endpoint manifest rules shared
// by configuration loading and the PostgreSQL runtime adapter.
//
// profile:inbound-webhooks-standard:start
package manifest

import (
	"encoding/json/v2"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxDocumentBytes = 1 << 20
	maxEntries       = 4096
	maxIDBytes       = 64
)

// Endpoint is one non-secret inbound-webhook trust identity.
type Endpoint struct {
	ID                      string
	ActiveKeyReference      string
	PredecessorKeyReference string
}

// Endpoints is an immutable non-secret endpoint snapshot.
type Endpoints struct {
	endpoints map[string]Endpoint
}

type document struct {
	Endpoints []entry `json:"endpoints"`
}

type entry struct {
	EndpointID              string `json:"endpoint_id"`
	ActiveKeyReference      string `json:"active_key_reference"`
	PredecessorKeyReference string `json:"predecessor_key_reference,omitempty"`
}

// ParseEndpoints parses the non-secret endpoint document.
func ParseEndpoints(raw string) (*Endpoints, error) {
	if raw == "" {
		return &Endpoints{endpoints: map[string]Endpoint{}}, nil
	}
	if len(raw) > maxDocumentBytes {
		return nil, errors.New("parse inbound webhook endpoints: document size is invalid")
	}
	var parsed document
	if err := json.UnmarshalRead(strings.NewReader(raw), &parsed, json.RejectUnknownMembers(true)); err != nil {
		return nil, errors.New("parse inbound webhook endpoints: invalid JSON")
	}
	if len(parsed.Endpoints) == 0 || len(parsed.Endpoints) > maxEntries {
		return nil, errors.New("parse inbound webhook endpoints: entries are required")
	}
	result := &Endpoints{endpoints: make(map[string]Endpoint, len(parsed.Endpoints))}
	for _, value := range parsed.Endpoints {
		if !ValidEndpointID(value.EndpointID) || !ValidKeyReference(value.ActiveKeyReference) {
			return nil, errors.New("parse inbound webhook endpoints: invalid identifier")
		}
		if value.PredecessorKeyReference != "" {
			if !ValidKeyReference(value.PredecessorKeyReference) {
				return nil, errors.New("parse inbound webhook endpoints: invalid predecessor key")
			}
			if value.PredecessorKeyReference == value.ActiveKeyReference {
				return nil, errors.New("parse inbound webhook endpoints: predecessor key must differ from active key")
			}
		}
		if _, exists := result.endpoints[value.EndpointID]; exists {
			return nil, errors.New("parse inbound webhook endpoints: duplicate endpoint")
		}
		result.endpoints[value.EndpointID] = Endpoint{
			ID:                      value.EndpointID,
			ActiveKeyReference:      value.ActiveKeyReference,
			PredecessorKeyReference: value.PredecessorKeyReference,
		}
	}
	return result, nil
}

// Lookup returns the named endpoint.
func (m *Endpoints) Lookup(endpointID string) (Endpoint, bool) {
	if m == nil {
		return Endpoint{}, false
	}
	endpoint, ok := m.endpoints[endpointID]
	return endpoint, ok
}

// IDs returns configured endpoint identifiers.
func (m *Endpoints) IDs() []string {
	if m == nil {
		return nil
	}
	ids := make([]string, 0, len(m.endpoints))
	for id := range m.endpoints {
		ids = append(ids, id)
	}
	return ids
}

// ValidEndpointID reports whether value is one bounded ASCII endpoint token.
func ValidEndpointID(value string) bool {
	if value == "" || len(value) > maxIDBytes || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r > unicode.MaxASCII || (r != '_' && r != '-' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9')) {
			return false
		}
	}
	return true
}

// ValidKeyReference reports whether value is one bounded non-space key token.
func ValidKeyReference(value string) bool {
	if value == "" || len(value) > maxIDBytes || !utf8.ValidString(value) {
		return false
	}
	return strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) < 0
}

// profile:inbound-webhooks-standard:end
