# End-to-End Tests

This directory contains end-to-end tests for metabigor commands.

## Running Tests

### All E2E Tests

```bash
make e2e
```

This will:
1. Build the binary (`make build`)
2. Run all end-to-end tests in `test/run-e2e.sh`

### Run Tests Directly

```bash
cd test
./run-e2e.sh
```

## Test Coverage

The E2E tests verify:

### Commands
- ✅ `version` - Version information
- ✅ `update` - ASN database update
- ✅ `net` - Network discovery (ASN, IP, domain, org)
- ✅ `cert` - Certificate transparency search
- ✅ `ip` - IP enrichment via Shodan InternetDB
- ✅ `github` - GitHub code search via grep.app
- ✅ `ipc` - IP clustering by ASN
- ✅ `related` - Related domain discovery
- ✅ `cdn` - CDN detection

### Flags
- ✅ `--help` - Help text for all commands
- ✅ `-q, --silent` - Silent mode (errors only, no progress messages)
- ✅ `--debug` - Debug logging with `[debug]` prefix
- ✅ `--json` - JSON output format
- ✅ `--no-color` - Disable colored output
- ✅ `-o, --output` - Write to file
- ✅ `-i, --input` - Single input value
- ✅ `-I, --inputFile` - Read input from file
- ✅ `-c, --concurrency` - Concurrent workers
- ✅ Command-specific flags (--clean, --wildcard, --detail, --page, etc.)

### Input Methods
- ✅ Stdin (pipe)
- ✅ `-i, --input` flag
- ✅ `-I` file flag

### Output Modes
- ✅ Text output (default)
- ✅ JSON output (`--json`)
- ✅ File output (`-o`)

## Test Strategy

The E2E tests use a pragmatic approach:

1. **Help Text Validation**: Verify all commands have proper help text
2. **Flag Existence**: Check that documented flags exist
3. **Input Processing**: Test different input methods work
4. **Output Formatting**: Verify verbose/debug prefixes and JSON output
5. **Integration Points**: Test with safe/free inputs to avoid rate limits

**Note**: Some tests use `timeout` or skip actual API calls to:
- Avoid rate limiting during CI/CD
- Keep tests fast and reliable
- Prevent hammering free services

## Adding New Tests

To add a test for a new command or feature:

1. Open `run-e2e.sh`
2. Add tests using helper functions:

```bash
# Check command output contains pattern
run_test "test name" \
    "command to run" \
    "expected pattern in output"

# Check complex conditions
check_output "test name" \
    "command to run" \
    'grep -q "pattern" && grep -q "another"'
```

3. Run tests: `make e2e`

## CI/CD Integration

These tests are designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run E2E tests
  run: make e2e
```

The tests are:
- Fast (< 30 seconds typically)
- Non-destructive (no external mutations)
- Rate-limit friendly (use timeouts, skip heavy queries)
- Self-contained (no external dependencies beyond binary)

## Troubleshooting

### Binary not found
```bash
Error: Binary not found at ../bin/metabigor
```
**Solution**: Run `make build` first

### Tests timing out
Some tests use `timeout` to prevent hanging on network calls. If your network is slow, you may need to increase timeout values in `run-e2e.sh`.

### Tests failing on network commands
Network-dependent tests (`net`, `cert`, `github`, etc.) may fail if:
- No internet connection
- External services are down
- Rate limiting is active

These are expected failures in offline/restricted environments. The test script tries to be resilient with `|| true` and timeouts.
