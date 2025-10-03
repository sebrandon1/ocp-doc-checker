.PHONY: build test test-unit test-integration test-ci clean install fmt lint help

BINARY_NAME=ocp-doc-checker
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

all: test build

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	@echo "Running Go tests..."
	go test -v ./...
	@echo ""
	@echo "Running integration tests..."
	@if [ -f "./$(BINARY_NAME)" ]; then \
		./test.sh; \
	else \
		$(MAKE) build && ./test.sh; \
	fi

test-unit:
	@echo "Running Go unit tests only..."
	go test -v ./...

test-integration:
	@echo "Running integration tests..."
	@if [ ! -f "./$(BINARY_NAME)" ]; then \
		$(MAKE) build; \
	fi
	./test.sh

test-ci:
	@echo "Running all CI tests..."
	@if [ ! -f "./$(BINARY_NAME)" ]; then \
		$(MAKE) build; \
	fi
	@echo ""
	@echo "================================================"
	@./scripts/test-outdated-url.sh || (echo "Test 1 failed" && exit 1)
	@echo ""
	@echo "================================================"
	@./scripts/test-current-url.sh || (echo "Test 2 failed" && exit 1)
	@echo ""
	@echo "================================================"
	@./scripts/test-url-accessibility.sh || (echo "Test 3 failed" && exit 1)
	@echo ""
	@echo "================================================"
	@./scripts/test-invalid-url.sh || (echo "Test 4 failed" && exit 1)
	@echo ""
	@echo "================================================"
	@./scripts/test-url-replacement.sh || (echo "Test 5 failed" && exit 1)
	@echo ""
	@echo "================================================"
	@echo "✅ All CI tests passed!"

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f test*_output.json
	rm -f test*_output.txt
	go clean

install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

help:
	@echo "Available targets:"
	@echo "  build            - Build the binary"
	@echo "  test             - Run all tests (unit + integration)"
	@echo "  test-unit        - Run only Go unit tests"
	@echo "  test-integration - Run only integration tests"
	@echo "  test-ci          - Run all CI test scripts"
	@echo "  clean            - Remove build artifacts"
	@echo "  install          - Install the binary to GOPATH/bin"
	@echo "  fmt              - Format code"
	@echo "  lint             - Run linters"
	@echo "  help             - Show this help message"

