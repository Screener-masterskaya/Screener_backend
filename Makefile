APP_NAME := screener
BIN_PATH := bin/$(APP_NAME)

.PHONY: run build test lint

run:
	go run ./cmd/$(APP_NAME)

build:
	mkdir -p bin
	go build -o $(BIN_PATH) ./cmd/$(APP_NAME)

test:
	go test -race ./...

lint:
	go vet ./...