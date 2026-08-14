package postgreswebhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

type SigningKey struct {
	Reference string
	Bytes     []byte
}

func SignV1(deliveryID string, attemptedAt time.Time, body []byte, keys []SigningKey) (string, [32]byte, error) {
	if err := validateToken("delivery_id", deliveryID); err != nil || strings.Contains(deliveryID, ".") {
		return "", [32]byte{}, ErrConfig
	}
	if len(keys) < 1 || len(keys) > 2 {
		return "", [32]byte{}, errors.New("sign webhook: one active and optional predecessor key are required")
	}
	timestamp := strconv.FormatInt(attemptedAt.Unix(), 10)
	input := make([]byte, 0, len(deliveryID)+len(timestamp)+len(body)+2)
	input = append(input, deliveryID...)
	input = append(input, '.')
	input = append(input, timestamp...)
	input = append(input, '.')
	input = append(input, body...)
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(key.Bytes) < 32 || len(key.Bytes) > 64 {
			return "", [32]byte{}, errors.New("sign webhook: key must contain 32..64 bytes")
		}
		mac := hmac.New(sha256.New, key.Bytes)
		_, _ = mac.Write(input)
		entries = append(entries, "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	}
	header := strings.Join(entries, " ")
	return header, sha256.Sum256([]byte(header)), nil
}

func VerifyV1(deliveryID, timestamp string, body []byte, header string, keys [][]byte) bool {
	input := []byte(deliveryID + "." + timestamp + ".")
	input = append(input, body...)
	for entry := range strings.FieldsSeq(header) {
		version, raw, ok := strings.Cut(entry, ",")
		if !ok || version != "v1" {
			continue
		}
		signature, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(signature) != sha256.Size {
			continue
		}
		for _, key := range keys {
			mac := hmac.New(sha256.New, key)
			_, _ = mac.Write(input)
			if hmac.Equal(signature, mac.Sum(nil)) {
				return true
			}
		}
	}
	return false
}
