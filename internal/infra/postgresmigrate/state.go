package postgresmigrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

func emptySourceVersion(
	ctx context.Context,
	conn *sql.Conn,
	store database.Store,
) (int64, error) {
	var tableExists bool
	if err := conn.QueryRowContext(
		ctx,
		"SELECT to_regclass($1) IS NOT NULL",
		goose.DefaultTablename,
	).Scan(&tableExists); err != nil {
		return 0, fmt.Errorf("inspect goose version table: %w", err)
	}
	if !tableExists {
		return 0, nil
	}

	version, err := store.GetLatestVersion(ctx, conn)
	if err != nil {
		return 0, fmt.Errorf("read goose migration version: %w", err)
	}
	if version != 0 {
		return version, fmt.Errorf("database migration history is ahead of the empty source at version %d", version)
	}
	return 0, nil
}
