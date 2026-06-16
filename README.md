# ocp-doc-checker

A tool to check if OpenShift Container Platform (OCP) documentation URLs are outdated and suggest newer versions.

[![Tests](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test.yml)
[![Nightly Tests](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/nightly.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/nightly.yml)
[![Release Binaries](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/release-binaries.yaml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/release-binaries.yaml)
[![Publish Container Image](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/publish-container.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/publish-container.yml)

## Key Features

- Check single OCP documentation URLs or scan entire directories
- Automatically fix outdated URLs in place
- Anchor validation — verifies URL fragments exist on the target page
- JSON output for automation
- GitHub Action for CI/CD integration
- Container image for zero-install usage

## Quick Start

```bash
# Build from source
git clone https://github.com/sebrandon1/ocp-doc-checker.git
cd ocp-doc-checker && make build

# Check a URL
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"

# Scan and fix a directory
./ocp-doc-checker -dir ./docs -fix

# Or use the container image
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest -dir /workspace
```

## Guides

| Guide | Description |
|-------|-------------|
| [CLI Usage](docs/cli-usage.md) | Flags, container usage, exit codes, JSON output |
| [GitHub Action](docs/github-action.md) | Action inputs, outputs, and workflow examples |
| [How It Works](docs/how-it-works.md) | URL parsing, anchor validation, supported formats |
| [Batch Mode](docs/BATCH_MODE.md) | Batch processing with detailed reports |
| [Slack Integration](docs/SLACK_INTEGRATION.md) | Approval workflows via Slack |

## Development

```bash
make build            # Build binary
make test             # Run all tests
make test-unit        # Run unit tests only
make lint             # Run linters
make fmt              # Format code
make install          # Install to $GOPATH/bin
make build-image      # Build container image
```

## License

MIT License — see [LICENSE](LICENSE) file for details.
