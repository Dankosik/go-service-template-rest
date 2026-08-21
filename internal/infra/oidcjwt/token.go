package oidcjwt

import (
	"strings"
	"time"

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

type parsedToken struct {
	principal reqctx.Principal
	expiresAt time.Time
}

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

func validAccessTokenType(value any) bool {
	typ, ok := value.(string)
	return ok && (strings.EqualFold(typ, "at+jwt") || strings.EqualFold(typ, "application/at+jwt"))
}

func principalFromClaims(claims *accessTokenClaims, strict bool) (reqctx.Principal, error) {
	if claims == nil {
		return reqctx.Principal{}, failure(KindInvalid)
	}
	clientID, err := oneClientID(claims.ClientID, claims.AuthorizedParty, claims.ApplicationID, claims.OktaClientID)
	if err != nil {
		return reqctx.Principal{}, err
	}
	subject := claims.Subject
	if strings.TrimSpace(subject) != subject || (subject == "" && clientID == "") {
		return reqctx.Principal{}, failure(KindInvalid)
	}
	if strict && (strings.TrimSpace(claims.ClientID) == "" || strings.TrimSpace(claims.ID) == "" || claims.IssuedAt == nil) {
		return reqctx.Principal{}, failure(KindInvalid)
	}
	return reqctx.Principal{Issuer: claims.Issuer, Subject: subject, ClientID: clientID}, nil
}

func oneClientID(values ...string) (string, error) {
	selected := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if strings.TrimSpace(value) != value {
			return "", failure(KindInvalid)
		}
		if selected != "" && value != selected {
			return "", failure(KindInvalid)
		}
		selected = value
	}
	return selected, nil
}
