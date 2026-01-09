.PHONY: help deps build test lint clean test-db test-logger

# Default target
help:
	@echo "Aegis v13 Backend - Available targets:"
	@echo "  make deps          - Install dependencies"
	@echo "  make build         - Build unified CLI binary (bin/quant)"
	@echo "  make test          - Run all tests"
	@echo "  make test-db       - Test database connection"
	@echo "  make test-logger   - Test logger functionality"
	@echo "  make lint          - Run linter"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "Run commands (development):"
	@echo "  go run ./cmd/quant [command]"
	@echo ""
	@echo "Available commands:"
	@echo "  go run ./cmd/quant api              - Run API server"
	@echo "  go run ./cmd/quant fetcher collect  - Run data fetcher"
	@echo "  go run ./cmd/quant test-db          - Test database"
	@echo "  go run ./cmd/quant test-logger      - Test logger"

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build unified CLI binary
build:
	@echo "Building unified CLI (quant)..."
	@mkdir -p bin
	go build -o bin/quant ./cmd/quant
	@echo "✅ Built: bin/quant"
	@echo ""
	@echo "Usage:"
	@echo "  ./bin/quant [command]"
	@echo ""
	@echo "Examples:"
	@echo "  ./bin/quant api"
	@echo "  ./bin/quant fetcher collect all"
	@echo "  ./bin/quant test-db"
	@echo "  ./bin/quant test-logger"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Test database connection
test-db:
	@echo "Testing database connection..."
	go run ./cmd/quant test-db

# Test logger functionality
test-logger:
	@echo "Testing logger..."
	go run ./cmd/quant test-logger

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
