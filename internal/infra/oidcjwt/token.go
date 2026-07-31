package oidcjwt

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	jose "github.com/go-jose/go-jose/v4"
)

type protectedHeader struct {
	Algorithm string          `json:"alg"`
	Type      string          `json:"typ"`
	KeyID     string          `json:"kid"`
	Critical  json.RawMessage `json:"crit"`
	B64       json.RawMessage `json:"b64"`
	JKU       json.RawMessage `json:"jku"`
	X5U       json.RawMessage `json:"x5u"`
	JWK       json.RawMessage `json:"jwk"`
}

type numericDate struct {
	value   int64
	present bool
}

func (n *numericDate) UnmarshalJSON(data []byte) error {
	if bytes.ContainsAny(data, ".eE") || bytes.Equal(data, []byte("null")) {
		return errors.New("NumericDate must be an integer")
	}
	var value int64
	if err := strictUnmarshal(data, &value); err != nil {
		return errors.New("NumericDate must be an integer")
	}
	n.value = value
	n.present = true
	return nil
}

type audienceClaim struct {
	values  []string
	present bool
}

func (a *audienceClaim) UnmarshalJSON(data []byte) error {
	var one string
	if err := strictUnmarshal(data, &one); err == nil {
		if one == "" {
			return errors.New("audience is empty")
		}
		a.values = []string{one}
		a.present = true
		return nil
	}

	var many []string
	if err := strictUnmarshal(data, &many); err != nil || len(many) == 0 {
		return errors.New("audience is invalid")
	}
	seen := make(map[string]struct{}, len(many))
	for _, value := range many {
		if value == "" {
			return errors.New("audience is empty")
		}
		if _, exists := seen[value]; exists {
			return errors.New("audience is duplicated")
		}
		seen[value] = struct{}{}
	}
	a.values = many
	a.present = true
	return nil
}

type accessTokenClaims struct {
	Issuer    string        `json:"iss"`
	Subject   string        `json:"sub"`
	Audience  audienceClaim `json:"aud"`
	ClientID  string        `json:"client_id"`
	JWTID     string        `json:"jti"`
	Expires   numericDate   `json:"exp"`
	IssuedAt  numericDate   `json:"iat"`
	NotBefore numericDate   `json:"nbf"`
}

type parsedToken struct {
	signed    *jose.JSONWebSignature
	keyID     string
	payload   []byte
	principal reqctx.Principal
}

func parseToken(compact string, policy Policy, now time.Time) (parsedToken, error) {
	if len(compact) > MaxTokenBytes {
		return parsedToken{}, failure(KindOversize)
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 || parts[2] == "" {
		return parsedToken{}, failure(KindInvalid)
	}
	header, err := parseProtectedHeader(parts[0])
	if err != nil {
		return parsedToken{}, err
	}
	payload, err := decodeCanonicalSegment(parts[1])
	if err != nil {
		return parsedToken{}, failure(KindInvalid)
	}
	if _, err := decodeCanonicalSegment(parts[2]); err != nil {
		return parsedToken{}, failure(KindInvalid)
	}
	claims, err := parseAccessTokenClaims(payload, policy, now)
	if err != nil {
		return parsedToken{}, err
	}

	signed, err := jose.ParseSignedCompact(compact, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil || len(signed.Signatures) != 1 {
		return parsedToken{}, failure(KindInvalid)
	}
	return parsedToken{
		signed:    signed,
		keyID:     header.KeyID,
		payload:   payload,
		principal: reqctx.Principal{Subject: claims.Subject},
	}, nil
}

func parseProtectedHeader(segment string) (protectedHeader, error) {
	headerBytes, err := decodeCanonicalSegment(segment)
	if err != nil {
		return protectedHeader{}, failure(KindInvalid)
	}
	var header protectedHeader
	if err := strictUnmarshal(headerBytes, &header); err != nil ||
		header.Algorithm != AllowedAlgorithm ||
		strings.TrimSpace(header.KeyID) == "" ||
		!validAccessTokenType(header.Type) ||
		header.Critical != nil ||
		header.B64 != nil ||
		header.JKU != nil ||
		header.X5U != nil ||
		header.JWK != nil {
		return protectedHeader{}, failure(KindInvalid)
	}
	return header, nil
}

func parseAccessTokenClaims(payload []byte, policy Policy, now time.Time) (accessTokenClaims, error) {
	var claims accessTokenClaims
	if err := strictUnmarshal(payload, &claims); err != nil ||
		claims.Issuer != policy.issuer ||
		strings.TrimSpace(claims.Subject) == "" ||
		strings.TrimSpace(claims.ClientID) == "" ||
		strings.TrimSpace(claims.JWTID) == "" ||
		!claims.Audience.present ||
		!slices.Contains(claims.Audience.values, policy.audience) ||
		!validTimes(claims, now) {
		return accessTokenClaims{}, failure(KindInvalid)
	}
	return claims, nil
}

func validAccessTokenType(value string) bool {
	return strings.EqualFold(value, "at+jwt") || strings.EqualFold(value, "application/at+jwt")
}

func validTimes(claims accessTokenClaims, now time.Time) bool {
	if !claims.Expires.present || !claims.IssuedAt.present {
		return false
	}
	nowSeconds := now.Unix()
	skewSeconds := int64(ClockSkew / time.Second)
	if claims.Expires.value <= nowSeconds-skewSeconds {
		return false
	}
	if claims.IssuedAt.value > nowSeconds+skewSeconds {
		return false
	}
	if claims.NotBefore.present && claims.NotBefore.value > nowSeconds+skewSeconds {
		return false
	}
	return claims.Expires.value != math.MinInt64
}
