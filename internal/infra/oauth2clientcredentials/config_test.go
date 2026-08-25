package oauth2clientcredentials

import (
	"errors"
	"testing"
)

func TestNewValidatesOnlyThePortableMinimum(t *testing.T) {
	valid := Config{ //nolint:gosec // Obvious local test fixture, not a live credential.
		TokenURL:     "https://auth.example.com/oauth/token",
		ClientID:     "client",
		ClientSecret: "secret",
		Scopes:       []string{"payments.read"},
	}
	client, err := New(valid)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	client.Close()

	for name, cfg := range map[string]Config{
		"missing token URL":     {ClientID: "client", ClientSecret: "secret"},
		"plaintext token URL":   {TokenURL: "http://auth.example.com/token", ClientID: "client", ClientSecret: "secret"}, //nolint:gosec // Rejected test fixture.
		"missing client ID":     {TokenURL: valid.TokenURL, ClientSecret: "secret"},
		"missing client secret": {TokenURL: valid.TokenURL, ClientID: "client"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(cfg)
			if !errors.Is(err, ErrInvalidConfiguration) {
				t.Fatalf("New() error = %v, want ErrInvalidConfiguration", err)
			}
		})
	}
}
