APP_NAME := screener
BIN_PATH := bin/$(APP_NAME)

.PHONY: run build test lint up down

run:
	go run ./cmd/$(APP_NAME)

build:
	mkdir -p bin
	go build -o $(BIN_PATH) ./cmd/$(APP_NAME)

test:
	go test -race ./...

lint:
	go vet ./...

up:
	docker compose up -d --build --wait

down:
	docker compose down
.PHONY: migrate-up migrate-down migrate-status migrate-create test-migrations

export name

migrate-up migrate-down migrate-status:
	docker compose run --rm -T --build migrate $(@:migrate-%=%)

migrate-create:
	@test -n "$$name" || { echo 'Usage: make migrate-create name=migration_name' >&2; exit 1; }
	docker compose run --rm -T --build --no-deps --user "$$(id -u):$$(id -g)" migrate create "$$name" sql

test-migrations:
	sh scripts/test-migrations.sh