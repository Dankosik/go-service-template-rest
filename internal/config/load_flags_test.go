package config

import (
	"slices"
	"strings"
	"testing"
)

// TestParseLoadOptionsAcceptsTheSharedFlagSurface pins what every binary in this
// repository accepts, which is the reason the parser is shared: a change here is
// a change to all of them at once.
func TestParseLoadOptionsAcceptsTheSharedFlagSurface(t *testing.T) {
	t.Parallel()

	options, err := ParseLoadOptions([]string{
		"--config", "base.yaml",
		"--config-overlay", "one.yaml",
		"--config-overlay=two.yaml",
	})
	if err != nil {
		t.Fatalf("ParseLoadOptions() error = %v", err)
	}
	if options.ConfigPath != "base.yaml" {
		t.Errorf("ConfigPath = %q, want base.yaml", options.ConfigPath)
	}
	// Order matters: overlays are applied in the order they were given, so the
	// last one wins on a key two of them set.
	if want := []string{"one.yaml", "two.yaml"}; !slices.Equal(options.ConfigOverlays, want) {
		t.Errorf("ConfigOverlays = %v, want %v", options.ConfigOverlays, want)
	}
}

func TestParseLoadOptionsRefusesWhatWouldStartTheWrongProcess(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		args []string
	}{
		{name: "blank config path", args: []string{"--config", "   "}},
		{name: "blank overlay path", args: []string{"--config-overlay", "   "}},
		{name: "unknown flag", args: []string{"--unknown-flag"}},
		// A positional argument is a mistyped flag often enough that accepting one
		// would start a process with the configuration it was meant to replace.
		{name: "positional argument", args: []string{"--config", "base.yaml", "serve"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseLoadOptions(testCase.args)
			if err == nil {
				t.Fatalf("ParseLoadOptions(%q) error = nil, want a rejection", testCase.args)
			}
			if !strings.Contains(err.Error(), "parse flags") {
				t.Errorf("ParseLoadOptions(%q) err = %v, want the parse flags stage named", testCase.args, err)
			}
		})
	}
}
