package postgresmigrate

import (
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var migrationFilenamePattern = regexp.MustCompile(`^([0-9]{6})_[a-z][a-z0-9]*(_[a-z0-9]+)*\.sql$`)

type sourceMigration struct {
	Version  int64
	Filename string
}

type admittedSource struct {
	FS         fs.FS
	Migrations []sourceMigration
}

func admitSource(sourceFS fs.FS, sourcePath string) (admittedSource, error) {
	normalizedPath, err := normalizeMigrationSourcePath(sourcePath)
	if err != nil {
		return admittedSource{}, err
	}
	if sourceFS == nil {
		sourceFS = os.DirFS("/")
	}

	rooted, err := fs.Sub(sourceFS, normalizedPath)
	if err != nil {
		return admittedSource{}, fmt.Errorf("open migration source %q: %w", sourcePath, err)
	}
	entries, err := fs.ReadDir(rooted, ".")
	if err != nil {
		return admittedSource{}, fmt.Errorf("read migration source %q: %w", sourcePath, err)
	}

	migrations := make([]sourceMigration, 0, len(entries))
	seenVersions := make(map[int64]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			return admittedSource{}, fmt.Errorf("migration source contains nested directory %q", entry.Name())
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return admittedSource{}, fmt.Errorf("migration source contains symbolic link %q", entry.Name())
		}
		if entry.Type()&fs.ModeType != 0 {
			return admittedSource{}, fmt.Errorf("migration source contains non-regular file %q", entry.Name())
		}

		matches := migrationFilenamePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return admittedSource{}, fmt.Errorf("migration filename %q is not canonical", entry.Name())
		}
		version, parseErr := strconv.ParseInt(matches[1], 10, 64)
		if parseErr != nil || version == 0 {
			return admittedSource{}, fmt.Errorf("migration filename %q has invalid version", entry.Name())
		}
		if earlier, exists := seenVersions[version]; exists {
			return admittedSource{}, fmt.Errorf(
				"migration version %06d is duplicated by %q and %q",
				version,
				earlier,
				entry.Name(),
			)
		}
		if err := validateMigrationFile(rooted, entry.Name()); err != nil {
			return admittedSource{}, err
		}
		seenVersions[version] = entry.Name()
		migrations = append(migrations, sourceMigration{Version: version, Filename: entry.Name()})
	}

	slices.SortFunc(migrations, func(a, b sourceMigration) int {
		return strings.Compare(a.Filename, b.Filename)
	})
	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Version >= migrations[i].Version {
			return admittedSource{}, fmt.Errorf("migration versions are not strictly increasing by filename")
		}
	}
	return admittedSource{FS: rooted, Migrations: migrations}, nil
}

func validateMigrationFile(source fs.FS, filename string) error {
	content, err := fs.ReadFile(source, filename)
	if err != nil {
		return fmt.Errorf("read migration %q: %w", filename, err)
	}

	var upCount, downCount, upLine, downLine int
	for lineNumber, line := range strings.Split(string(content), "\n") {
		annotation := strings.TrimSuffix(line, "\r")
		switch annotation {
		case "-- +goose Up":
			upCount++
			upLine = lineNumber
		case "-- +goose Down":
			downCount++
			downLine = lineNumber
		case "-- +goose NO TRANSACTION":
			return fmt.Errorf("migration %q cannot disable transactions", filename)
		case "-- +goose ENVSUB ON", "-- +goose ENVSUB OFF":
			return fmt.Errorf("migration %q cannot enable environment substitution", filename)
		case "-- +goose StatementBegin", "-- +goose StatementEnd":
			// Goose statement blocks are valid and do not change transaction policy.
		default:
			if strings.HasPrefix(strings.TrimSpace(annotation), "--") &&
				strings.Contains(annotation, "+goose") {
				return fmt.Errorf("migration %q contains non-canonical goose annotation %q", filename, annotation)
			}
		}
	}
	if upCount != 1 || downCount != 1 {
		return fmt.Errorf(
			"migration %q must contain exactly one goose Up and one goose Down annotation",
			filename,
		)
	}
	if upLine >= downLine {
		return fmt.Errorf("migration %q must place goose Up before goose Down", filename)
	}
	return nil
}

func normalizeMigrationSourcePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("migration source path is empty")
	}

	normalized := pathpkg.Clean(trimmed)
	if pathpkg.IsAbs(normalized) {
		normalized = strings.TrimPrefix(normalized, "/")
	} else if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", fmt.Errorf("migration source path %q escapes its filesystem root", raw)
	}
	if normalized == "." || normalized == "" {
		return "", fmt.Errorf("migration source path is empty")
	}
	if !fs.ValidPath(normalized) {
		return "", fmt.Errorf("migration source path %q is invalid", raw)
	}
	return normalized, nil
}
