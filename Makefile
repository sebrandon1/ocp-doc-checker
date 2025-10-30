.PHONY: build test test-unit test-integration test-ci clean install fmt lint build-image help scan-openshift scan-redhat-openshift-ecosystem scan-openshift-kni scan-redhatci

BINARY_NAME=ocp-doc-checker
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

IMAGE_NAME?=ocp-doc-checker
IMAGE_TAG?=$(VERSION)
IMAGE_REGISTRY?=quay.io
IMAGE_REPO?=$(IMAGE_REGISTRY)/$(IMAGE_NAME)

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

build-image:
	@echo "Building Docker image $(IMAGE_REPO):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_REPO):$(IMAGE_TAG) .
	@echo "Tagging as $(IMAGE_REPO):latest..."
	docker tag $(IMAGE_REPO):$(IMAGE_TAG) $(IMAGE_REPO):latest

scan-openshift:
	@echo "Scanning openshift org with --fix and linking to issue #18..."
	./scripts/scan-org-for-ocp-docs.sh --fix --link-to https://github.com/sebrandon1/ocp-doc-checker/issues/18 openshift

scan-redhat-openshift-ecosystem:
	@echo "Scanning redhat-openshift-ecosystem org with --fix and linking to issue #23..."
	./scripts/scan-org-for-ocp-docs.sh --fix --link-to https://github.com/sebrandon1/ocp-doc-checker/issues/23 redhat-openshift-ecosystem

scan-openshift-kni:
	@echo "Scanning openshift-kni org with --fix and linking to issue #21..."
	./scripts/scan-org-for-ocp-docs.sh --fix --link-to https://github.com/sebrandon1/ocp-doc-checker/issues/21 openshift-kni

scan-redhatci:
	@echo "Scanning redhatci org with --fix and linking to issue #22..."
	./scripts/scan-org-for-ocp-docs.sh --fix --link-to https://github.com/sebrandon1/ocp-doc-checker/issues/22 redhatci

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
	@echo "  build-image      - Build Docker image"
	@echo "  scan-openshift   - Scan openshift org with --fix (issue #18)"
	@echo "  scan-redhat-openshift-ecosystem - Scan redhat-openshift-ecosystem org with --fix (issue #23)"
	@echo "  scan-openshift-kni - Scan openshift-kni org with --fix (issue #21)"
	@echo "  scan-redhatci    - Scan redhatci org with --fix (issue #22)"
	@echo "  help             - Show this help message"

