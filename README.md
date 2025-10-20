# ocp-doc-checker

A tool to check if OpenShift Container Platform (OCP) documentation URLs are outdated and suggest newer versions.

[![Tests](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test.yml)
[![Nightly Tests](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/nightly.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/nightly.yml)
[![Test Batch Mode](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test-batch-mode.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test-batch-mode.yml)
[![Test Fix Mode](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test-fix-mode.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/test-fix-mode.yml)
[![Lint Markdown](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/lint-markdown.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/lint-markdown.yml)
[![Release Binaries](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/release-binaries.yaml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/release-binaries.yaml)
[![Publish Container Image](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/publish-container.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/publish-container.yml)
[![Update Major Tag](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/update-major-tag.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/update-major-tag.yml)
[![Example](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/example.yml/badge.svg)](https://github.com/sebrandon1/ocp-doc-checker/actions/workflows/example.yml)

## Features

- ✅ Check single OCP documentation URLs
- 📁 Scan files and directories for OCP URLs
- 🔧 Automatically fix outdated URLs in files
- 📊 JSON output for automation
- 🎯 GitHub Action for CI/CD integration
- 🔍 Batch processing with detailed reports
- ⚓ **Anchor validation** - Verifies that URL fragments/anchors exist on the target page

## Installation

### Container Image

The easiest way to use the tool without installing anything:

```bash
# Using Docker
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest -dir /workspace

# Using Podman
podman run --rm -v $(pwd):/workspace:Z quay.io/bapalm/ocp-doc-checker:latest -dir /workspace
```

Images are available at `quay.io/bapalm/ocp-doc-checker` with the following tags:
- `latest` - Latest stable release
- `v1` - Latest v1.x release
- `v1.x.x` - Specific version (e.g., `v1.0.0`)

### Binary

Build from source:

```bash
git clone https://github.com/sebrandon1/ocp-doc-checker.git
cd ocp-doc-checker
make build
```

The binary will be created as `ocp-doc-checker` in the current directory.

### GitHub Action

Add to your workflow (see [GitHub Action Usage](#github-action-usage) below).

## CLI Usage

### Basic Commands

#### Check a single URL

```bash
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
```

#### Scan a directory for OCP URLs

```bash
./ocp-doc-checker -dir ./docs
```

#### Scan a single file

```bash
./ocp-doc-checker -dir ./README.md
```

#### Automatically fix outdated URLs

```bash
./ocp-doc-checker -dir ./docs -fix
```

This will:
1. Find all OCP documentation URLs in the directory
2. Check which ones are outdated
3. Automatically update them to the latest version in place

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-url` | Single OCP documentation URL to check | - |
| `-dir` | Directory or file to scan for OCP URLs | - |
| `-fix` | Automatically fix outdated URLs in files (requires `-dir`) | `false` |
| `-json` | Output results in JSON format | `false` |
| `-verbose` | Enable verbose output | `false` |
| `-all-available` | Show all available newer versions (default: latest only) | `false` |
| `-version` | Print version information | - |

### Examples

#### Show all available newer versions

```bash
./ocp-doc-checker -url "https://docs.redhat.com/..." -all-available
```

#### Get JSON output for automation

```bash
./ocp-doc-checker -url "https://docs.redhat.com/..." -json
```

#### Scan and fix with verbose output

```bash
./ocp-doc-checker -dir ./docs -fix -verbose
```

### Container Usage Examples

All CLI examples above can be run using the container image by mounting your workspace:

#### Check a single URL

```bash
docker run --rm quay.io/bapalm/ocp-doc-checker:latest \
  -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
```

#### Scan current directory

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace
```

#### Scan specific files with JSON output

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace/docs -json
```

#### Fix outdated URLs automatically

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace/docs -fix
```

#### Using Podman (with SELinux)

```bash
podman run --rm -v $(pwd):/workspace:Z quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace -fix -verbose
```

**Note:** When using the container, paths must be relative to `/workspace` since that's where your local directory is mounted.

### Exit Codes

- `0`: All URLs are up-to-date, or `-fix` was used successfully
- `1`: Outdated URLs found (when not using `-fix`), or error occurred

## GitHub Action Usage

### Basic Example - Check Single URL

```yaml
name: Check Documentation
on: [pull_request]

jobs:
  check-docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Check OCP Documentation URL
        uses: sebrandon1/ocp-doc-checker@v1
        with:
          url: 'https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full'
          fail-on-outdated: true
```

### Scan Files and Directories

```yaml
- name: Scan documentation files
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/ README.md CONTRIBUTING.md'
    fail-on-outdated: true
```

### Automatically Fix Outdated URLs

```yaml
- name: Fix outdated documentation URLs
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/ README.md'
    fix: true
```

In conjunction with a PR creation action, this can create a powerful tool to automatically fix your outdated URLs.

This will automatically update any outdated OCP documentation URLs in the specified files to their latest versions. When `fix` is enabled, the action won't fail even if outdated URLs were found (since they've been fixed).

### Fix All URLs in Repository

```yaml
- name: Fix all outdated URLs in repository
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    fix: true
```

If you don't specify `paths`, the action defaults to scanning the current directory (`.`), which will recursively scan all supported files in your repository.

### Non-Blocking Check

```yaml
- name: Check documentation (warning only)
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/'
    fail-on-outdated: false
```

### With All Available Versions

```yaml
- name: Check with all versions
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    url: 'https://docs.redhat.com/...'
    all-available: true
    verbose: true
```

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `url` | Single OCP documentation URL to check | No | - |
| `paths` | Space-separated list of files/directories to scan | No | `.` (current dir) |
| `fail-on-outdated` | Fail the action if outdated URLs are found | No | `true` |
| `all-available` | Show all available newer versions | No | `false` |
| `verbose` | Enable verbose output | No | `false` |
| `fix` | Automatically fix outdated URLs in files (only works with `paths`) | No | `false` |

**Note:** `url` and `paths` are mutually exclusive. If neither is specified, `paths` defaults to the current directory (`.`). The `fix` option can only be used with `paths` mode.

### Action Outputs

#### Single URL Mode

| Output | Description |
|--------|-------------|
| `is-outdated` | Whether the URL is outdated (`true`/`false`) |
| `latest-version` | The latest version where the doc exists (e.g., `4.19`) |
| `newer-versions` | JSON array of newer versions |

#### Batch Mode (paths)

| Output | Description |
|--------|-------------|
| `is-outdated` | Whether any URLs are outdated (`true`/`false`) |
| `outdated-count` | Number of outdated URLs found |
| `uptodate-count` | Number of up-to-date URLs found |
| `total-count` | Total number of URLs checked |

### Using Outputs

```yaml
- name: Check documentation
  id: doc-check
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    url: 'https://docs.redhat.com/...'
    fail-on-outdated: false

- name: Use outputs
  run: |
    echo "Is outdated: ${{ steps.doc-check.outputs.is-outdated }}"
    echo "Latest version: ${{ steps.doc-check.outputs.latest-version }}"
```

## JSON Output Format

### Single URL

```json
{
  "original_url": "https://docs.redhat.com/.../4.17/...",
  "original_version": "4.17",
  "latest_version": "4.19",
  "is_outdated": true,
  "newer_versions": [
    {
      "version": "4.18",
      "url": "https://docs.redhat.com/.../4.18/..."
    },
    {
      "version": "4.19",
      "url": "https://docs.redhat.com/.../4.19/..."
    }
  ]
}
```

## Development

### Build

```bash
make build
```

### Run Tests

```bash
# Run all tests (unit + integration)
make test

# Run only Go unit tests
make test-unit

# Run only integration tests
make test-integration

# Run all CI tests
make test-ci
```

### Format and Lint

```bash
# Format code
make fmt

# Run linters
make lint
```

### Install Locally

```bash
make install
```

This installs the binary to `$GOPATH/bin`.

### Build Container Image

```bash
make build-image
```

Builds the Docker image locally and tags it as `quay.io/ocp-doc-checker:dev` and `quay.io/ocp-doc-checker:latest`. You can customize the image name and tag:

```bash
make build-image IMAGE_REGISTRY=docker.io IMAGE_NAME=myorg/ocp-doc-checker VERSION=v1.0.0
```

## How It Works

1. **URL Parsing**: Extracts OCP version and document structure from Red Hat documentation URLs
2. **Version Discovery**: Checks newer OCP versions to see if the same document exists
3. **URL Validation**: Verifies that suggested URLs are accessible (HTTP HEAD/GET requests)
4. **Anchor Validation**: When a URL contains a fragment (`#anchor`), the tool:
   - Fetches and parses the HTML page
   - Searches for the anchor ID in the page content
   - Reports when a page exists but the anchor is missing
   - Prevents false positives when content is reorganized between versions
5. **Smart Recommendations**: Only suggests versions where both the page AND anchor (if present) exist

### Anchor Validation Details

The tool validates anchors by checking for:
- HTML `id` attributes (e.g., `<h2 id="section-name">`)
- Legacy `<a name="anchor">` tags
- Red Hat docs style anchors (e.g., `#installing-operator_procedure-name`)

This prevents suggesting URLs where the documentation page exists but the specific section has moved or been renamed.

**Example:** If you have a URL pointing to 4.17 with anchor `#installing-sr-iov-operator_installing-sriov-operator`, and in 4.19 the SR-IOV content moved from the networking guide to the hardware_networks guide, the tool will detect that the anchor doesn't exist in the 4.19 networking guide and won't suggest it as an upgrade path.

**Performance Note:** Anchor validation requires fetching and parsing full HTML pages, which is slower than simple HEAD requests. For URLs without anchors, the tool uses fast HEAD requests. URLs with anchors will take longer to validate (typically 1-3 seconds per URL).

## Supported URL Formats

The tool supports Red Hat OpenShift Container Platform documentation URLs in the following formats:

- `https://docs.redhat.com/en/documentation/openshift_container_platform/{version}/html-single/{document}/index#{anchor}`
- `https://docs.redhat.com/en/documentation/openshift_container_platform/{version}/html/{document}/{page}#{anchor}`

Examples:
- Single-page HTML: `.../4.17/html-single/disconnected_environments/index#anchor`
- Multi-page HTML: `.../4.17/html/telco_ref_design_specs/telco-hub-ref-design-specs#anchor`

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

Brandon Palm

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
