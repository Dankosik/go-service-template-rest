-- +goose Up

-- River v0.49.0 migration 008 is a no-op on PostgreSQL. Record it so River's
-- ledger keeps matching the vendored schema and its tooling still validates it.
INSERT INTO /* TEMPLATE: schema */river_migration (line, version) VALUES ('main', 8);

-- +goose Down

DELETE FROM /* TEMPLATE: schema */river_migration WHERE line = 'main' AND version = 8;
