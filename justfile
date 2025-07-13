# itsjustintv - Just commands for development workflow
# https://github.com/casey/just

# Default recipe to display available commands
default:
    @just --list

# Variables
binary_name := "itsjustintv"
main_package := "./cmd/itsjustintv"
config_file := "config.example.toml"

# Build the application
build:
    @echo "Building {{binary_name}}..."
    go build -o {{binary_name}} {{main_package}}

# Build with version information (like CI does)
build-release version="dev":
    @echo "Building {{binary_name}} with version {{version}}..."
    #!/usr/bin/env bash
    COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    go build -ldflags "-s -w -X github.com/rmoriz/itsjustintv/internal/cli.Version={{version}} -X github.com/rmoriz/itsjustintv/internal/cli.GitCommit=${COMMIT} -X github.com/rmoriz/itsjustintv/internal/cli.BuildDate=${BUILD_DATE}" -o {{binary_name}} {{main_package}}

# Run the application with example config
run: build
    @echo "Running {{binary_name}} with example config..."
    ./{{binary_name}} --config {{config_file}} --verbose

# Run the application directly with go run
run-dev:
    @echo "Running {{binary_name}} in development mode..."
    go run {{main_package}} --config {{config_file}} --verbose

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests and show coverage in terminal
test-cover:
    @echo "Running tests with coverage..."
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run linting (requires golangci-lint to be installed)
lint:
    @echo "Running linter..."
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run; \
    else \
        echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
        exit 1; \
    fi

# Install golangci-lint
install-lint:
    @echo "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...

# Tidy dependencies
tidy:
    @echo "Tidying dependencies..."
    go mod tidy

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -f {{binary_name}}
    rm -f coverage.out coverage.html
    rm -f itsjustintv-*
    go clean

# Run all checks (format, lint, test)
check: fmt lint test

# Build Docker image
docker-build tag="itsjustintv:latest":
    @echo "Building Docker image {{tag}}..."
    docker build -t {{tag}} .

# Run Docker container
docker-run tag="itsjustintv:latest" port="8080":
    @echo "Running Docker container {{tag}} on port {{port}}..."
    docker run -p {{port}}:8080 -v $(pwd)/{{config_file}}:/app/config.toml {{tag}} --config config.toml

# Show version information
version:
    @if [ -f "./{{binary_name}}" ]; then \
        ./{{binary_name}} version; \
    else \
        echo "Binary not found. Run 'just build' first."; \
    fi

# Show help for the application
help: build
    ./{{binary_name}} --help

# Generate config file from example
config:
    @if [ ! -f "config.toml" ]; then \
        cp {{config_file}} config.toml; \
        echo "Created config.toml from {{config_file}}"; \
        echo "Please edit config.toml with your Twitch credentials"; \
    else \
        echo "config.toml already exists"; \
    fi

# Watch for changes and rebuild (requires entr: brew install entr or apt install entr)
watch:
    @echo "Watching for changes... (requires 'entr' to be installed)"
    @if command -v entr >/dev/null 2>&1; then \
        find . -name "*.go" | entr -r just run-dev; \
    else \
        echo "entr not found. Install it with: brew install entr (macOS) or apt install entr (Ubuntu)"; \
        exit 1; \
    fi

# Install development tools
install-tools:
    @echo "Installing development tools..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "Development tools installed!"

# Run integration tests (if any)
test-integration:
    @echo "Running integration tests..."
    go test -v -tags=integration ./...

# Build for multiple platforms (like CI does)
build-all version="dev":
    @echo "Building for multiple platforms..."
    #!/usr/bin/env bash
    COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    LDFLAGS="-s -w -X github.com/rmoriz/itsjustintv/internal/cli.Version={{version}} -X github.com/rmoriz/itsjustintv/internal/cli.GitCommit=${COMMIT} -X github.com/rmoriz/itsjustintv/internal/cli.BuildDate=${BUILD_DATE}"
    
    echo "Building for linux/amd64..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o {{binary_name}}-linux-amd64 {{main_package}}
    
    echo "Building for linux/arm64..."
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o {{binary_name}}-linux-arm64 {{main_package}}
    
    echo "Building for darwin/arm64..."
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o {{binary_name}}-darwin-aarch64 {{main_package}}
    
    echo "Built binaries:"
    ls -la {{binary_name}}-*

# Prepare for release (run all checks and build)
release version: check
    @echo "Preparing release {{version}}..."
    just build-release {{version}}
    just test-coverage
    @echo "Release {{version}} ready!"

# Quick development cycle: format, test, build, run
dev: fmt test build run-dev