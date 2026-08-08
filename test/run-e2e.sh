#!/usr/bin/env bash
# End-to-end checks for the metabigor CLI surface.
# Note: no 'set -e', so a failure does not stop the remaining tests.

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TEST_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/bin/metabigor"
PASSED=0
FAILED=0

if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: binary not found at $BINARY${NC}"
    echo "Run 'make build' first"
    exit 1
fi

echo "Running end-to-end tests for metabigor..."
echo "=========================================="
echo ""

# run_test <name> <command> <pattern> [should_match]
run_test() {
    local test_name=$1
    local command=$2
    local expected_pattern=$3
    local should_match=${4:-true}

    echo -n "Testing: $test_name... "
    output=$(eval "$command" 2>&1 </dev/null || true)

    if [ "$should_match" = true ]; then
        if echo "$output" | grep -qF -- "$expected_pattern"; then
            echo -e "${GREEN}PASS${NC}"; ((PASSED++)); return 0
        fi
        echo -e "${RED}FAIL${NC}"
        echo "  Expected to find: $expected_pattern"
        echo "  Got: $(echo "$output" | head -20)"
        ((FAILED++)); return 1
    else
        if echo "$output" | grep -qF -- "$expected_pattern"; then
            echo -e "${RED}FAIL${NC}"
            echo "  Expected NOT to find: $expected_pattern"
            echo "  Got: $(echo "$output" | head -20)"
            ((FAILED++)); return 1
        fi
        echo -e "${GREEN}PASS${NC}"; ((PASSED++)); return 0
    fi
}

# run_exit <name> <command> <expected exit code>
run_exit() {
    local test_name=$1
    local command=$2
    local expected=$3

    echo -n "Testing: $test_name... "
    eval "$command" >/dev/null 2>&1 </dev/null
    local code=$?

    if [ "$code" -eq "$expected" ]; then
        echo -e "${GREEN}PASS${NC}"; ((PASSED++)); return 0
    fi
    echo -e "${RED}FAIL${NC}"
    echo "  Expected exit code $expected, got $code"
    ((FAILED++)); return 1
}

echo "=== Basic Commands ==="

run_test "version command" "$BINARY version" "Version:"
run_test "--version flag" "$BINARY --version" "metabigor"
run_test "root help lists command groups" "$BINARY --help" "Recon commands:"
run_test "update --help" "$BINARY update --help" "Refresh the local"

echo ""
echo "=== Exit Codes ==="

run_exit "success exits 0" "$BINARY net AS13335 -q" 0
run_exit "missing target exits 1" "$BINARY net" 1
run_exit "invalid --format exits 1" "$BINARY net AS13335 -f yaml" 1
run_exit "unknown related source exits 1" "$BINARY related example.com -s bogus" 1
run_exit "unknown url source exits 1" "$BINARY url example.com -s bogus" 1
run_exit "invalid url --keywords regex exits 1" "$BINARY url example.com --keywords '(' " 1
run_exit "invalid url --from exits 1" "$BINARY url example.com --from lastyear" 1
run_exit "url keyed source without a key exits 1" "env -u VT_API_KEY -u VIRUSTOTAL_API_KEY $BINARY url example.com -s virustotal" 1
run_exit "conflicting net type flags exit 1" "$BINARY net 1.1.1.1 --ip --org" 1
run_exit "unknown flag exits 1" "$BINARY net --nope" 1

run_test "missing target explains how to pass one" "$BINARY cert" "no target given"

echo ""
echo "=== Log Verbosity ==="

run_test "quiet by default (no verbose noise)" "$BINARY net AS13335" "[verbose]" false
run_test "info shown by default" "$BINARY net AS13335" "[info]"
run_test "--verbose enables verbose" "$BINARY net AS13335 -v" "[verbose]"
run_test "--debug implies verbose" "$BINARY net AS13335 --debug" "[verbose]"
run_test "--debug enables debug" "$BINARY net AS13335 --debug" "[debug]"
run_test "--quiet hides info" "$BINARY net AS13335 -q" "[info]" false
run_test "--quiet hides verbose" "$BINARY net AS13335 -q" "[verbose]" false
run_test "results survive --quiet" "$BINARY net AS13335 -q" "1.1.1.0/24"
run_exit "--verbose and --quiet conflict" "$BINARY net AS13335 -v -q" 1

echo ""
echo "=== Network Discovery ==="

run_test "net finds CIDRs for an ASN" "$BINARY net AS13335 -q" "1.1.1.0/24"
run_test "net --ip resolves the owning ASN" "$BINARY net 1.1.1.1 --ip -q --detail" "AS13335"
run_test "net --detail adds columns" "$BINARY net 1.1.1.1 --ip -q -d" "Cloudflare"
run_test "net default output is a bare CIDR" "$BINARY net 1.1.1.1 --ip -q" "1.1.1.0/24"

echo ""
echo "=== Output Formats ==="

run_test "net -f json" "$BINARY net 1.1.1.1 --ip -q -f json" '"asn":"AS13335"'
run_test "net -f csv writes a header" "$BINARY net 1.1.1.1 --ip -q -f csv" "input,asn,cidr,org,country,source"
run_test "net -f flat" "$BINARY net 1.1.1.1 --ip -q -f flat" "1.1.1.0/24"
run_test "cluster -f json" "echo 1.1.1.1 | $BINARY cluster -q -f json" '"asn":"AS13335"'
run_test "cluster -f csv writes a header" "echo 1.1.1.1 | $BINARY cluster -q -f csv" "asn,cidr,count"
run_test "cdn text shows vendor and type" "$BINARY cdn 1.1.1.1 -q" "cloudflare"
run_test "cdn -f json" "$BINARY cdn 1.1.1.1 -q -f json" '"is_cdn":true'
run_test "cdn --exclude drops CDN IPs" "$BINARY cdn 1.1.1.1 -q --exclude" "1.1.1.1" false

echo ""
echo "=== url ==="

# One source with a tight request budget keeps this quick and kind to the archives.
URL_FAST="--sources otx --limit-requests 1 -q"

run_test "url returns in-scope URLs" "$BINARY url tesla.com $URL_FAST" "tesla.com"
run_test "url -f json" "$BINARY url tesla.com $URL_FAST -f json" '"source":"otx"'
run_test "url -f csv writes a header" "$BINARY url tesla.com $URL_FAST -f csv" "input,url,source,status,mime,timestamp"
run_test "url --detail tags the source" "$BINARY url tesla.com $URL_FAST --detail" "| otx"
run_test "url -f flat ignores --detail" "$BINARY url tesla.com $URL_FAST -f flat --detail" "| otx" false
run_test "url --keywords filters" "$BINARY url tesla.com $URL_FAST --keywords 'THIS_MATCHES_NOTHING'" "tesla.com" false
run_test "url subdomains are included by default" "$BINARY url tesla.com --sources wayback --limit-requests 2 -v 2>&1" "*.tesla.com/*"
run_test "url --no-subs narrows the scope" "$BINARY url tesla.com --sources wayback --limit-requests 2 --no-subs -v 2>&1" "Scope for tesla.com: tesla.com/*"

echo ""
echo "=== Input Methods ==="

TEMP_INPUT=$(mktemp)
echo "AS13335" > "$TEMP_INPUT"

run_test "positional argument" "$BINARY net AS13335 -q" "1.1.1.0/24"
run_test "-i flag" "$BINARY net -i AS13335 -q" "1.1.1.0/24"
run_test "--input flag" "$BINARY net --input AS13335 -q" "1.1.1.0/24"
run_test "-I file" "$BINARY net -I $TEMP_INPUT -q" "1.1.1.0/24"
run_test "stdin" "echo AS13335 | $BINARY net -q" "1.1.1.0/24"
run_test "-I - reads stdin" "echo AS13335 | $BINARY net -I - -q" "1.1.1.0/24"

run_test "several positional targets" \
    "$BINARY cluster 1.1.1.1 8.8.8.8 9.9.9.9 -q -f flat | wc -l | tr -d ' '" "3"
run_test "arguments, -i, and -I merge" \
    "printf '9.9.9.9\n' > $TEMP_INPUT && $BINARY cluster 1.1.1.1 -i 8.8.8.8 -I $TEMP_INPUT -q -f flat | wc -l | tr -d ' '" "3"
run_test "merged targets are deduplicated" \
    "printf '1.1.1.1\n' > $TEMP_INPUT && $BINARY cluster 1.1.1.1 -i 1.1.1.1 -I $TEMP_INPUT -q -f flat | wc -l | tr -d ' '" "1"
run_test "-i accepts a value starting with a dash" \
    "$BINARY cluster -i --endpoint-url --debug 2>&1" "Read target from --input"
run_test "-- ends flag parsing" \
    "$BINARY cluster --debug -- 1.1.1.1 -q 2>&1" "Read 2 target(s) from arguments"
run_test "comments in the input file are ignored" \
    "printf '# a comment\nAS13335\n' > $TEMP_INPUT && $BINARY net -I $TEMP_INPUT -q" "1.1.1.0/24"
run_test "arguments do not block on stdin" "$BINARY net AS13335 -q" "1.1.1.0/24"

rm -f "$TEMP_INPUT"

echo ""
echo "=== Output File ==="

TEMP_OUTPUT=$(mktemp)

run_test "-o writes results to a file" \
    "$BINARY net 1.1.1.1 --ip -q -o $TEMP_OUTPUT && cat $TEMP_OUTPUT" "1.1.1.0/24"
run_test "-o overwrites on rerun" \
    "$BINARY net 1.1.1.1 --ip -q -o $TEMP_OUTPUT && $BINARY net 1.1.1.1 --ip -q -o $TEMP_OUTPUT && [ \"\$(wc -l < $TEMP_OUTPUT)\" -eq 1 ] && echo SINGLE" "SINGLE"
run_test "--append keeps prior results" \
    "$BINARY net 1.1.1.1 --ip -q -o $TEMP_OUTPUT --append && [ \"\$(wc -l < $TEMP_OUTPUT)\" -eq 2 ] && echo APPENDED" "APPENDED"

rm -f "$TEMP_OUTPUT"

echo ""
echo "=== Flags and Help ==="

run_test "-I is --input-file" "$BINARY net --help" "-I, --input-file"
run_test "-q is --quiet" "$BINARY net --help" "-q, --quiet"
run_test "-f is --format" "$BINARY net --help" "-f, --format"
run_test "-v is --verbose" "$BINARY net --help" "-v, --verbose"
run_test "cert --detail exists" "$BINARY cert --help" "-d, --detail"
run_test "cdn --exclude exists" "$BINARY cdn --help" "--exclude"
run_test "github --subs exists" "$BINARY github --help" "--subs"
run_test "related --sources exists" "$BINARY related --help" "-s, --sources"
run_test "url --sources exists" "$BINARY url --help" "-s, --sources"
run_test "url --no-subs exists" "$BINARY url --help" "--no-subs"
run_test "url --limit-requests exists" "$BINARY url --help" "--limit-requests"
run_test "url --limit-collections exists" "$BINARY url --help" "--limit-collections"
run_test "url is listed in the root help" "$BINARY --help" "url"
run_test "net --live exists" "$BINARY net --help" "--live"
run_test "cluster replaced ipc" "$BINARY --help" "cluster"
run_test "ipc is gone" "$BINARY ipc --help" "unknown command"
run_test "--no-color strips ANSI" "$BINARY version --no-color" "Version:"

echo ""
echo "=== Skills ==="

run_test "skills is listed in the root help" "$BINARY --help" "skills"
run_test "skills lists the bundled skill" "$BINARY skills -q" "metabigor"
run_test "skills list -f flat prints the name" "$BINARY skills list -f flat -q" "metabigor"
run_test "skills list -f json emits a JSON object" "$BINARY skills -f json -q" '"embed_path"'
run_test "skills get prints SKILL.md frontmatter" "$BINARY skills get metabigor" "name: metabigor"
run_test "skills get --full includes references" "$BINARY skills get metabigor --full" "references/recipes.md"
run_test "skills get --all works without a name" "$BINARY skills get --all" "name: metabigor"
run_exit "skills get with an unknown name exits 1" "$BINARY skills get nope" 1
run_exit "skills get with no name exits 1" "$BINARY skills get" 1
run_test "skills install has --agent" "$BINARY skills install --help" "--agent"
run_test "skills install rejects a bad --agent" "$BINARY skills install --agent nope" "unknown --agent"

SKILLS_DIR="$(mktemp -d)"
run_test "skills install --dir copies the bundle" \
    "$BINARY skills install --dir '$SKILLS_DIR' && cat '$SKILLS_DIR/metabigor/SKILL.md'" "name: metabigor"
run_test "skills install skips an existing bundle" \
    "$BINARY skills install --dir '$SKILLS_DIR'" "already installed"
run_test "skills install --force overwrites" \
    "$BINARY skills install --dir '$SKILLS_DIR' --force" "Installed"
rm -rf "$SKILLS_DIR"

echo ""
echo "=== Pipelines ==="

run_test "net -f flat pipes into cluster" \
    "$BINARY net 1.1.1.1 --ip -q -f flat | head -1 | $BINARY cluster -q" "AS13335"

echo ""
echo "=========================================="
echo -e "Test Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
echo -e "${RED}Some tests failed!${NC}"
exit 1
