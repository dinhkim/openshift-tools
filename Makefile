.PHONY: build install clean test lint

BINARY_NAME=openshift-auth-plugin
INSTALL_PATH=/usr/local/bin

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	install -m 755 $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: brew install golangci-lint"; \
		exit 1; \
	fi

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  install  - Build and install to $(INSTALL_PATH)"
	@echo "  clean    - Remove built binary"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  vet      - Run go vet"
	@echo "  deps     - Download and tidy dependencies"
