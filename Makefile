.PHONY: all build clean install test lint fmt run clean-build

# Variables
BINARY_NAME=siby
VERSION?=0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Go commands
GO=go
GOFMT=$(GO) fmt
GOVET=$(GO) vet
GOBUILD=$(GO) build $(LDFLAGS)

# Directories
BUILD_DIR=./bin
CMD_DIR=./cmd/siby-agentiq

# OS/ARCH combinations
LINUX_AMD64=linux-amd64/$(BINARY_NAME)
LINUX_ARM64=linux-arm64/$(BINARY_NAME)
MAC_AMD64=darwin-amd64/$(BINARY_NAME)
MAC_ARM64=darwin-arm64/$(BINARY_NAME)
WINDOWS=windows-amd64/$(BINARY_NAME).exe

all: clean test build

build: build-linux build-mac build-windows

# Local builds
build-local:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-local.exe:
	@echo "Building $(BINARY_NAME).exe..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(CMD_DIR)

# Cross-compilation
build-linux:
	@echo "Building Linux binaries..."
	@mkdir -p $(BUILD_DIR)/linux-amd64 $(BUILD_DIR)/linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(LINUX_AMD64) $(CMD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(LINUX_ARM64) $(CMD_DIR)

build-mac:
	@echo "Building macOS binaries..."
	@mkdir -p $(BUILD_DIR)/darwin-amd64 $(BUILD_DIR)/darwin-arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(MAC_AMD64) $(CMD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(MAC_ARM64) $(CMD_DIR)

build-windows:
	@echo "Building Windows binary..."
	@mkdir -p $(BUILD_DIR)/windows-amd64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(WINDOWS) $(CMD_DIR)

# Static builds (no CGO)
build-static:
	@echo "Building static binaries..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Testing
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Linting
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Formatting
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

vet:
	@echo "Running vet..."
	$(GOVET) ./...

# Cleanup
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

clean-build:
	rm -f $(BINARY_NAME) *.exe

# Installation
install: build-local
	@echo "Installing..."
	install -D -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

install-home:
	@echo "Installing to ~/.local/bin..."
	mkdir -p $(HOME)/.local/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/
	@echo "Add to PATH: export PATH=\$$HOME/.local/bin:\$$PATH"

# Development
dev:
	@echo "Running in development mode..."
	$(GO) run $(CMD_DIR)

watch:
	@echo "Watching for changes..."
	reflex -r '\.go$' -s -- go run $(CMD_DIR)

# Docker
docker-build:
	docker build -t siby-terminal:latest .

docker-run:
	docker run --rm -it siby-terminal:latest

# Help
help:
	@echo "Siby-Terminal Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Clean, test and build all"
	@echo "  build        - Build for all platforms"
	@echo "  build-local  - Build for current platform"
	@echo "  build-static - Build static binaries"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install system-wide"
	@echo "  install-home - Install to ~/.local/bin"
	@echo "  dev          - Run in development"
	@echo "  help         - Show this help"
