package postgresmigrate

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

func rootMigrationSource(source fs.FS, sourcePath string) (fs.FS, error) {
	if source == nil {
		return nil, errors.New("migration source filesystem is required")
	}
	path := strings.TrimSpace(sourcePath)
	if path == "" || path == "." || !fs.ValidPath(path) {
		return nil, fmt.Errorf("migration source path %q is invalid", sourcePath)
	}

	rooted, err := fs.Sub(source, path)
	if err != nil {
		return nil, fmt.Errorf("open migration source %q: %w", sourcePath, err)
	}
	if _, err := fs.Stat(rooted, "."); err != nil {
		return nil, fmt.Errorf("open migration source %q: %w", sourcePath, err)
	}
	return rooted, nil
}
