package natsjs

import (
	"testing"
)

func TestNewOutboxAppenderRejectsInvalidRoutes(t *testing.T) {
	t.Parallel()

	for name, routes := range map[string][]Route{
		"empty":    nil,
		"wildcard": {{Type: "order.updated", Version: 1, Subject: "events.*"}},
		"version":  {{Type: "order.updated", Subject: "events.orders"}},
		"duplicate": {
			{Type: "order.updated", Version: 1, Subject: "events.orders"},
			{Type: "order.updated", Version: 1, Subject: "events.orders"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewOutboxAppender(testMaxPayloadBytes, routes...); err == nil {
				t.Fatal("NewOutboxAppender() error = nil")
			}
		})
	}
}
