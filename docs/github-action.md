# GitHub Action Usage

## Basic Example — Check Single URL

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

## Scan Files and Directories

```yaml
- name: Scan documentation files
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/ README.md CONTRIBUTING.md'
    fail-on-outdated: true
```

## Automatically Fix Outdated URLs

```yaml
- name: Fix outdated documentation URLs
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/ README.md'
    fix: true
```

In conjunction with a PR creation action, this can create a powerful tool to automatically fix your outdated URLs.

This will automatically update any outdated OCP documentation URLs in the specified files to their latest versions. When `fix` is enabled, the action won't fail even if outdated URLs were found (since they've been fixed).

## Fix All URLs in Repository

```yaml
- name: Fix all outdated URLs in repository
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    fix: true
```

If you don't specify `paths`, the action defaults to scanning the current directory (`.`), which will recursively scan all supported files in your repository.

## Non-Blocking Check

```yaml
- name: Check documentation (warning only)
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/'
    fail-on-outdated: false
```

## With All Available Versions

```yaml
- name: Check with all versions
  uses: sebrandon1/ocp-doc-checker@v1
  with:
    url: 'https://docs.redhat.com/...'
    all-available: true
    verbose: true
```

## Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `url` | Single OCP documentation URL to check | No | - |
| `paths` | Space-separated list of files/directories to scan | No | `.` (current dir) |
| `fail-on-outdated` | Fail the action if outdated URLs are found | No | `true` |
| `all-available` | Show all available newer versions | No | `false` |
| `verbose` | Enable verbose output | No | `false` |
| `fix` | Automatically fix outdated URLs in files (only works with `paths`) | No | `false` |

**Note:** `url` and `paths` are mutually exclusive. If neither is specified, `paths` defaults to the current directory (`.`). The `fix` option can only be used with `paths` mode.

## Action Outputs

### Single URL Mode

| Output | Description |
|--------|-------------|
| `is-outdated` | Whether the URL is outdated (`true`/`false`) |
| `latest-version` | The latest version where the doc exists (e.g., `4.19`) |
| `newer-versions` | JSON array of newer versions |

### Batch Mode (paths)

| Output | Description |
|--------|-------------|
| `is-outdated` | Whether any URLs are outdated (`true`/`false`) |
| `outdated-count` | Number of outdated URLs found |
| `uptodate-count` | Number of up-to-date URLs found |
| `total-count` | Total number of URLs checked |

## Using Outputs

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
