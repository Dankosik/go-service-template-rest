package oauth2clientcredentials

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewValidatesOnlyThePortableMinimum(t *testing.T) {
	valid := Config{
		TokenURL:     "https://auth.example.com/oauth/token",
		ClientID:     "client",
		ClientSecret: "secret",
		Scopes:       []string{"payments.read"},
	}
	client, err := New(valid)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Close(closeCtx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	for name, cfg := range map[string]Config{
		"missing token URL":     {ClientID: "client", ClientSecret: "secret"},
		"plaintext token URL":   {TokenURL: "http://auth.example.com/token", ClientID: "client", ClientSecret: "secret"},
		"missing client ID":     {TokenURL: valid.TokenURL, ClientSecret: "secret"},
		"missing client secret": {TokenURL: valid.TokenURL, ClientID: "client"},
		"two target selectors":  {TokenURL: valid.TokenURL, ClientID: "client", ClientSecret: "secret", Audience: "api", Resource: "api"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(cfg)
			if !errors.Is(err, ErrInvalidConfiguration) {
				t.Fatalf("New() error = %v, want ErrInvalidConfiguration", err)
			}
		})
	}
}
