package postgresmigrate

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

var (
	canonicalMigrationFilename = regexp.MustCompile(`^\d+_[a-z0-9]+(?:_[a-z0-9]+)*\.sql$`)
	gooseNoTransaction         = regexp.MustCompile(`(?im)^[\t ]*--[\t ]+\+goose[\t ]+no[\t ]+transaction[\t ]*\r?$`)
	gooseEnvSub                = regexp.MustCompile(`(?im)^[\t ]*--[\t ]+\+goose[\t ]+envsub(?:[\t ]+(?:on|off))?[\t ]*\r?$`)
)

func rootMigrationSource(source fs.FS, sourcePath string) (fs.FS, error) {
	if source == nil {
		return nil, errors.New("migration source filesystem is required")
	}
	path := strings.TrimSpace(sourcePath)
	if path == "" || path == "." || !fs.ValidPath(path) {
		return nil, fmt.Errorf("migration source path %q is invalid", sourcePath)
	}
	info, err := fs.Lstat(source, path)
	if err != nil {
		return nil, fmt.Errorf("open migration source %q: %w", sourcePath, err)
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, fmt.Errorf("migration source %q must not be a symbolic link", sourcePath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("migration source %q is not a directory", sourcePath)
	}

	rooted, err := fs.Sub(source, path)
	if err != nil {
		return nil, fmt.Errorf("open migration source %q: %w", sourcePath, err)
	}
	if err := validateMigrationSource(rooted); err != nil {
		return nil, err
	}
	return rooted, nil
}

func validateMigrationSource(source fs.FS) error {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return fmt.Errorf("read migration source: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		version, _, _ := strings.Cut(name, "_")
		info, err := fs.Lstat(source, name)
		if err != nil {
			return fmt.Errorf("inspect migration source entry %q: %w", name, err)
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			return fmt.Errorf("migration source entry %q must not be a symbolic link", name)
		case info.IsDir():
			return fmt.Errorf("migration source contains nested directory %q", name)
		case !info.Mode().IsRegular():
			return fmt.Errorf("migration source entry %q is not a regular file", name)
		case !canonicalMigrationFilename.MatchString(name) || strings.TrimLeft(version, "0") == "":
			return fmt.Errorf("migration filename %q is not canonical", name)
		}

		contents, err := fs.ReadFile(source, name)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}
		switch {
		case gooseNoTransaction.Match(contents):
			return fmt.Errorf("migration %q disables transactions", name)
		case gooseEnvSub.Match(contents):
			return fmt.Errorf("migration %q enables environment substitution", name)
		}
	}
	return nil
}
