package bearerauthn

import "strings"

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
