.PHONY: help build run test clean fmt lint deps

# Variables
BINARY_NAME=xarb
GO=go
MAIN_PACKAGE=./cmd/$(BINARY_NAME)

help:
	@echo "Available commands:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application"
	@echo "  make test        - Run all tests"
	@echo "  make test-unit   - Run unit tests only"
	@echo "  make test-sqlite - Run SQLite integration tests"
	@echo "  make test-cover  - Run tests with coverage report"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linter"
	@echo "  make deps        - Download and manage dependencies"
	@echo "  make tidy        - Tidy dependencies"

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: ./$(BINARY_NAME)"

# Build Linux binary
build-linux:
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux $(MAIN_PACKAGE)
	@echo "Build complete: ./$(BINARY_NAME)-linux"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) -config configs/config.toml

# Run all tests
test:
	@echo "Running all tests..."
	$(GO) test ./... -v

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GO) test ./internal/application/service/... -v

# Run SQLite integration tests
test-sqlite:
	@echo "Running SQLite integration tests..."
	$(GO) test ./internal/infrastructure/storage/sqlite/... -v

# Run container integration tests
test-container:
	@echo "Running container integration tests..."
	$(GO) test ./internal/application/container/... -v

# Generate coverage report
test-cover:
	@echo "Running tests with coverage..."
	$(GO) test ./... -v -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Format complete"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./... || true

# Download and manage dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies downloaded"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy
	@echo "Dependencies tidied"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f *.db
	$(GO) clean
	@echo "Clean complete"

# Run all checks before commit
check: fmt lint test
	@echo "All checks passed!"

# Build and run
dev: build run

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"
