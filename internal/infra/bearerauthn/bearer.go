package bearerauthn

import "strings"

// bearerToken extracts the credential from exactly one Authorization value of
// the form "Bearer <token>" (RFC 6750 section 2.1). The scheme matches
// case-insensitively and is separated from the token by spaces; the value has
// no surrounding whitespace and no comma, and the token has no whitespace.
func bearerToken(values []string) (string, error) {
	if len(values) == 0 {
		return "", failure(KindMissing)
	}
	if len(values) != 1 {
		return "", failure(KindMalformed)
	}
	value := values[0]
	if strings.TrimSpace(value) != value || strings.Contains(value, ",") {
		return "", failure(KindMalformed)
	}
	scheme, token, found := strings.Cut(value, " ")
	token = strings.TrimLeft(token, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", failure(KindMalformed)
	}
	if len(token) > MaxTokenBytes {
		return "", failure(KindOversize)
	}
	return token, nil
}
