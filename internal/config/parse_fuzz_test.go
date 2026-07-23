package config

import (
	"testing"
	"time"
)

func FuzzParseDuration(f *testing.F) {
	for _, seed := range []string{
		"",
		"0s",
		"1s",
		"1h30m",
		"-2ms",
		"42",
		"1",
		"1us",
		"999999999999999999999h",
		"not-a-duration",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		got, err := parseDuration(raw)
		want, wantErr := time.ParseDuration(raw)
		if wantErr != nil {
			if err == nil {
				t.Fatalf("parseDuration(%q) error = nil, want sanitized parse error", raw)
			}
			switch detail := err.Error(); detail {
			case "missing duration unit", "invalid duration syntax":
			default:
				t.Fatalf("parseDuration(%q) error = %q, want sanitized detail", raw, detail)
			}
			return
		}
		if err != nil {
			t.Fatalf("parseDuration(%q) error = %v, want nil", raw, err)
		}
		if got != want {
			t.Fatalf("parseDuration(%q) = %s, want %s", raw, got, want)
		}
	})
}
