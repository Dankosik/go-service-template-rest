package httpclient

import (
	"strings"
	"testing"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "missing dependency", mutate: func(cfg *Config) { cfg.DependencyName = "" }},
		{name: "invalid URL", mutate: func(cfg *Config) { cfg.BaseURL = "://invalid" }},
		{name: "relative URL", mutate: func(cfg *Config) { cfg.BaseURL = "/provider" }},
		{name: "user info", mutate: func(cfg *Config) { cfg.BaseURL = "https://user@example.com" }},
		{name: "query", mutate: func(cfg *Config) { cfg.BaseURL = "https://example.com?token=secret" }},
		{name: "fragment", mutate: func(cfg *Config) { cfg.BaseURL = "https://example.com#fragment" }},
		{name: "missing request timeout", mutate: func(cfg *Config) { cfg.RequestTimeout = 0 }},
		{name: "missing header timeout", mutate: func(cfg *Config) { cfg.ResponseHeaderTimeout = 0 }},
		{name: "missing header limit", mutate: func(cfg *Config) { cfg.MaxResponseHeaderBytes = 0 }},
		{name: "missing body limit", mutate: func(cfg *Config) { cfg.MaxResponseBodyBytes = 0 }},
		{name: "missing conn limit", mutate: func(cfg *Config) { cfg.MaxConnsPerHost = 0 }},
		{name: "negative idle conn limit", mutate: func(cfg *Config) { cfg.MaxIdleConnsPerHost = -1 }},
		{name: "idle conn limit above conn limit", mutate: func(cfg *Config) {
			cfg.MaxConnsPerHost = 4
			cfg.MaxIdleConnsPerHost = 5
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := validExternalConfig()
			test.mutate(&cfg)
			_, err := New(cfg, nil)
			if err == nil {
				t.Fatal("New() error = nil, want non-nil")
			}
			if strings.Contains(err.Error(), "token=secret") {
				t.Fatalf("New() error leaks query secret: %q", err)
			}
		})
	}
}
