# Batch Mode Documentation

The OCP Documentation Checker GitHub Action now supports **Batch Mode** - the ability to scan entire directories and multiple files for OCP documentation URLs and check them all automatically.

## Quick Start

```yaml
- uses: sebrandon1/ocp-doc-checker@v1
  with:
    paths: 'docs/ README.md'
```

## Features

### 🔍 Automatic URL Detection
- Scans files for OCP documentation URLs using regex pattern
- Supports multiple file formats: `.md`, `.markdown`, `.txt`, `.adoc`
- Recursively searches directories
- Deduplicates URLs (each unique URL checked once)

### 📊 Comprehensive Reporting
- **Summary Table**: Shows status of each URL
- **Statistics**: Total, up-to-date, and outdated counts
- **Recommendations**: Provides exact old → new URL replacements
- **GitHub Actions Integration**: Rich summary in Actions UI

### 🎯 Flexible Configuration
- Single file: `paths: 'README.md'`
- Multiple files: `paths: 'README.md CONTRIBUTING.md'`
- Directory: `paths: 'docs/'`
- Mixed: `paths: 'docs/ README.md CONTRIBUTING.md'`

## Example Workflows

### Basic Documentation Linting

```yaml
name: Lint Docs

on:
  pull_request:
    paths:
      - '**.md'

jobs:
  check-urls:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: sebrandon1/ocp-doc-checker@v1
        with:
          paths: '.'
          fail-on-outdated: true
```

### Non-Blocking Check with PR Comment

```yaml
name: Check Docs

on: [pull_request]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: sebrandon1/ocp-doc-checker@v1
        id: check
        continue-on-error: true
        with:
          paths: 'docs/ README.md'
          fail-on-outdated: false
      
      - uses: actions/github-script@v7
        if: steps.check.outputs.is-outdated == 'true'
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `⚠️ Found ${context.payload.steps.check.outputs['outdated-count']} outdated URLs`
            })
```

### Specific Documentation Directories

```yaml
name: Check Product Docs

on:
  push:
    branches:
      - main

jobs:
  check-docs:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        docs:
          - path: 'docs/installation/'
            name: 'Installation Docs'
          - path: 'docs/networking/'
            name: 'Networking Docs'
          - path: 'docs/security/'
            name: 'Security Docs'
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Check ${{ matrix.docs.name }}
        uses: sebrandon1/ocp-doc-checker@v1
        with:
          paths: ${{ matrix.docs.path }}
          fail-on-outdated: true
```

## Output Summary Example

When outdated URLs are found, you'll see a summary like this in the GitHub Actions UI:

```
📋 OCP Documentation URL Check

Scanned paths: docs/ README.md

Found 8 unique URL(s)

| Status | Current Version | Latest Version | URL |
|--------|-----------------|----------------|-----|
| ✅ | 4.19 | 4.19 | Link |
| ✅ | 4.19 | 4.19 | Link |
| ⚠️ | 4.17 | 4.19 | Link |
| ✅ | 4.18 | 4.18 | Link |
| ⚠️ | 4.16 | 4.19 | Link |
| ⚠️ | 4.15 | 4.19 | Link |
| ✅ | 4.19 | 4.19 | Link |
| ✅ | 4.19 | 4.19 | Link |

Summary: 5 up-to-date, 3 outdated

🔧 Recommended Updates

Replace the following URLs in your documentation:

- 🔄 Update:
  - Old: `https://docs.redhat.com/.../4.17/...`
  - New: `https://docs.redhat.com/.../4.19/...`

- 🔄 Update:
  - Old: `https://docs.redhat.com/.../4.16/...`
  - New: `https://docs.redhat.com/.../4.19/...`

- 🔄 Update:
  - Old: `https://docs.redhat.com/.../4.15/...`
  - New: `https://docs.redhat.com/.../4.19/...`
```

## Outputs

When using batch mode, the following outputs are available:

| Output | Description | Example |
|--------|-------------|---------|
| `is-outdated` | Any URLs outdated? | `true` |
| `outdated-count` | Number of outdated URLs | `3` |
| `uptodate-count` | Number of up-to-date URLs | `5` |
| `total-count` | Total URLs checked | `8` |

## Best Practices

### 1. Use on Pull Requests
Check documentation changes before merging:
```yaml
on:
  pull_request:
    paths:
      - 'docs/**'
      - '**.md'
```

### 2. Set Appropriate Paths
Be specific to avoid scanning unnecessary files:
```yaml
paths: 'docs/ README.md CONTRIBUTING.md'  # ✅ Good
paths: '.'  # ⚠️ May be too broad
```

### 3. Use fail-on-outdated Wisely
- **CI/CD blocking**: `fail-on-outdated: true`
- **Informational only**: `fail-on-outdated: false` with `continue-on-error: true`

### 4. Combine with PR Comments
Make it easy for developers to fix issues:
```yaml
- uses: sebrandon1/ocp-doc-checker@v1
  id: check
  continue-on-error: true
  with:
    paths: 'docs/'
    fail-on-outdated: false

- if: steps.check.outputs.is-outdated == 'true'
  uses: actions/github-script@v7
  # ... post helpful comment
```

## Comparison: Single URL vs Batch Mode

| Feature | Single URL Mode | Batch Mode |
|---------|----------------|------------|
| **Input** | `url: 'https://...'` | `paths: 'docs/ README.md'` |
| **Use Case** | Check specific URL | Lint documentation |
| **Output** | Single URL result | Multiple URLs summary |
| **Performance** | Fast (1 check) | Depends on URL count |
| **Recommendations** | Single replacement | All replacements listed |
| **Best For** | Targeted checks | Repository-wide scanning |

## FAQ

### Q: How many files/URLs can it handle?
A: There's no hard limit, but for best performance, keep it under 100 URLs. For larger repos, consider splitting by directory.

### Q: What if a URL is not found in newer versions?
A: It will be marked as outdated but no newer version will be shown in recommendations.

### Q: Can I exclude certain directories?
A: Not directly in the action. Use specific paths or pre-filter with `find` commands.

### Q: Does it modify my files?
A: No, it only reads files and provides recommendations. You make the changes.

### Q: Can I use it with other documentation formats?
A: Yes! Any plain text file can be scanned when specified directly. Directories auto-filter for common formats.

## Troubleshooting

### No URLs Found
```
✅ No OCP Documentation URLs Found
```
**Solution**: Check that your files contain OCP doc URLs matching the pattern:
```
https://docs.redhat.com/en/documentation/openshift_container_platform/X.XX/...
```

### Path Not Found Warning
```
⚠️ Path not found: docs/missing/
```
**Solution**: Verify the path exists and is relative to repository root.

### Many False Positives
If URLs are incorrectly detected:
- Make sure URLs are properly formatted
- Check for broken/incomplete URLs in your docs
- URLs must match the exact OCP documentation pattern

## Advanced Usage

### Running Locally with Grep
You can use the CLI tool with a similar pattern:

```bash
# Find and check all URLs
grep -roh 'https://docs\.redhat\.com/en/documentation/openshift_container_platform/[0-9]\+\.[0-9]\+/[^[:space:])\]"]*' . \
  --include="*.md" | sort -u | \
  while read url; do
    ocp-doc-checker -url "$url"
  done
```

### Integration with Other Tools

Combine with [danger.js](https://danger.systems/) or other PR automation tools for richer feedback.

