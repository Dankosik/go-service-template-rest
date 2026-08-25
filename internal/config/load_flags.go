package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// ParseLoadOptions reads the configuration flags every binary in this repository
// accepts out of args: -config once, and -config-overlay repeatably in the order
// the overlays are applied.
//
// Usage output is discarded because a composition root reports its own startup
// rejection through the process logger, and flag's default output would print a
// second, unstructured copy to stderr.
func ParseLoadOptions(args []string) (LoadOptions, error) {
	var configPath string
	var overlays []string

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Func("config", "path to base config file", func(value string) error {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return errors.New("config path cannot be empty")
		}
		configPath = trimmed
		return nil
	})
	flags.Func("config-overlay", "path to config overlay file (repeatable)", func(value string) error {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return errors.New("config overlay path cannot be empty")
		}
		overlays = append(overlays, trimmed)
		return nil
	})
	if err := flags.Parse(args); err != nil {
		return LoadOptions{}, fmt.Errorf("parse flags: %w", err)
	}
	// Refused rather than ignored: a positional argument is a mistyped flag often
	// enough that accepting one silently starts a process with the configuration
	// the operator thought they were replacing.
	if positional := flags.Args(); len(positional) > 0 {
		return LoadOptions{}, fmt.Errorf("parse flags: unexpected positional arguments: %v", positional)
	}
	return LoadOptions{ConfigPath: configPath, ConfigOverlays: overlays}, nil
}
