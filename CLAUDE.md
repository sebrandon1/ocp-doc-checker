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
./ocp-doc-checker --url "https://docs.openshift.com/container-platform/4.14/..."

# Scan a file or directory
./ocp-doc-checker --scan /path/to/files

# Fix URLs in place
./ocp-doc-checker --fix /path/to/files
```

### Test
```bash
make test
./test.sh  # Integration tests
```

### Lint
```bash
make lint
```

## Architecture

- **`main.go`** - Core logic for URL checking and fixing
- **`pkg/`** - Reusable packages
- **`scripts/`** - Helper scripts for various operations
- **`docs/`** - Documentation
- **`action.yml`** - GitHub Action definition

## Features

- Single URL checking
- File/directory scanning for OCP URLs
- Automatic URL fixing to latest version
- JSON output for automation
- GitHub Action for CI/CD integration
- Anchor validation (verifies URL fragments exist)

## Requirements

- Go 1.21+
- Internet access to check OCP documentation

## Code Style

- Follow standard Go conventions
- Use `go fmt` before committing
- Run `golangci-lint` for linting
