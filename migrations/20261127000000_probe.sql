-- +goose Up
CREATE SCHEMA migration_probe;

-- +goose Down
DROP SCHEMA migration_probe;