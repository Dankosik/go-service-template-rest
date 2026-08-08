package oidcjwt

import (
	"encoding/json"
	"errors"
	"math/big"
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

// numericDate is a JWT NumericDate held at time.Time's nanosecond resolution,
// with presence recorded separately so validTimes can tell an absent claim from
// a zero one. Values finer than a nanosecond are rounded down, so accepted trust
// never lasts longer than the instant the token names.
type numericDate struct {
	nanoseconds int64
	present     bool
}

func (n *numericDate) UnmarshalJSON(data []byte) error {
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return errors.New("NumericDate must be a number")
	}
	value, ok := new(big.Rat).SetString(number.String())
	if !ok {
		return errors.New("NumericDate must be a number")
	}
	scaled := new(big.Rat).Mul(value, big.NewRat(int64(time.Second), 1))
	whole, remainder := new(big.Int), new(big.Int)
	whole.QuoRem(scaled.Num(), scaled.Denom(), remainder)
	if scaled.Sign() < 0 && remainder.Sign() != 0 {
		whole.Sub(whole, big.NewInt(1))
	}
	if !whole.IsInt64() {
		return errors.New("NumericDate is outside supported time range")
	}
	n.nanoseconds = whole.Int64()
	n.present = true
	return nil
}

func (n *numericDate) time() time.Time {
	return time.Unix(0, n.nanoseconds)
}

// audienceClaim is the aud claim in either RFC 7519 spelling, scalar or array.
//
// It owns half of the audience decision and only half: whether the claim is well
// formed — a legal spelling, no empty entry, no repeat — is settled here, while
// whether this service is named in it is a term in parseAccessTokenClaims. The
// split keeps a malformed claim from reaching the membership test, where an empty
// or duplicated entry would decide the answer. An added claim needing the same
// treatment belongs in a type like this one.
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
	expiresAt time.Time
}

// bearerToken extracts the compact token from the credential header values.
//
// Both transports share it: HTTP passes the Authorization header values and gRPC
// passes the authorization metadata values, which carry the same RFC 6750
// syntax. Sharing one owner is what keeps a single header and a single metadata
// entry meaning the same thing, so a client cannot reach a laxer reading by
// switching protocol.
//
// One value, no surrounding space, no comma, and only RFC 6750's one-or-more
// ASCII spaces between the scheme and token: a credential this boundary acts on
// may have exactly one reading. RFC 9110 lets a field repeat and lets one line
// carry a comma-separated list, so a request
// offering two credentials is refused rather than resolved — choosing among them
// is how one intermediary's reading comes to differ from this service's. Refusal
// is KindMalformed, reported as 400 rather than 401: the framing was wrong, so
// re-presenting the same pair against a challenge would fail identically.
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
	// cap for verify as a local invariant, even though adapters already checked
	// the credential before reaching it.
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
	// are jose's to check. decodeCanonicalSegment owns what dropping this would
	// cost.
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
		signed:  signed,
		keyID:   header.KeyID,
		payload: payload,
		principal: reqctx.Principal{
			Issuer:   claims.Issuer,
			Subject:  claims.Subject,
			ClientID: claims.ClientID,
		},
		expiresAt: claims.Expires.time(),
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
	issuedAt := claims.IssuedAt.time()
	expires := claims.Expires.time()
	if !expires.After(issuedAt) || expires.Sub(issuedAt) > MaxTokenLifetime {
		return false
	}
	if !expires.After(now.Add(-ClockSkew)) {
		return false
	}
	if issuedAt.After(now.Add(ClockSkew)) {
		return false
	}
	if !claims.NotBefore.present {
		return true
	}
	notBefore := claims.NotBefore.time()
	return notBefore.Before(expires) && !notBefore.After(now.Add(ClockSkew))
}
