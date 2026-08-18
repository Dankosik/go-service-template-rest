package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateHTTPIdempotencyActive(t *testing.T) {
	valid := HTTPIdempotencyConfig{Retention: time.Hour}
	for _, tc := range []struct {
		name     string
		cfg      HTTPIdempotencyConfig
		postgres PostgresConfig
		want     string
	}{
		{name: "valid", cfg: valid, postgres: PostgresConfig{Enabled: true}},
		{name: "postgres disabled", cfg: valid, want: "postgres.enabled"},
		{name: "missing retention", postgres: PostgresConfig{Enabled: true}, want: "http_idempotency.retention"},
		{name: "negative retention", cfg: HTTPIdempotencyConfig{Retention: -time.Second}, postgres: PostgresConfig{Enabled: true}, want: "http_idempotency.retention"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateHTTPIdempotencyActive(tc.cfg, tc.postgres)
			if tc.want == "" && err != nil {
				t.Fatalf("ValidateHTTPIdempotencyActive() error = %v", err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("ValidateHTTPIdempotencyActive() error = %v, want leaf %q", err, tc.want)
			}
		})
	}
}
