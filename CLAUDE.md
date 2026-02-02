# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A tool to check if OpenShift Container Platform (OCP) documentation URLs are outdated and suggest newer versions. Supports single URL checks, batch file scanning, and automatic URL fixing.

## Common Commands

### Build
```bash
make build
```

### Run
```bash
# Check single URL
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/..."

# Scan a file or directory
./ocp-doc-checker -dir /path/to/files

# Fix URLs in place
./ocp-doc-checker -dir /path/to/files -fix
```

### Test
```bash
make test              # Run all tests (unit + integration)
make test-unit         # Run only Go unit tests
make test-integration  # Run only integration tests
make test-ci           # Run all CI test scripts
```

### Lint
```bash
make lint
make fmt   # Format code
```

### Container
```bash
make build-image       # Build Docker image
```

## Architecture

- **`main.go`** - Core logic for URL checking and fixing
- **`pkg/checker/`** - URL checking and validation logic
- **`pkg/parser/`** - URL parsing utilities
- **`scripts/`** - Helper scripts (org scanning, Slack integration, tests)
- **`docs/`** - Documentation (batch mode, Slack integration)
- **`action.yml`** - GitHub Action definition
- **`Dockerfile`** - Container image build

## Features

- Single URL checking
- File/directory scanning for OCP URLs
- Automatic URL fixing to latest version
- JSON output for automation
- GitHub Action for CI/CD integration
- Anchor validation (verifies URL fragments exist)
- Container image support (Docker/Podman)
- Slack integration for approval workflows

## Requirements

- Go 1.25+
- Internet access to check OCP documentation

## Code Style

- Follow standard Go conventions
- Use `go fmt` before committing
- Run `golangci-lint` for linting
