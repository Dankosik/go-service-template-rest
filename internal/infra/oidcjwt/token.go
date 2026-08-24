package oidcjwt

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
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

func principalFromClaims(claims *accessTokenClaims, strict bool) (reqctx.Principal, error) {
	if claims == nil {
		return reqctx.Principal{}, failure(bearerauthn.KindInvalid)
	}
	clientID, err := oneClientID(claims.ClientID, claims.AuthorizedParty, claims.ApplicationID, claims.OktaClientID)
	if err != nil {
		return reqctx.Principal{}, err
	}
	subject := claims.Subject
	if strings.TrimSpace(subject) != subject || (subject == "" && clientID == "") {
		return reqctx.Principal{}, failure(bearerauthn.KindInvalid)
	}
	if strict && (subject == "" || strings.TrimSpace(claims.ClientID) == "" || strings.TrimSpace(claims.ID) == "" || claims.IssuedAt == nil) {
		return reqctx.Principal{}, failure(bearerauthn.KindInvalid)
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
			return "", failure(bearerauthn.KindInvalid)
		}
		if selected != "" && value != selected {
			return "", failure(bearerauthn.KindInvalid)
		}
		selected = value
	}
	return selected, nil
}

func failure(kind bearerauthn.Kind) error {
	return fmt.Errorf("verify access token: %w", bearerauthn.NewError(kind))
}
