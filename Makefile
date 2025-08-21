# Elasticsearch Log Trimmer Makefile

# Application info
APP_NAME := log-trimmer
VERSION := 1.0.0
BINARY_NAME := log-trimmer
MAIN_PATH := ./cmd/log-trimmer

# Build info
BUILD_TIME := $(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | cut -d " " -f 3)

# Directories
BUILD_DIR := ./build
DIST_DIR := ./dist
COVERAGE_DIR := ./coverage

# Go build flags
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -X main.GoVersion=$(GO_VERSION)

# Default target
.PHONY: all
all: clean deps build

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Build for multiple platforms
.PHONY: build-all
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux amd64
	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# Linux arm64
	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	
	# macOS amd64
	@echo "Building for macOS amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS arm64
	@echo "Building for macOS arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	# Windows amd64
	@echo "Building for Windows amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

# Run the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Install the binary to $GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(APP_NAME)..."
	go install -ldflags "$(LDFLAGS)" $(MAIN_PATH)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Lint the code
.PHONY: lint
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, please install it: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not found, skipping import formatting"

# Vet the code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf $(COVERAGE_DIR)
	go clean

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Security audit
.PHONY: audit
audit:
	@echo "Running security audit..."
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null || go install golang.org/x/tools/cmd/godoc@latest
	@echo "Documentation server will be available at http://localhost:6060"
	@echo "Run: godoc -http=:6060"

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

# Check if everything is ready for release
.PHONY: pre-release
pre-release: clean deps fmt vet lint test audit build-all
	@echo "Pre-release checks passed!"

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, install deps, and build"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  run          - Build and run the application (use ARGS='...' for arguments)"
	@echo "  install      - Install binary to \$$GOPATH/bin"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  bench        - Run benchmarks"
	@echo "  lint         - Lint the code"
	@echo "  fmt          - Format the code"
	@echo "  vet          - Vet the code"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
	@echo "  update-deps  - Update dependencies"
	@echo "  audit        - Run security audit"
	@echo "  docs         - Generate documentation"
	@echo "  dev-setup    - Setup development environment"
	@echo "  pre-release  - Run all pre-release checks"
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help"

# Example usage targets
.PHONY: example-dry-run
example-dry-run: build
	@echo "Running dry-run example..."
	$(BUILD_DIR)/$(BINARY_NAME) --host https://localhost:9200 --max-age 7d --pattern 'vector-*' --log-level debug

.PHONY: example-with-env
example-with-env: build
	@echo "Running example with environment variables..."
	ES_HOST=https://localhost:9200 MAX_AGE=7d INDEX_PATTERN='vector-*' LOG_LEVEL=info $(BUILD_DIR)/$(BINARY_NAME)
