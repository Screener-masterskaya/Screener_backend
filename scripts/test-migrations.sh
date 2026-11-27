#!/bin/sh
set -eu

test_dir=$(mktemp -d)
project="screener-migrations-test-$$"

cat > "$test_dir/compose.yml" <<'YAML'
services:
  postgres:
    container_name: !reset null
    ports: !reset []
    environment:
      POSTGRES_USER: migrations_test
      POSTGRES_PASSWORD: migrations_test
      POSTGRES_DB: migrations_test
  migrate:
    environment:
      GOOSE_DBSTRING: postgres://migrations_test:migrations_test@postgres:5432/migrations_test
YAML

export COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}:$test_dir/compose.yml"

compose() {
    docker compose --project-name "$project" "$@"
}

cleanup() {
    compose down --volumes >/dev/null || true
    rm -rf "$test_dir"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

migrate() {
    compose run --rm -T migrate "$@"
}

query() {
    compose exec -T postgres psql -X -Aqt -v ON_ERROR_STOP=1 \
        -U migrations_test -d migrations_test -c "$1"
}

check_probe() {
    actual=$(query "SELECT to_regnamespace('migration_probe') IS NOT NULL")
    if [ "$actual" != "$1" ]; then
        echo "Expected migration_probe existence to be $1, got $actual" >&2
        exit 1
    fi
}

compose build migrate
compose up -d --wait postgres
version=20261127000000

check_probe f
migrate up-to "$version"
check_probe t

history=$(query 'SELECT * FROM goose_db_version ORDER BY id')
migrate up-to "$version"
test "$history" = "$(query 'SELECT * FROM goose_db_version ORDER BY id')"

migrate down
check_probe f
migrate up-to "$version"
check_probe t