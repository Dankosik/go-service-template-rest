package oauthintrospection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

func admitResponse(body []byte, policy Policy, now time.Time) (bearerauthn.Result, error) {
	members, err := decodeObjectMembers(body)
	if err != nil {
		return bearerauthn.Result{}, err
	}
	active, err := booleanMember(members, "active")
	if err != nil {
		return bearerauthn.Result{}, err
	}
	if !active {
		return bearerauthn.Result{}, failure(bearerauthn.KindInvalid)
	}

	issuer, err := requiredStringMember(members, "iss")
	if err != nil {
		return bearerauthn.Result{}, err
	}
	audiences, err := audienceMember(members)
	if err != nil {
		return bearerauthn.Result{}, err
	}
	expiresAt, err := requiredNumericDate(members, "exp")
	if err != nil {
		return bearerauthn.Result{}, err
	}
	notBefore, hasNotBefore, err := optionalNumericDate(members, "nbf")
	if err != nil {
		return bearerauthn.Result{}, err
	}
	subject, err := optionalIdentityMember(members, "sub")
	if err != nil {
		return bearerauthn.Result{}, err
	}
	clientID, err := optionalIdentityMember(members, "client_id")
	if err != nil {
		return bearerauthn.Result{}, err
	}
	if subject == "" && clientID == "" {
		return bearerauthn.Result{}, failure(bearerauthn.KindUnavailable)
	}

	if issuer != policy.issuer || !slices.Contains(audiences, policy.audience) {
		return bearerauthn.Result{}, failure(bearerauthn.KindInvalid)
	}
	if expiresAt.Add(bearerauthn.ClockSkew).Before(now) {
		return bearerauthn.Result{}, failure(bearerauthn.KindInvalid)
	}
	if hasNotBefore && now.Add(bearerauthn.ClockSkew).Before(notBefore) {
		return bearerauthn.Result{}, failure(bearerauthn.KindInvalid)
	}

	return bearerauthn.Result{
		Principal: reqctx.Principal{
			Issuer:   policy.issuer,
			Subject:  subject,
			ClientID: clientID,
		},
		ExpiresAt: expiresAt,
	}, nil
}

func decodeObjectMembers(body []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return nil, failure(bearerauthn.KindUnavailable)
	}

	members := make(map[string]json.RawMessage)
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, failure(bearerauthn.KindUnavailable)
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, failure(bearerauthn.KindUnavailable)
		}
		if _, exists := members[key]; exists {
			return nil, failure(bearerauthn.KindUnavailable)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, failure(bearerauthn.KindUnavailable)
		}
		members[key] = value
	}
	end, err := decoder.Token()
	if err != nil {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	if endDelim, ok := end.(json.Delim); !ok || endDelim != '}' {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	return members, nil
}

func booleanMember(members map[string]json.RawMessage, name string) (bool, error) {
	raw, ok := members[name]
	if !ok {
		return false, failure(bearerauthn.KindUnavailable)
	}
	var value *bool
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return false, failure(bearerauthn.KindUnavailable)
	}
	return *value, nil
}

func requiredStringMember(members map[string]json.RawMessage, name string) (string, error) {
	raw, ok := members[name]
	if !ok {
		return "", failure(bearerauthn.KindUnavailable)
	}
	value, err := exactJSONString(raw)
	if err != nil || value == "" {
		return "", failure(bearerauthn.KindUnavailable)
	}
	return value, nil
}

func optionalIdentityMember(members map[string]json.RawMessage, name string) (string, error) {
	raw, ok := members[name]
	if !ok {
		return "", nil
	}
	value, err := exactJSONString(raw)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", nil
	}
	if strings.TrimSpace(value) != value {
		return "", failure(bearerauthn.KindUnavailable)
	}
	return value, nil
}

func exactJSONString(raw json.RawMessage) (string, error) {
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return "", failure(bearerauthn.KindUnavailable)
	}
	return *value, nil
}

func audienceMember(members map[string]json.RawMessage) ([]string, error) {
	raw, ok := members["aud"]
	if !ok {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	var single *string
	if err := json.Unmarshal(raw, &single); err == nil && single != nil {
		if *single == "" {
			return nil, failure(bearerauthn.KindUnavailable)
		}
		return []string{*single}, nil
	}
	var audiences []string
	if err := json.Unmarshal(raw, &audiences); err != nil || len(audiences) == 0 {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	if slices.Contains(audiences, "") {
		return nil, failure(bearerauthn.KindUnavailable)
	}
	return audiences, nil
}

func requiredNumericDate(members map[string]json.RawMessage, name string) (time.Time, error) {
	raw, ok := members[name]
	if !ok {
		return time.Time{}, failure(bearerauthn.KindUnavailable)
	}
	seconds, err := exactIntegralNumber(raw)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(seconds, 0).UTC(), nil
}

func optionalNumericDate(members map[string]json.RawMessage, name string) (time.Time, bool, error) {
	raw, ok := members[name]
	if !ok {
		return time.Time{}, false, nil
	}
	seconds, err := exactIntegralNumber(raw)
	if err != nil {
		return time.Time{}, false, err
	}
	return time.Unix(seconds, 0).UTC(), true, nil
}

func exactIntegralNumber(raw json.RawMessage) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, failure(bearerauthn.KindUnavailable)
	}
	return value, nil
}

func failure(kind bearerauthn.Kind) error {
	return fmt.Errorf("verify access token: %w", bearerauthn.NewError(kind))
}
