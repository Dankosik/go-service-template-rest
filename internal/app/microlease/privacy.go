package microlease

import "strings"

var unsafeMetadataTerms = []string{
	"raw_prompt",
	"prompt",
	"completion",
	"sse",
	"bearer",
	"token",
	"api_key",
	"apikey",
	"dsn",
	"password",
	"secret",
	"raw_provider_payload",
	"raw_event_payload",
	"dynamic_proof_url",
	"sensitive_request_body",
	"request_body",
}

func ValidateSafeMetadata(metadata map[string]string) error {
	for key, value := range metadata {
		if unsafeSupportText(key) || unsafeSupportText(value) {
			return ErrUnsafeMetadata
		}
	}
	return nil
}

func unsafeSupportText(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, term := range unsafeMetadataTerms {
		if strings.Contains(normalized, term) {
			return true
		}
	}
	return false
}
