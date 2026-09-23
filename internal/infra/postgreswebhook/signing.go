package postgreswebhook

import (
	"errors"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/webhooksecret"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

// signV1 signs with the active key and, when present, the predecessor key, in
// that order.
func signV1(deliveryID string, attemptedAt time.Time, body []byte, active, predecessor []byte) (string, error) {
	if err := validateToken("delivery_id", deliveryID); err != nil || strings.Contains(deliveryID, ".") {
		return "", ErrConfig
	}
	keys := [][]byte{active}
	if predecessor != nil {
		keys = append(keys, predecessor)
	}
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		if !webhooksecret.ValidKey(key) {
			return "", errors.New("sign webhook: key must contain 32..64 bytes")
		}
		webhook, err := standardwebhooks.NewWebhookRaw(key)
		if err != nil {
			return "", errors.New("sign webhook: initialize signer")
		}
		signature, err := webhook.Sign(deliveryID, attemptedAt, body)
		if err != nil {
			return "", errors.New("sign webhook: create signature")
		}
		entries = append(entries, signature)
	}
	return strings.Join(entries, " "), nil
}
