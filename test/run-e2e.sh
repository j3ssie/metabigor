#!/usr/bin/env bash
# Note: Not using 'set -e' so tests continue after failures

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Determine project root and binary location
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TEST_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/bin/metabigor"
PASSED=0
FAILED=0

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY${NC}"
    echo "Run 'make build' first"
    exit 1
fi

echo "Running end-to-end tests for metabigor..."
echo "=========================================="
echo ""

# Helper function to run test
run_test() {
    local test_name=$1
    local command=$2
    local expected_pattern=$3
    local should_match=${4:-true}  # Default: expect pattern to match

    echo -n "Testing: $test_name... "

    # Run command and capture output
    output=$(eval "$command" 2>&1 || true)

    if [ "$should_match" = true ]; then
        if echo "$output" | grep -qF -- "$expected_pattern"; then
            echo -e "${GREEN}PASS${NC}"
            ((PASSED++))
            return 0
        else
            echo -e "${RED}FAIL${NC}"
            echo "  Expected pattern: $expected_pattern"
            echo "  Got: $(echo "$output" | head -20)"
            ((FAILED++))
            return 1
        fi
    else
        if echo "$output" | grep -qF -- "$expected_pattern"; then
            echo -e "${RED}FAIL${NC}"
            echo "  Expected pattern NOT to match: $expected_pattern"
            echo "  Got: $(echo "$output" | head -20)"
            ((FAILED++))
            return 1
        else
            echo -e "${GREEN}PASS${NC}"
            ((PASSED++))
            return 0
        fi
    fi
}

echo "=== Basic Command Tests ==="

# Version command
run_test "version command" \
    "$BINARY version" \
    "Version:"

run_test "version --help" \
    "$BINARY version --help" \
    "Display the version"

echo ""
echo "=== Update Command Tests ==="

run_test "update --help" \
    "$BINARY update --help" \
    "Download or update"

echo ""
echo "=== Network Discovery Tests ==="

run_test "net --help" \
    "$BINARY net --help" \
    "Discover CIDRs"

run_test "net with ASN input (stdin)" \
    "echo 'AS13335' | $BINARY net --debug 2>&1" \
    "AS13335"

run_test "net with --input flag" \
    "$BINARY net --input AS13335 --debug 2>&1" \
    "AS13335"

run_test "net with --asn flag" \
    "echo '13335' | $BINARY net --asn --debug 2>&1" \
    "13335"

echo ""
echo "=== Certificate Search Tests ==="

run_test "cert --help" \
    "$BINARY cert --help" \
    "Certificate Transparency"

run_test "cert with domain" \
    "echo 'example.com' | timeout 10s $BINARY cert --debug 2>&1 || true" \
    "example.com"

run_test "cert --clean flag exists" \
    "$BINARY cert --help" \
    "--clean"

run_test "cert --wildcard flag exists" \
    "$BINARY cert --help" \
    "--wildcard"

echo ""
echo "=== IP Enrichment Tests ==="

run_test "ip --help" \
    "$BINARY ip --help" \
    "IP enrichment"

run_test "ip --flat flag exists" \
    "$BINARY ip --help" \
    "--flat"

run_test "ip --csv flag exists" \
    "$BINARY ip --help" \
    "--csv"

echo ""
echo "=== GitHub Search Tests ==="

run_test "github --help" \
    "$BINARY github --help" \
    "grep.app"

run_test "github --detail flag exists" \
    "$BINARY github --help" \
    "--detail"

run_test "github --page flag exists" \
    "$BINARY github --help" \
    "--page"

run_test "github shows verbose logs by default" \
    "echo 'test' | timeout 5s $BINARY github 2>&1 || true" \
    "[verbose]"

echo ""
echo "=== IP Clustering Tests ==="

run_test "ipc --help" \
    "$BINARY ipc --help" \
    "Group a list of IPs"

run_test "ipc with IP input" \
    "echo '1.1.1.1' | $BINARY ipc --debug 2>&1 || true" \
    "AS13335"

echo ""
echo "=== Related Domain Tests ==="

run_test "related --help" \
    "$BINARY related --help" \
    "Find domains related"

run_test "related -s flag exists" \
    "$BINARY related --help" \
    "-s, --source"

echo ""
echo "=== CDN Detection Tests ==="

run_test "cdn --help" \
    "$BINARY cdn --help" \
    "Check if IP addresses"

echo ""
echo "=== Global Flag Tests ==="

run_test "silent flag (-q) hides verbose" \
    "echo 'AS13335' | $BINARY net -q 2>&1 | grep -q '\\[verbose\\]' && echo 'FOUND' || echo 'NOT_FOUND'" \
    "NOT_FOUND"

run_test "silent flag (-q) hides info" \
    "echo 'AS13335' | $BINARY net -q 2>&1 | grep -q '\\[info\\]' && echo 'FOUND' || echo 'NOT_FOUND'" \
    "NOT_FOUND"

run_test "default mode shows info prefix" \
    "echo 'AS13335' | $BINARY net 2>&1 || true" \
    "[info]"

run_test "default mode shows verbose prefix" \
    "echo 'AS13335' | $BINARY net 2>&1 || true" \
    "[verbose]"

run_test "debug flag (--debug)" \
    "echo 'AS13335' | $BINARY net --debug 2>&1 || true" \
    "[debug]"

run_test "no-color flag works" \
    "$BINARY version --no-color" \
    "Version:"

run_test "json flag exists" \
    "$BINARY cert --help" \
    "--json"

run_test "input flag shorthand (-i) exists" \
    "$BINARY net --help" \
    "-i, --input"

run_test "inputFile flag (-I) exists" \
    "$BINARY net --help" \
    "-I, --inputFile"

run_test "silent flag (-q) exists" \
    "$BINARY net --help" \
    "-q, --silent"

echo ""
echo "=== Input Method Tests ==="

# Create temp input file
TEMP_INPUT=$(mktemp)
echo "AS13335" > "$TEMP_INPUT"

run_test "file input (-I)" \
    "$BINARY net -I $TEMP_INPUT --debug 2>&1" \
    "AS13335"

run_test "input flag shorthand (-i)" \
    "$BINARY net -i AS13335 --debug 2>&1" \
    "AS13335"

run_test "input flag long form (--input)" \
    "$BINARY net --input AS13335 --debug 2>&1" \
    "AS13335"

rm -f "$TEMP_INPUT"

echo ""
echo "=== Output Tests ==="

# Create temp output file
TEMP_OUTPUT=$(mktemp)

run_test "output to file (-o)" \
    "$BINARY version -o $TEMP_OUTPUT && cat $TEMP_OUTPUT" \
    "Version:"

rm -f "$TEMP_OUTPUT"

echo ""
echo "=========================================="
echo -e "Test Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
