# Project info
APP_NAME := linkstash
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Directories
BIN_DIR := bin
DATA_DIR := data
CONF := conf/app_dev.yaml

.PHONY: all build build-server build-cli clean run stop restart test smoke-test wire tidy lint fmt help

## Default target
all: build

## Build both server and CLI
build: build-server build-cli

build-server:
	@echo ">>> Building server..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/linkstash-server ./cmd/server/

build-cli:
	@echo ">>> Building CLI..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/linkstash ./cmd/cli/

## Run server (foreground)
run: build-server
	@mkdir -p $(DATA_DIR)
	. $$HOME/.my.env && $(BIN_DIR)/linkstash-server -conf $(CONF)

## Run server in background
start: build-server
	@mkdir -p $(DATA_DIR)
	@echo ">>> Starting LinkStash server..."
	@. $$HOME/.my.env && nohup $(BIN_DIR)/linkstash-server -conf $(CONF) > /tmp/linkstash.log 2>&1 & echo $$! > /tmp/linkstash.pid
	@echo "Server started (PID: $$(cat /tmp/linkstash.pid)), log: /tmp/linkstash.log"

## Stop background server (only kills linkstash-server processes)
stop:
	@if [ -f /tmp/linkstash.pid ]; then \
		PID=$$(cat /tmp/linkstash.pid); \
		if kill -0 $$PID 2>/dev/null; then \
			kill $$PID && echo "Server stopped (PID: $$PID)"; \
		else \
			echo "PID $$PID not running"; \
		fi; \
		rm -f /tmp/linkstash.pid; \
	else \
		echo "No PID file found, trying pkill..."; \
		pkill -f 'linkstash-server' 2>/dev/null && echo "Server stopped" || echo "Server not running"; \
	fi

## Restart server
restart: stop
	@sleep 1
	@$(MAKE) start

## Run smoke test (starts server, tests, stops)
smoke-test: build
	@echo ">>> Running smoke tests..."
	@./scripts/smoke_test.sh

## Run go test
test:
	go test -v -race ./...

## Generate wire code
wire:
	@echo ">>> Generating wire code..."
	cd app/di && wire

## go mod tidy
tidy:
	go mod tidy

## Format code
fmt:
	gofmt -s -w .

## Lint
lint:
	@which golangci-lint > /dev/null 2>&1 || echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	golangci-lint run ./...

## Clean build artifacts
clean:
	rm -rf $(BIN_DIR)
	rm -f linkstash linkstash-server
	rm -f /tmp/linkstash.pid /tmp/linkstash.log

## Cross-compile for all platforms
release:
	@echo ">>> Building release binaries..."
	@mkdir -p $(BIN_DIR)/release
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-server-linux-amd64 ./cmd/server/
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-linux-amd64 ./cmd/cli/
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-server-linux-arm64 ./cmd/server/
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-linux-arm64 ./cmd/cli/
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-server-darwin-amd64 ./cmd/server/
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-darwin-amd64 ./cmd/cli/
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-server-darwin-arm64 ./cmd/server/
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/release/linkstash-darwin-arm64 ./cmd/cli/
	@echo ">>> Release binaries in $(BIN_DIR)/release/"
	@ls -lh $(BIN_DIR)/release/

## Show help
help:
	@echo "LinkStash Makefile Targets:"
	@echo ""
	@echo "  make build        Build server + CLI"
	@echo "  make run          Build and run server (foreground)"
	@echo "  make start        Build and start server (background)"
	@echo "  make stop         Stop background server"
	@echo "  make restart      Restart background server"
	@echo "  make smoke-test   Run smoke test suite"
	@echo "  make test         Run Go unit tests"
	@echo "  make wire         Generate wire DI code"
	@echo "  make tidy         Run go mod tidy"
	@echo "  make fmt          Format code"
	@echo "  make lint         Run linter"
	@echo "  make release      Cross-compile release binaries"
	@echo "  make clean        Clean build artifacts"
	@echo "  make help         Show this help"
