package redpanda

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	eventsv1 "github.com/Dankosik/billing-service/internal/api/events/v1"
)

const eventFingerprintPrefix = "sha256:"

func FingerprintTerminalSubmitted(event eventsv1.MicroleaseChildTerminalSubmitted) (string, error) {
	event.Envelope.EventFingerprint = ""
	return fingerprintJSON(event)
}

func FingerprintCheckpointReported(event eventsv1.MicroleaseCheckpointReported) (string, error) {
	event.Envelope.EventFingerprint = ""
	return fingerprintJSON(event)
}

func FingerprintCloseReported(event eventsv1.MicroleaseCloseReported) (string, error) {
	event.Checkpoint.Envelope.EventFingerprint = ""
	return fingerprintJSON(event)
}

func FingerprintOutboxPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return eventFingerprintPrefix + hex.EncodeToString(sum[:])
}

func fingerprintJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("%w: marshal event fingerprint: %w", ErrInvalidEvent, err)
	}
	sum := sha256.Sum256(data)
	return eventFingerprintPrefix + hex.EncodeToString(sum[:]), nil
}

func fingerprintMatches(got, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got == "" || want == "" {
		return false
	}
	return got == want
}
