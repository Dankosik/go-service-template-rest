package bootstrap

import (
	"strings"
	"testing"
)

func TestParseLoadOptions(t *testing.T) {
	t.Parallel()

	t.Run("parses flags", func(t *testing.T) {
		t.Parallel()

		opts, err := parseLoadOptions([]string{
			"--config", "/tmp/base.yaml",
			"--config-overlay", "  /tmp/o1.yaml  ",
			"--config-overlay", "/tmp/o2.yaml",
			"--config-strict",
		})
		if err != nil {
			t.Fatalf("parseLoadOptions() error = %v, want nil", err)
		}
		if opts.ConfigPath != "/tmp/base.yaml" {
			t.Fatalf("ConfigPath = %q, want %q", opts.ConfigPath, "/tmp/base.yaml")
		}
		if !opts.Strict {
			t.Fatal("Strict = false, want true")
		}
		if len(opts.ConfigOverlays) != 2 || opts.ConfigOverlays[0] != "/tmp/o1.yaml" || opts.ConfigOverlays[1] != "/tmp/o2.yaml" {
			t.Fatalf("ConfigOverlays = %v, want trimmed [/tmp/o1.yaml /tmp/o2.yaml]", opts.ConfigOverlays)
		}
	})

	t.Run("fails on empty overlay path", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--config-overlay", "   "})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--unknown-flag"})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
	})

	t.Run("fails on positional arguments", func(t *testing.T) {
		t.Parallel()

		_, err := parseLoadOptions([]string{"--config", "/tmp/base.yaml", "serve"})
		if err == nil {
			t.Fatal("parseLoadOptions() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse flags") {
			t.Fatalf("parseLoadOptions() err = %v, want parse flags context", err)
		}
		if !strings.Contains(err.Error(), "serve") {
			t.Fatalf("parseLoadOptions() err = %v, want unexpected positional argument detail", err)
		}
	})
}

func TestRunReturnsParseErrorForInvalidFlags(t *testing.T) {
	t.Parallel()

	err := Run([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "parse flags") {
		t.Fatalf("Run() err = %v, want parse flags context", err)
	}
}
