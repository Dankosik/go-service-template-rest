// profile:inbound-webhooks-standard:start
package inboundwebhook

import (
	"bytes"
	"testing"
)

func TestInboundWebhookDeliveryCloneCopiesBody(t *testing.T) {
	t.Parallel()

	original := []byte(`{"hello":"world"}`)
	delivery := Delivery{Body: original}.Clone()
	delivery.Body[0] = 'X'
	if bytes.Equal(original, delivery.Body) {
		t.Fatal("clone shared the body buffer")
	}
}

// profile:inbound-webhooks-standard:end
