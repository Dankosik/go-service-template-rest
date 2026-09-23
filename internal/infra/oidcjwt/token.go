package oidcjwt

import (
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/golang-jwt/jwt/v5"
)

type accessTokenClaims struct {
	jwt.RegisteredClaims

	ClientID        string `json:"client_id"`
	AuthorizedParty string `json:"azp"`
	ApplicationID   string `json:"appid"`
	OktaClientID    string `json:"cid"`
}

func validAccessTokenType(value any) bool {
	typ, ok := value.(string)
	return ok && (strings.EqualFold(typ, "at+jwt") || strings.EqualFold(typ, "application/at+jwt"))
}

// principalFromClaims reports false when the claims do not name one caller.
func principalFromClaims(claims *accessTokenClaims, strict bool) (reqctx.Principal, bool) {
	if claims == nil {
		return reqctx.Principal{}, false
	}
	clientID, ok := oneClientID(claims.ClientID, claims.AuthorizedParty, claims.ApplicationID, claims.OktaClientID)
	if !ok {
		return reqctx.Principal{}, false
	}
	subject := claims.Subject
	if strings.TrimSpace(subject) != subject || (subject == "" && clientID == "") {
		return reqctx.Principal{}, false
	}
	if strict && (subject == "" || strings.TrimSpace(claims.ClientID) == "" || strings.TrimSpace(claims.ID) == "" || claims.IssuedAt == nil) {
		return reqctx.Principal{}, false
	}
	return reqctx.Principal{Issuer: claims.Issuer, Subject: subject, ClientID: clientID}, true
}

// oneClientID reports false when the client ID claims disagree or carry
// surrounding whitespace.
func oneClientID(values ...string) (string, bool) {
	selected := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if strings.TrimSpace(value) != value {
			return "", false
		}
		if selected != "" && value != selected {
			return "", false
		}
		selected = value
	}
	return selected, true
}
