APP_NAME := screener
BIN_PATH := bin/$(APP_NAME)
CONFIG_PATH ?= config.yaml

.PHONY: run build test lint

run:
	CONFIG_PATH=$(CONFIG_PATH) go run ./cmd/$(APP_NAME)

build:
	mkdir -p bin
	go build -o $(BIN_PATH) ./cmd/$(APP_NAME)

test:
	go test -race ./...

lint:
	go vet ./...