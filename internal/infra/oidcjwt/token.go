package oidcjwt

import (
	"bytes"
	"encoding/json"
	"errors"
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

// numericDate is a JWT NumericDate held to integer seconds, with presence
// recorded separately so validTimes can tell an absent claim from a zero one.
//
// A fractional or exponent-notation value is refused rather than truncated: the
// two spellings would compare differently against ClockSkew while naming the same
// instant, and a claim this boundary acts on may have only one reading.
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

// audienceClaim is the aud claim in either RFC 7519 spelling, scalar or array.
//
// It owns half of the audience decision and only half: whether the claim is well
// formed at all — a legal spelling, no empty entry, no repeat — is settled here,
// while whether this service is named in it is a term in
// parseAccessTokenClaims. The split is what keeps a malformed claim from
// reaching the membership test, where an empty or duplicated entry would decide
// the answer. An added claim that needs the same treatment belongs in a type
// like this one; one that only needs a value belongs in the conjunction there.
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

// bearerToken extracts the compact token from the credential header values.
//
// Both transports share it: HTTP passes the Authorization header values and gRPC
// passes the authorization metadata values, which carry the same RFC 6750
// syntax. Sharing one owner is what keeps a single header and a single metadata
// entry meaning the same thing, so a client cannot reach a laxer reading by
// switching protocol.
//
// One value, no surrounding space, no comma, no whitespace inside the token: a
// credential this boundary acts on may have exactly one reading. RFC 9110 lets a
// field repeat and lets one line carry a comma-separated list, so a request
// offering two credentials has to be refused rather than resolved — choosing
// among them is how one intermediary's reading comes to differ from this
// service's. Refusal is KindMalformed, which the HTTP adapter reports as 400
// rather than 401: the framing was wrong, so re-presenting the same pair against
// a challenge would fail identically.
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
	if !found ||
		!strings.EqualFold(scheme, "Bearer") ||
		token == "" ||
		strings.ContainsAny(token, " \t\r\n") {
		return "", failure(KindMalformed)
	}
	if len(token) > MaxTokenBytes {
		return "", failure(KindOversize)
	}
	return token, nil
}

func parseToken(compact string, policy Policy, now time.Time) (parsedToken, error) {
	// bearerToken already bounds everything the adapters pass. This repeats the
	// cap for [Verifier.Verify], which is exported and reachable with a token that
	// never came through a credential header.
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
	// The signature segment is decoded only to be held to one encoding; the bytes
	// are jose's to check. Dropping this would let the same token be rewritten
	// into several compact forms that all verify, which is what
	// decodeCanonicalSegment exists to prevent and what any downstream identity or
	// replay decision keyed on the compact text depends on.
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
	if err := strictUnmarshal(headerBytes, &header); err != nil {
		return protectedHeader{}, failure(KindInvalid)
	}
	// The four raw fields must be absent rather than valid: jku, x5u, and jwk all
	// name a key this service did not choose, and b64 changes what the signature
	// covers. crit is refused because honouring an extension we do not implement
	// is not something a verifier may guess at.
	if header.Algorithm != AllowedAlgorithm ||
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
	if err := strictUnmarshal(payload, &claims); err != nil {
		return accessTokenClaims{}, failure(KindInvalid)
	}
	// One conjunction on purpose: every term is a condition this service requires
	// of an access token, and each is equally fatal. An additional claim
	// requirement is a term added here.
	if claims.Issuer != policy.issuer ||
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
	return !claims.NotBefore.present || claims.NotBefore.value <= nowSeconds+skewSeconds
}
