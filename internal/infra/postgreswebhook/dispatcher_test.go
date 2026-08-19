package postgreswebhook

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestDispatcherPreparesStableReceiverJobs(t *testing.T) {
	endpoints, err := ParseEndpointManifest(`{"endpoints":[
		{"owner_scope":"orders","receiver_id":"beta","generation":2,"url":"https://beta.example/hooks","active_key_reference":"beta-v2"},
		{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"alpha-v1"}
	]}`)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewDispatcher(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	event := Event{
		OwnerScope: "orders", ID: "evt-1", Type: "order.created",
		OccurredAt: time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC),
		Data:       json.RawMessage(`{"order_id":"ord-1"}`),
	}
	prepared, err := dispatcher.Prepare(event, []ReceiverID{"beta", "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	ids := prepared.DeliveryIDs()
	if len(ids) != 2 || ids[0] == ids[1] || ids[0][:4] != "whd_" || ids[1][:4] != "whd_" {
		t.Fatalf("delivery ids = %v", ids)
	}

	first := prepared.deliveries[0].args
	if first.ReceiverID != "alpha" || first.ReceiverGeneration != 1 || first.URL != "https://alpha.example:443/hooks" {
		t.Fatalf("first delivery = %+v", first)
	}
	var body struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(first.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Type != event.Type || body.Timestamp != "2026-08-18T10:00:00Z" || string(body.Data) != `{"order_id":"ord-1"}` {
		t.Fatalf("body = %s", first.Body)
	}

	repeated, err := dispatcher.Prepare(event, []ReceiverID{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if got := repeated.DeliveryIDs(); ids[0] != got[0] || ids[1] != got[1] {
		t.Fatalf("repeated ids = %v, want %v", got, ids)
	}
}

func TestDispatcherRejectsInvalidBusinessInput(t *testing.T) {
	endpoints, err := ParseEndpointManifest(`{"endpoints":[{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"alpha-v1"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewDispatcher(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	valid := Event{OwnerScope: "orders", ID: "evt-1", Type: "order.created", OccurredAt: time.Now().UTC(), Data: json.RawMessage(`{"id":1}`)}
	oversized := append([]byte{'"'}, bytes.Repeat([]byte{'a'}, MaxEventDataBytes)...)
	oversized = append(oversized, '"')
	for _, test := range []struct {
		name      string
		event     Event
		receivers []ReceiverID
	}{
		{name: "duplicate receiver", event: valid, receivers: []ReceiverID{"alpha", "alpha"}},
		{name: "missing receiver", event: valid, receivers: []ReceiverID{"missing"}},
		{name: "invalid JSON", event: Event{OwnerScope: "orders", ID: "evt-1", Type: "order.created", OccurredAt: time.Now().UTC(), Data: json.RawMessage(`{`)}, receivers: []ReceiverID{"alpha"}},
		{name: "oversized JSON", event: Event{OwnerScope: "orders", ID: "evt-1", Type: "order.created", OccurredAt: time.Now().UTC(), Data: oversized}, receivers: []ReceiverID{"alpha"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := dispatcher.Prepare(test.event, test.receivers); err == nil {
				t.Fatal("Prepare() succeeded")
			}
		})
	}
}
