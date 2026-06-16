# CLI Usage

## Basic Commands

### Check a single URL

```bash
./ocp-doc-checker -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
```

### Scan a directory for OCP URLs

```bash
./ocp-doc-checker -dir ./docs
```

### Scan a single file

```bash
./ocp-doc-checker -dir ./README.md
```

### Automatically fix outdated URLs

```bash
./ocp-doc-checker -dir ./docs -fix
```

This will:
1. Find all OCP documentation URLs in the directory
2. Check which ones are outdated
3. Automatically update them to the latest version in place

## CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-url` | Single OCP documentation URL to check | - |
| `-dir` | Directory or file to scan for OCP URLs | - |
| `-fix` | Automatically fix outdated URLs in files (requires `-dir`) | `false` |
| `-json` | Output results in JSON format | `false` |
| `-verbose` | Enable verbose output | `false` |
| `-all-available` | Show all available newer versions (default: latest only) | `false` |
| `-version` | Print version information | - |

## Examples

### Show all available newer versions

```bash
./ocp-doc-checker -url "https://docs.redhat.com/..." -all-available
```

### Get JSON output for automation

```bash
./ocp-doc-checker -url "https://docs.redhat.com/..." -json
```

### Scan and fix with verbose output

```bash
./ocp-doc-checker -dir ./docs -fix -verbose
```

## Container Usage

All CLI examples above can be run using the container image by mounting your workspace:

### Check a single URL

```bash
docker run --rm quay.io/bapalm/ocp-doc-checker:latest \
  -url "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full"
```

### Scan current directory

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace
```

### Scan specific files with JSON output

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace/docs -json
```

### Fix outdated URLs automatically

```bash
docker run --rm -v $(pwd):/workspace quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace/docs -fix
```

### Using Podman (with SELinux)

```bash
podman run --rm -v $(pwd):/workspace:Z quay.io/bapalm/ocp-doc-checker:latest \
  -dir /workspace -fix -verbose
```

**Note:** When using the container, paths must be relative to `/workspace` since that's where your local directory is mounted.

## Exit Codes

- `0`: All URLs are up-to-date, or `-fix` was used successfully
- `1`: Outdated URLs found (when not using `-fix`), or error occurred

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
