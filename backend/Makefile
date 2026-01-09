.PHONY: help deps build build-api build-fetcher run-api run-fetcher test lint clean

# Default target
help:
	@echo "Aegis v13 Backend - Available targets:"
	@echo "  make deps          - Install dependencies"
	@echo "  make build         - Build all binaries"
	@echo "  make build-api     - Build API server"
	@echo "  make build-fetcher - Build fetcher CLI"
	@echo "  make run-api       - Run API server"
	@echo "  make run-fetcher   - Run fetcher (collect all)"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make clean         - Clean build artifacts"

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build all binaries
build: build-api build-fetcher

# Build API server
build-api:
	@echo "Building API server..."
	go build -o bin/api ./cmd/api

# Build fetcher CLI
build-fetcher:
	@echo "Building fetcher CLI..."
	go build -o bin/fetcher ./cmd/fetcher

# Run API server
run-api:
	@echo "Running API server..."
	go run ./cmd/api

# Run fetcher (collect all data)
run-fetcher:
	@echo "Running fetcher..."
	go run ./cmd/fetcher collect all

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean
