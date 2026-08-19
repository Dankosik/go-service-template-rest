package postgreswebhook

import (
	"errors"
	"strings"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

type SigningKey struct {
	Reference string
	Bytes     []byte
}

func SignV1(deliveryID string, attemptedAt time.Time, body []byte, keys []SigningKey) (string, error) {
	if err := validateToken("delivery_id", deliveryID); err != nil || strings.Contains(deliveryID, ".") {
		return "", ErrConfig
	}
	if len(keys) < 1 || len(keys) > 2 {
		return "", errors.New("sign webhook: one active and optional predecessor key are required")
	}
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key.Bytes) < 32 || len(key.Bytes) > 64 {
			return "", errors.New("sign webhook: key must contain 32..64 bytes")
		}
		webhook, err := standardwebhooks.NewWebhookRaw(key.Bytes)
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
