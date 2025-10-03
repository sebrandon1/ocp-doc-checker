# Testing Guide

This document describes the comprehensive testing approach for the OCP Documentation Checker.

## Testing Philosophy

The project uses an **expected/actual assertion** style where we:
1. Pass known outdated URLs and expect specific behavior (failure + suggestions)
2. Pass current URLs and expect success
3. Verify the exact replacement URLs match expected values
4. Test edge cases and error handling

## Test Levels

### 1. Unit Tests (`go test`)

Located in `pkg/*/parser_test.go`, these test individual components:

```bash
go test ./...
```

**Tests:**
- URL parsing for various formats
- Version extraction and comparison
- URL reconstruction with different versions
- Error handling for malformed URLs

### 2. Integration Tests (`./test.sh`)

End-to-end testing with real URLs and assertions:

```bash
./test.sh
```

**Test Cases:**

#### Test 1: Outdated URL Detection
- **Input:** 4.17 URL
- **Expected:** Exit code 1, `is_outdated=true`, newer versions listed
- **Validates:** Tool correctly identifies outdated documentation

#### Test 2: Current URL Validation  
- **Input:** 4.19 URL (current latest)
- **Expected:** Exit code 0, `is_outdated=false`, `latest_version=4.19`
- **Validates:** Tool correctly identifies up-to-date documentation

#### Test 3: URL Replacement
- **Input:** 4.17 URL with specific anchor
- **Expected:** Exact 4.19 URL with same anchor
- **Validates:** Correct URL construction for newer versions

#### Test 4: Invalid URL Handling
- **Input:** Non-OCP documentation URL
- **Expected:** Graceful error message
- **Validates:** Error handling and user-friendly messages

#### Test 5: Anchor Preservation
- **Input:** URL with anchor/fragment
- **Expected:** All suggested URLs preserve the anchor
- **Validates:** Deep linking is maintained

#### Test 6: Version Comparison
- **Input:** 4.15 URL
- **Expected:** Multiple newer versions detected, sorted correctly
- **Validates:** Version comparison logic

### 3. GitHub Actions CI Tests (`.github/workflows/test.yml`)

Automated tests that run on every PR:

#### Job: `test-go`
- Runs Go unit tests
- Builds the binary

#### Job: `test-cli`
- All 6 integration tests from `test.sh`
- Additional HTTP verification of suggested URLs
- Test summaries in GitHub UI

#### Job: `test-action`
- Tests the GitHub Action itself
- Validates action inputs/outputs
- Verifies fail-on-outdated behavior

## Writing New Tests

### Adding a Unit Test

Add to `pkg/*/parser_test.go`:

```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "description",
            input: "test input",
            want:  "expected output",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Adding an Integration Test

Add to `test.sh`:

```bash
test_new_feature() {
    echo "Testing new feature..."
    
    local url="https://docs.redhat.com/..."
    local expected="expected_value"
    
    set +e
    OUTPUT=$(./ocp-doc-checker -url "$url" -json 2>&1)
    EXIT_CODE=$?
    set -e
    
    echo "$OUTPUT" | jq .
    
    # Assertions
    if [ $EXIT_CODE -ne 0 ]; then
        echo "Expected exit code 0, got $EXIT_CODE"
        return 1
    fi
    
    ACTUAL=$(echo "$OUTPUT" | jq -r '.field')
    if [ "$ACTUAL" != "$expected" ]; then
        echo "Expected $expected, got $ACTUAL"
        return 1
    fi
    
    return 0
}
```

Then add to the main function:
```bash
run_test "Test N: New Feature" test_new_feature || true
```

### Adding a GitHub Action Test

Add to `.github/workflows/test.yml`:

```yaml
- name: Test New Feature
  id: test_new
  run: |
    echo "Testing new feature..."
    ./ocp-doc-checker -url "URL" -json > output.json
    
    EXPECTED="value"
    ACTUAL=$(jq -r '.field' output.json)
    
    if [ "$ACTUAL" != "$EXPECTED" ]; then
      echo "❌ FAIL: Expected $EXPECTED, got $ACTUAL"
      exit 1
    fi
    
    echo "✅ PASS: Test passed"
```

## Running Tests Locally

### Quick Test
```bash
make test
```

### Unit Tests Only
```bash
make test-unit
```

### Integration Tests Only
```bash
make test-integration
```

### Specific Test
```bash
# Run a specific Go test
go test -v -run TestParseOCPDocURL ./pkg/parser

# Run integration tests with verbose output
./test.sh
```

## Test Coverage

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## Continuous Integration

All tests run automatically on:
- ✅ Pull requests
- ✅ Pushes to main branch
- ✅ Manual workflow dispatch

Test results appear in:
- GitHub Actions summary
- PR checks
- Workflow logs

## Expected Test Times

- Unit tests: < 1 second
- Integration tests: ~15-30 seconds (makes HTTP requests)
- Full CI pipeline: ~2-3 minutes

## Debugging Failed Tests

### Local Debugging

```bash
# Run with verbose output
./test.sh

# Check specific URL manually
./ocp-doc-checker -url "URL" -verbose

# Get JSON for inspection
./ocp-doc-checker -url "URL" -json | jq .
```

### CI Debugging

1. Check the Actions tab in GitHub
2. Click on the failed workflow run
3. Expand the failed test step
4. Review the assertion output
5. Check "Test Summary" section

## Common Test Failures

### "Expected exit code 1, got 0"
- The URL might now be outdated
- The document might have been removed from newer versions
- Update the test case or URL

### "Expected exit code 0, got 1"
- A newer version was released
- Update `knownVersions` in `checker.go`
- Update test expectations

### "URL mismatch"
- Document structure changed in newer version
- Anchor/section was renamed
- Update expected URL or investigate document changes

## Best Practices

1. **Always add tests** for new features
2. **Test both success and failure** cases
3. **Use real URLs** when possible (integration tests)
4. **Mock only when necessary** (unit tests)
5. **Keep tests fast** - unit tests should be < 1s
6. **Make assertions specific** - don't just check for non-nil
7. **Test edge cases** - empty strings, invalid input, etc.
8. **Document test intent** - use descriptive names and comments

