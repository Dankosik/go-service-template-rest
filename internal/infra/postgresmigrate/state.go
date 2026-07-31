package postgresmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

type migrationState struct {
	Applied []int64
	Current int64
	Target  int64
}

func loadMigrationState(
	ctx context.Context,
	conn *sql.Conn,
	store database.Store,
	source []sourceMigration,
) (migrationState, error) {
	var tableExists bool
	if err := conn.QueryRowContext(
		ctx,
		"SELECT to_regclass($1) IS NOT NULL",
		goose.DefaultTablename,
	).Scan(&tableExists); err != nil {
		return migrationState{}, fmt.Errorf("inspect goose version table: %w", err)
	}

	var rows []*database.ListMigrationsResult
	if tableExists {
		var err error
		rows, err = store.ListMigrations(ctx, conn)
		if err != nil {
			return migrationState{}, fmt.Errorf("list goose migration history: %w", err)
		}
	}
	return validateMigrationState(rows, tableExists, source)
}

func validateMigrationState(
	newestFirst []*database.ListMigrationsResult,
	tableExists bool,
	source []sourceMigration,
) (migrationState, error) {
	target := int64(0)
	if len(source) > 0 {
		target = source[len(source)-1].Version
	}
	if !tableExists {
		return migrationState{Target: target}, nil
	}
	if len(newestFirst) == 0 {
		return migrationState{}, fmt.Errorf("goose version table exists without bootstrap row")
	}

	applied, err := validateAppliedVersions(newestFirst)
	if err != nil {
		return migrationState{}, err
	}
	if len(applied) > len(source) {
		return migrationState{}, fmt.Errorf("database migration history is ahead of the source")
	}
	for i, version := range applied {
		if source[i].Version != version {
			return migrationState{}, fmt.Errorf(
				"database migration history is not a source prefix at position %d",
				i,
			)
		}
	}

	current := int64(0)
	if len(applied) > 0 {
		current = applied[len(applied)-1]
	}
	return migrationState{Applied: applied, Current: current, Target: target}, nil
}

func validateAppliedVersions(newestFirst []*database.ListMigrationsResult) ([]int64, error) {
	bootstrapCount := 0
	for i, row := range newestFirst {
		if row == nil {
			return nil, fmt.Errorf("goose migration history contains a nil row")
		}
		if row.Version == 0 {
			bootstrapCount++
			if i != len(newestFirst)-1 || !row.IsApplied {
				return nil, fmt.Errorf("goose bootstrap version must be the oldest applied row")
			}
			continue
		}
		if !row.IsApplied {
			return nil, fmt.Errorf("goose migration version %d is not applied", row.Version)
		}
	}
	if bootstrapCount != 1 {
		return nil, fmt.Errorf("goose migration history must contain exactly one bootstrap row")
	}

	applied := make([]int64, 0, len(newestFirst)-1)
	for i := len(newestFirst) - 2; i >= 0; i-- {
		applied = append(applied, newestFirst[i].Version)
	}
	if !slices.IsSortedFunc(applied, func(a, b int64) int {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		default:
			return 0
		}
	}) {
		return nil, fmt.Errorf("goose migration history is not ascending by application order")
	}
	for i := 1; i < len(applied); i++ {
		if applied[i-1] == applied[i] {
			return nil, fmt.Errorf("goose migration version %d is duplicated", applied[i])
		}
	}
	return applied, nil
}
