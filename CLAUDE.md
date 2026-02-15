# metabigor - AI Agent Development Guide

This file provides comprehensive guidance for AI agents (Claude, etc.) working on the metabigor codebase.

## Project Overview

**metabigor** is an OSINT (Open Source Intelligence) CLI tool written in Go that provides network reconnaissance and information gathering capabilities without requiring API keys. The name combines "metadata" + "bigor" (bigger) - making OSINT bigger by aggregating metadata from multiple free sources.

**Core Philosophy**: API-key-free OSINT automation with Unix-style composability (pipe-friendly).

**Key Characteristics**:
- **Simple**: Unix philosophy - do one thing well, compose with pipes
- **Free**: No API keys, no accounts, no rate limit headaches
- **Respectful**: Don't abuse free services (delays, low concurrency)
- **Reliable**: Retry logic, fallbacks, graceful degradation
- **Hackable**: Easy to extend, clear code, good examples

## Architecture

### Project Structure

```
metabigor/
├── cmd/metabigor/          # Main entry point
│   └── main.go             # CLI initialization, version info injection
├── internal/
│   ├── cli/                # Command definitions (cobra commands)
│   │   ├── root.go         # Root command & global flags
│   │   ├── net.go          # Network discovery (ASN/CIDR)
│   │   ├── cert.go         # Certificate transparency search
│   │   ├── ip.go           # IP enrichment (Shodan InternetDB)
│   │   ├── github.go       # GitHub code search (grep.app)
│   │   ├── ipc.go          # IP clustering by ASN
│   │   ├── related.go      # Related domain discovery
│   │   ├── cdn.go          # CDN detection
│   │   ├── update.go       # ASN/Country database updater
│   │   └── version.go      # Version command
│   ├── asndb/              # ASN database management
│   │   ├── db.go           # Database loading and lookup
│   │   ├── download.go     # Database downloader
│   │   └── models.go       # ASN entry struct
│   ├── countrydb/          # Country database management (NEW)
│   │   ├── db.go           # Database loading and lookup
│   │   ├── download.go     # Database downloader
│   │   └── models.go       # Country entry struct
│   ├── cert/               # Certificate transparency logic
│   │   └── crt.go          # crt.sh search, parsing, grouping
│   ├── gitsearch/          # GitHub/grep.app integration
│   │   └── grepapp.go      # grep.app API client and parsing
│   ├── httpclient/         # HTTP client with retry logic
│   │   ├── client.go       # Retryable HTTP client
│   │   └── chrome.go       # Headless Chrome automation
│   ├── ipinfo/             # IP information gathering
│   │   ├── cluster.go      # IP clustering by ASN
│   │   └── internetdb.go   # Shodan InternetDB client
│   ├── netdiscovery/       # Network range discovery
│   │   ├── sources.go      # Source definitions
│   │   ├── static.go       # Local database lookup
│   │   └── dynamic.go      # Live API/scraping
│   ├── related/            # Related domain discovery sources
│   │   ├── crt.go          # CT log-based
│   │   ├── whois.go        # Reverse WHOIS
│   │   ├── analytics.go    # Google Analytics tracking
│   │   └── builtwith.go    # BuiltWith API
│   ├── options/            # Shared options/config structs
│   │   └── options.go      # All command options
│   ├── output/             # Logging and output utilities
│   │   ├── logger.go       # Color logging (Info, Debug, etc.)
│   │   └── writer.go       # Thread-safe output writer
│   └── runner/             # Parallel execution utilities
│       └── runner.go       # Input reading, parallel processing
└── public/                 # Embedded assets
    ├── embed.go            # Go embed directives
    ├── ip-to-asn.csv.zip   # Embedded ASN database (fallback)
    └── ip-to-country.csv.zip # Embedded country database (fallback)
```

### Command Architecture

Each command follows this consistent pattern:

1. **Definition** (`internal/cli/<command>.go`):
   - Cobra command setup with flags in `init()`
   - `Run` function orchestrates execution

2. **Logic** (`internal/<domain>/`):
   - Business logic in separate packages
   - Querying external sources
   - Parsing and data extraction

3. **Runner** (`internal/runner/`):
   - Parallel execution with worker pools
   - Input deduplication

4. **Output** (`internal/output/`):
   - Logging with color and levels
   - Thread-safe file writing

### Key Dependencies

```go
// CLI Framework
github.com/spf13/cobra@v1.10.2

// HTTP Client
github.com/hashicorp/go-retryablehttp@v0.7.8

// HTML Parsing
github.com/PuerkitoBio/goquery@v1.11.0

// Browser Automation
github.com/chromedp/chromedp@v0.14.2

// Terminal Rendering
github.com/charmbracelet/glamour@v0.8.0  // Markdown
github.com/fatih/color@v1.18.0           // Colors

// Network Utilities
github.com/projectdiscovery/mapcidr@v1.1.97    // CIDR manipulation
github.com/projectdiscovery/cdncheck@v1.2.23   // CDN detection
```

## Commands Reference

### 1. `net` - Network Discovery

**Purpose**: Discover IP ranges (CIDRs) from ASN, organization, IP, or domain.

**Modes**:
- **Static** (default): Uses local ASN database (fast, offline)
- **Dynamic** (`--dynamic`): Queries live sources (slower, more complete)

**Flags**:
- `--asn`: Force ASN input mode
- `--org`: Force organization input mode
- `--ip`: Force IP input mode
- `--domain`: Force domain input mode
- `--dynamic`: Use live sources instead of local database
- `--detail`: Show detailed BGP info (type, description, country)

**Input Auto-Detection**:
```go
// Detects: ASN (AS####), IP (x.x.x.x), CIDR (x.x.x.x/##), Org (text)
// Falls back to org search if no pattern matches
```

**Data Sources**:
- Static: Local ASN database (`~/.metabigor/ip-to-asn.csv`)
- Dynamic: bgp.he.net (requires Chrome), ipinfo.io, asnlookup.com

**Output Format**:
```
AS13335 | 1.0.0.0/24 | Cloudflare, Inc. | OC (AU)
AS13335 | 1.1.1.0/24 | Cloudflare, Inc. | OC (AU)
```

**Example**:
```bash
echo "AS13335" | metabigor net
echo "Cloudflare" | metabigor net --org
echo "1.1.1.1" | metabigor net --ip
echo "cloudflare.com" | metabigor net --domain --dynamic
```

**Implementation Notes**:
- Country enrichment added via `countrydb` package
- Static mode: Binary search on sorted ASN database
- Dynamic mode: Parallel queries with fallback chain
- Chrome required for bgp.he.net (JavaScript-rendered)

---

### 2. `cert` - Certificate Transparency Search

**Purpose**: Search certificate transparency logs via crt.sh for subdomain discovery.

**Flags**:
- `--clean`: Strip wildcard prefix `*.` from domains
- `--wildcard`: Show only wildcard entries
- `--simple`: Output only domain names (old behavior, backward compatible)

**Query Strategy**:
1. Try organization search: `https://crt.sh/?O=<query>`
2. Fall back to general search: `https://crt.sh/?q=<query>`

**Output Modes**:

**Default (Grouped View)**:
```
Domain: api.example.com
  Cert IDs (3): 12345, 67890, 11111
  Not Before: 2024-01-01
  Not After: 2025-12-31
  Issuers: Let's Encrypt, DigiCert
  Common Names: example.com

Domain: *.example.com
  Cert IDs (5): 22222, 33333, ...
  ...
```

**Simple Mode** (`--simple`):
```
api.example.com
*.example.com
example.com
```

**JSON Mode**:
```json
{"domain":"api.example.com","cert_ids":["12345","67890"],"count":2,"first_seen":"2024-01-01","last_expires":"2025-12-31","issuers":["C=US, O=Let's Encrypt, CN=R3"]}
```

**Example**:
```bash
echo "hackerone.com" | metabigor cert
echo "Tesla Motors" | metabigor cert
echo "example.com" | metabigor cert --clean --wildcard
echo "example.com" | metabigor cert --simple -o domains.txt
```

**Implementation Notes**:
- HTML table parsing with goquery
- Column validation: cert ID must be numeric
- Handles multi-domain certificates (SANs)
- `GroupByDomain()` aggregates certs per domain
- Tracks: cert IDs, dates, issuers, common names
- Gracefully skips malformed rows

---

### 3. `ip` - IP Enrichment

**Purpose**: Enrich IPs with port, service, and vulnerability data via Shodan InternetDB (free, no API key).

**Flags**:
- `--flat`: Output as `IP:PORT` (one per line)
- `--csv`: CSV format output

**Auto-Expansion**:
- CIDR input: Automatically expands to all IPs
- Example: `192.168.1.0/24` → 254 IPs

**Data Source**: `https://internetdb.shodan.io/<ip>`

**Output Format**:

**Default (JSON)**:
```json
{"ip":"1.1.1.1","ports":[80,443],"hostnames":["one.one.one.one"],"cpes":[],"vulns":[],"tags":["cdn"]}
```

**Flat Mode**:
```
1.1.1.1:80
1.1.1.1:443
```

**Example**:
```bash
echo "1.1.1.1" | metabigor ip
echo "8.8.8.8/30" | metabigor ip --flat
cat ips.txt | metabigor ip --csv -o results.csv
```

**Implementation Notes**:
- Free Shodan API, no authentication
- Concurrent IP lookups (default 5 workers)
- CIDR expansion via `mapcidr` library
- Empty response if IP not in Shodan database

---

### 4. `github` - GitHub Code Search

**Purpose**: Search GitHub code via grep.app API to find secrets, subdomains, or code patterns.

**Flags**:
- `--detail`: Show formatted code snippets with context
- `--page <n>`: Pagination limit (0 = unlimited)

**Rate Limiting**: **5 second delay between pages** (respectful to free service)

**Data Source**: `https://grep.app/api/search?q=<query>&page=<n>`

**Output Modes**:

**Default (Domain Extraction)**:
```
api.example.com
staging.example.com
```

**Detail Mode** (`--detail`):
```
Repository: owner/repo
File: path/to/file.js
Branch: main
Matches: 3

  12 | const API_URL = "https://api.example.com";
  13 | const SECRET_KEY = "sk_live_...";
  14 |
```

**Example**:
```bash
echo "example.com" | metabigor github
echo "api_key OR secret" | metabigor github --detail
echo "Authorization: Bearer" | metabigor github --page 5
```

**Implementation Notes**:
- Modern User-Agent (Chrome 119) with Sec-Ch-Ua headers
- HTML parsing for code snippets (table format)
- Fallback to text cleaning if HTML parse fails
- Glamour markdown renderer for formatted output
- Auto-extracts domains matching query pattern
- Pagination with configurable limits

---

### 5. `related` - Related Domain Discovery

**Purpose**: Find related domains via certificate logs, WHOIS, and analytics tracking.

**Flags**:
- `-s, --source <type>`: Source to use (crt, whois, ua, gtm, all)

**Sources**:
1. **crt** - Certificate transparency (crt.sh)
2. **whois** - Reverse WHOIS (viewdns.info)
3. **ua** - Google Analytics tracking (builtwith.com)
4. **gtm** - Google Tag Manager (builtwith.com)
5. **all** - All sources (default)

**Example**:
```bash
echo "example.com" | metabigor related
echo "example.com" | metabigor related --source whois
echo "example.com" | metabigor related -s all
```

**Implementation Notes**:
- Multiple sources run in parallel
- HTML parsing for viewdns and builtwith
- crt.sh same backend as `cert` command
- Deduplicates results across sources

---

### 6. `ipc` - IP Clustering

**Purpose**: Group IPs by ASN for infrastructure mapping.

**Output Format**:
```
AS13335 | 104.16.0.0/12 | 1048576 IPs | Cloudflare, Inc. | US
AS15169 | 8.8.8.0/24 | 256 IPs | Google LLC | US
```

**Example**:
```bash
cat ips.txt | metabigor ipc
echo -e "1.1.1.1\n8.8.8.8" | metabigor ipc
```

**Implementation Notes**:
- Uses local ASN database
- Groups by ASN, calculates CIDR coverage
- Counts IPs per ASN
- Shows org and country info

---

### 7. `cdn` - CDN/WAF Detection

**Purpose**: Detect if IPs/domains are behind CDN or WAF.

**Flags**:
- `--strip-cdn`: Output only non-CDN IPs

**CDN Libraries**: Uses projectdiscovery/cdncheck

**Output Format**:
```
1.1.1.1 | Cloudflare | cdn
93.184.216.34 | Fastly | cdn
192.168.1.1 | - | no-cdn
```

**Example**:
```bash
echo "example.com" | metabigor cdn
cat ips.txt | metabigor cdn --strip-cdn
```

---

### 8. `update` - Database Updater

**Purpose**: Download/update ASN and country databases.

**Downloads From**:
- `https://github.com/iplocate/ip-address-databases/raw/main/ip-to-asn/ip-to-asn.csv.zip`
- `https://github.com/iplocate/ip-address-databases/raw/main/ip-to-country/ip-to-country.csv.zip`

**Saves To**:
- `~/.metabigor/ip-to-asn.csv`
- `~/.metabigor/ip-to-country.csv`

**Example**:
```bash
metabigor update
```

**Implementation Notes**:
- Downloads, validates, extracts ZIP files
- Atomic replacement (writes to temp, then renames)
- Progress logging

---

### 9. `version` - Version Information

**Purpose**: Display version, commit, and build info.

**Output**:
```
metabigor v2.1.0 (commit: abc1234, built: 2024-01-15)
Author: j3ssie
```

**Version Injection**:
```go
// Set at build time via ldflags
var (
    version   = "dev"
    commit    = "none"
    buildDate = "unknown"
)
```

**Build Command**:
```bash
go build -ldflags "-X main.version=v2.1.0 -X main.commit=$(git rev-parse HEAD)"
```

## Development Guidelines

### Code Style

#### 1. Logging Conventions

```go
output.Info("message")     // [info] Blue - General informational messages
output.Good("message")     // [+] Green - Successful operations
output.Warn("message")     // [!] Yellow - Warnings
output.Error("message")    // [-] Red - Errors
output.Verbose("message")  // [verbose] Cyan - Detailed progress messages
output.Debug("message")    // [debug] Magenta - Debug mode (--debug) messages
```

**When to use each**:
- `Info`: Starting operations, counts, summaries
- `Good`: Successful completions, found results
- `Warn`: Non-fatal issues, fallbacks used
- `Error`: Errors that don't stop processing (ALWAYS shown, even in silent mode)
- `Verbose`: Progress updates, intermediate results
- `Debug`: HTTP requests/responses, parsing details

**Silent Mode**: All output (Info/Good/Warn/Verbose) is hidden in silent mode (`-q/--silent`) EXCEPT errors.
- Normal mode (default): Shows all messages (Info, Good, Warn, Verbose, Debug if --debug)
- Silent mode (`-q`): Shows only Error messages
- Debug mode (`--debug`): Shows all messages including Debug

#### 2. Input Handling

All commands support **three input methods** (automatically deduplicated):

```go
inputs := runner.ReadInputs(opt.Input, opt.InputFile, args)
```

1. **Stdin**: `echo "input" | metabigor <cmd>`
2. **-i/--input flag**: `metabigor <cmd> -i "value"` or `metabigor <cmd> --input "value"`
3. **-I file**: `metabigor <cmd> -I file.txt`

**Implementation**:
```go
func ReadInputs(input, inputFile string, args []string) []string {
    // Reads from all sources
    // Deduplicates using map
    // Returns sorted unique slice
}
```

#### 3. Output Handling

```go
w, err := output.NewWriter(opt.Output, opt.JSONOutput)
if err != nil {
    output.Error("%v", err)
    return
}
defer w.Close()

// Text output
w.WriteString("result line")

// JSON output
w.WriteJSON(structData)
```

**Writer Features**:
- Thread-safe (sync.Mutex)
- Auto-deduplication (tracks seen strings)
- Dual output (stdout + file if specified)
- Buffered writes with auto-flush

#### 4. HTTP Requests

**Always use the retryable HTTP client**:

```go
client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)
body, err := httpclient.Get(client, url)
if err != nil {
    output.Debug("Request failed: %v", err)
    return
}
defer resp.Body.Close()
```

**Configuration**:
- User-Agent: Chrome 119 with modern headers
- TLS: `InsecureSkipVerify: true` (intentional for OSINT)
- Retries: Default 3 (configurable via `--retry`)
- Timeout: Default 40s (configurable via `--timeout`)
- Proxy: Configurable via `--proxy`

**Rate Limiting**:
```go
// Example from github command
time.Sleep(5 * time.Second) // Between pages
```

#### 5. Concurrency

**Standard Pattern**:

```go
runner.RunParallel(inputs, opt.Concurrency, func(input string) {
    // This function runs concurrently
    // Each input processed independently

    results := processInput(input)
    for _, r := range results {
        w.WriteString(r) // Writer is thread-safe
    }
})
```

**Guidelines**:
- Default concurrency: 5 (respectful to free services)
- User-controllable via `-c` flag
- Failures in one worker don't affect others
- Use `output.Debug()` for per-worker logging

#### 6. Error Handling

**Don't panic, don't exit - continue processing**:

```go
// GOOD: Log error, continue with other inputs
if err != nil {
    output.Error("Failed to process %s: %v", input, err)
    continue
}

// BAD: Don't do this
if err != nil {
    panic(err)
}

// BAD: Don't do this either
if err != nil {
    os.Exit(1)
}
```

**HTTP Response Handling**:
```go
resp, err := client.Get(url)
if err != nil {
    output.Debug("Request failed: %v", err)
    return
}
defer resp.Body.Close() // ALWAYS defer close

body, err := io.ReadAll(resp.Body)
if err != nil {
    output.Debug("Read failed: %v", err)
    return
}
```

**JSON Parsing**:
```go
var data SomeStruct
if err := json.Unmarshal(body, &data); err != nil {
    output.Debug("JSON parse failed: %v", err)
    return // Don't crash, just skip
}
```

### Adding New Commands

To add a new command (example: `whois`):

#### 1. Create Command File

`internal/cli/whois.go`:

```go
package cli

import (
    "github.com/j3ssie/metabigor/internal/output"
    "github.com/j3ssie/metabigor/internal/runner"
    "github.com/spf13/cobra"
)

func init() {
    // Register flags
    whoisCmd.Flags().BoolVar(&opt.Whois.Raw, "raw", false, "Show raw WHOIS output")
    rootCmd.AddCommand(whoisCmd)
}

var whoisCmd = &cobra.Command{
    Use:   "whois",
    Short: "WHOIS lookup for domains",
    Long:  `Perform WHOIS lookups on domains and parse registrar information.`,
    Example: `  # Basic WHOIS lookup
  echo "example.com" | metabigor whois
  metabigor whois --input "google.com"

  # Raw output
  echo "example.com" | metabigor whois --raw

  # JSON output
  echo "example.com" | metabigor whois --json`,
    Run: runWhois,
}

func runWhois(cmd *cobra.Command, args []string) {
    output.SetupLogger(opt.Verbose, opt.Debug, opt.NoColor)
    inputs := runner.ReadInputs(opt.Input, opt.InputFile, args)
    if len(inputs) == 0 {
        output.Error("No input provided")
        return
    }

    w, err := output.NewWriter(opt.Output, opt.JSONOutput)
    if err != nil {
        output.Error("%v", err)
        return
    }
    defer w.Close()

    output.Info("Looking up WHOIS for %d domain(s)", len(inputs))
    client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

    runner.RunParallel(inputs, opt.Concurrency, func(domain string) {
        output.Verbose("WHOIS lookup for %q", domain)

        // TODO: Implement WHOIS logic
        result := performWhoisLookup(client, domain)

        if opt.JSONOutput {
            w.WriteJSON(result)
        } else {
            w.WriteString(result.String())
        }
    })
}
```

#### 2. Add Options

`internal/options/options.go`:

```go
type Options struct {
    // ... existing fields ...
    Whois WhoisOptions
}

type WhoisOptions struct {
    Raw bool
}
```

#### 3. Create Business Logic

`internal/whois/lookup.go`:

```go
package whois

import (
    "github.com/hashicorp/go-retryablehttp"
)

type WhoisResult struct {
    Domain     string `json:"domain"`
    Registrar  string `json:"registrar"`
    CreatedAt  string `json:"created_at"`
    ExpiresAt  string `json:"expires_at"`
    NameServer string `json:"nameserver"`
}

func PerformLookup(client *retryablehttp.Client, domain string) (*WhoisResult, error) {
    // Implement WHOIS logic
    return &WhoisResult{}, nil
}
```

#### 4. Test

```bash
# Build
make build

# Test all input methods
echo "example.com" | ./bin/metabigor whois
./bin/metabigor whois --input "example.com"
echo "example.com" > /tmp/domains.txt && ./bin/metabigor whois -f /tmp/domains.txt

# Test flags
echo "example.com" | ./bin/metabigor whois --raw
echo "example.com" | ./bin/metabigor whois --json
echo "example.com" | ./bin/metabigor whois --debug

# Test output
echo "example.com" | ./bin/metabigor whois -o output.txt
```

### Testing Commands

#### Unit Tests

Standard Go tests in `*_test.go` files:

```go
func TestWhoisLookup(t *testing.T) {
    client := httpclient.NewClient(10, 3, "")
    result, err := whois.PerformLookup(client, "example.com")
    if err != nil {
        t.Fatalf("Lookup failed: %v", err)
    }
    if result.Domain != "example.com" {
        t.Errorf("Expected example.com, got %s", result.Domain)
    }
}
```

Run: `make test`

#### End-to-End Tests

Comprehensive CLI tests in `test/run-e2e.sh`:

```bash
# Add to test/run-e2e.sh

test_whois_basic() {
    echo "example.com" | $BIN whois > /tmp/whois_output.txt
    assert_contains "/tmp/whois_output.txt" "example.com"
}

test_whois_json() {
    echo "example.com" | $BIN whois --json | jq -e '.domain == "example.com"'
    assert_success
}
```

Run: `make e2e`

#### Linting

```bash
make lint
```

See `.golangci.yml` for configuration.

### Rate Limiting & Ethical Use

**CRITICAL**: metabigor queries free public services. Be respectful:

#### Service-Specific Guidelines

| Service | Rate Limit | Implementation |
|---------|-----------|----------------|
| **grep.app** | No official limit | 5 second delay between pages |
| **crt.sh** | No official limit | Default concurrency (5) |
| **Shodan InternetDB** | Free tier | Reasonable concurrency |
| **viewdns.info** | Unknown | Low concurrency |
| **bgp.he.net** | Unknown | Chrome automation (slower) |

#### General Rules

1. **Delays**: Add delays for paginated/high-volume requests
2. **Concurrency**: Default to 5 workers, allow user to adjust
3. **Pagination**: Let users control with `--page` flags
4. **Caching**: Consider caching for repeated queries
5. **User-Agent**: Use legitimate, modern browser UA
6. **Respect robots.txt**: Even if scraping is allowed, be gentle

#### Example: Adding Rate Limiting

```go
const DelayBetweenPages = 5 * time.Second

for page := 1; page <= maxPages; page++ {
    results := fetchPage(page)
    processResults(results)

    if page < maxPages {
        output.Debug("Waiting %v before next page", DelayBetweenPages)
        time.Sleep(DelayBetweenPages)
    }
}
```

### Security Considerations

#### 1. Input Validation

**Always sanitize user inputs before using in URLs/queries**:

```go
// GOOD: URL-encode user input
url := fmt.Sprintf("https://api.example.com/search?q=%s", url.QueryEscape(query))

// BAD: Direct string interpolation
url := fmt.Sprintf("https://api.example.com/search?q=%s", query) // Injection risk!
```

#### 2. No Credential Storage

**Design principle**: This tool is API-key-free by design.

- Don't add features requiring API keys
- Don't store credentials
- Prefer free public APIs

#### 3. TLS Verification

**Intentionally disabled for OSINT**:

```go
// This is INTENTIONAL for reconnaissance
TLSClientConfig: &tls.Config{
    InsecureSkipVerify: true, // Required for OSINT work
}
```

**Why**: Many targets have invalid/self-signed certs. This is an OSINT tool, not a production service.

#### 4. HTML Entity Decoding

**Prevent injection when parsing HTML**:

```go
import "html"

text := html.UnescapeString(rawHTML)
```

#### 5. Command Injection

**Never use user input in shell commands**:

```go
// BAD: Command injection vulnerability
cmd := exec.Command("sh", "-c", "curl "+userInput)

// GOOD: Use proper HTTP client
resp, err := client.Get(userInput)
```

## Database Management

### ASN Database

**Purpose**: Maps IP ranges to ASN, organization, and country.

**Format**: CSV with 6 columns
```csv
network,asn,country_code,name,org,domain
1.0.0.0/24,13335,AU,CLOUDFLARENET,Cloudflare Inc,cloudflare.com
```

**Location**:
- Download cache: `~/.metabigor/ip-to-asn.csv`
- Embedded fallback: `public/ip-to-asn.csv.zip`

**Loading**:
```go
db, err := asndb.LoadDatabase()
// 1. Try ~/.metabigor/ip-to-asn.csv
// 2. If missing, extract from public/ip-to-asn.csv.zip
// 3. If extract fails, download from GitHub
```

**Lookup**:
```go
entry, err := db.LookupIP("1.1.1.1")
// Returns: ASN, Org, Country, CIDR
```

**Implementation**:
- In-memory: ~2M+ entries loaded
- Binary search on sorted StartIP
- Fast: O(log n) lookup

**Update**:
```bash
metabigor update  # Downloads latest from GitHub
make update        # Same, for development
```

---

### Country Database

**Purpose**: Maps IP ranges to continent and country codes (NEW feature).

**Format**: CSV with 3 columns
```csv
network,continent_code,country_code
1.0.0.0/24,OC,AU
1.1.1.0/24,OC,AU
```

**Location**:
- Download cache: `~/.metabigor/ip-to-country.csv`
- Embedded fallback: `public/ip-to-country.csv.zip`

**Loading**:
```go
db, err := countrydb.LoadDatabase()
// Same fallback chain as ASN database
```

**Lookup**:
```go
entry, err := db.LookupIP("1.1.1.1")
// Returns: ContinentCode, CountryCode
```

**Usage**: Automatically enriches `net` command output

**Output Example**:
```
AS13335 | 1.1.1.0/24 | Cloudflare, Inc. | OC (AU)
       ASN    CIDR         Organization      Continent (Country)
```

**Implementation Notes**:
- Similar structure to ASN database
- Binary search for performance
- Gracefully handles missing database (warns, continues)
- Loads on-demand (lazy loading)

---

### Database Update Workflow

```bash
# User runs update
metabigor update

# Download process
1. Create temp directory
2. Download ZIP from GitHub
3. Validate ZIP integrity
4. Extract CSV
5. Atomic rename to ~/.metabigor/
6. Clean up temp files

# Fallback chain
Database Load:
├─ Try: ~/.metabigor/ip-to-asn.csv
├─ Try: Extract public/ip-to-asn.csv.zip
└─ Try: Download from GitHub
```

## Build & Release

### Local Development

```bash
# Build binary
make build
# Output: bin/metabigor

# Install to $GOPATH/bin
make install

# Run unit tests
make test

# Run end-to-end tests (requires build)
make e2e
# Runs 29 CLI tests

# Format code
make fmt

# Vet code
make vet

# Lint code (requires golangci-lint)
make lint
# See .golangci.yml for configuration

# Update embedded ASN and country databases
make update
# Downloads latest from GitHub

# Clean build artifacts
make clean

# Cross-compile for all platforms
make build-all
# Outputs to dist/:
#   metabigor-linux-amd64
#   metabigor-linux-arm64
#   metabigor-darwin-amd64
#   metabigor-darwin-arm64
#   metabigor-windows-amd64.exe
```

### Version Information

Version info is injected at build time:

```go
// cmd/metabigor/main.go
var (
    version   = "dev"
    commit    = "none"
    buildDate = "unknown"
)
```

**Build with version**:
```bash
go build -ldflags "-X main.version=v2.1.0 -X main.commit=$(git rev-parse HEAD) -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

### Testing Infrastructure

#### End-to-End Tests (`test/run-e2e.sh`)

**Coverage**: 29 test cases across all commands

**Test Categories**:
1. Help text validation
2. Flag parsing
3. Input methods (stdin, -i/--input, -I file)
4. Output modes (text, JSON, file)
5. Global flags (-q/--silent, --debug, --json)
6. Command-specific flags

**Run Tests**:
```bash
make e2e
```

**Add New Test**:
```bash
# In test/run-e2e.sh

test_mycommand_basic() {
    echo "input" | $BIN mycommand > /tmp/output.txt
    assert_contains "/tmp/output.txt" "expected"
}

test_mycommand_json() {
    echo "input" | $BIN mycommand --json | jq -e '.field == "value"'
    assert_success
}
```

**Documentation**: See `test/README.md`

#### Linting (`.golangci.yml`)

**Enabled Linters**:
- errcheck: Check error handling
- gosimple: Simplify code
- govet: Vet for suspicious constructs
- ineffassign: Detect ineffectual assignments
- staticcheck: Static analysis
- unused: Detect unused code
- misspell: Spell checker
- gocritic: Opinionated checks
- gocyclo: Cyclomatic complexity
- revive: Golint replacement
- unconvert: Remove unnecessary conversions
- unparam: Detect unused function parameters

**Security Exceptions**:
```yaml
# gosec disabled - OSINT-specific patterns
linters:
  disable:
    - gosec  # G402 (TLS skip verify), G304 (file paths), etc.
```

**Run Linting**:
```bash
make lint
```

**Install golangci-lint**: https://golangci-lint.run/usage/install/

### GoReleaser (`.goreleaser.yaml`)

**Features**:
- Multi-platform builds (Linux, macOS, Windows)
- Multi-architecture (amd64, arm64, arm)
- **UPX compression** for smaller binaries
- Archive creation (tar.gz, zip)
- SHA256 checksums
- Auto-generated changelog
- GitHub release integration

**Platforms**:
```yaml
- linux/amd64
- linux/arm64
- linux/arm
- darwin/amd64
- darwin/arm64
- windows/amd64
```

**UPX Compression**:
```yaml
upx:
  - enabled: true
    compress: best
    lzma: true
    brute: false
```

**Install UPX** (optional, for compression):
```bash
# macOS
brew install upx

# Linux
sudo apt-get install upx-ucl

# Or download from https://upx.github.io/
```

**Binary Size Impact**:
- Without UPX: ~30-40 MB
- With UPX: ~15-25 MB (40-60% reduction)

**Usage**:
```bash
# Test build locally (no publish)
goreleaser release --snapshot --clean

# Create release (requires GitHub token and git tag)
git tag v2.2.0
git push origin v2.2.0
goreleaser release --clean
```

**Release Includes**:
- Compressed binaries (UPX)
- Source archives
- LICENSE
- README.md
- CLAUDE.md
- SHA256 checksums
- Auto-generated CHANGELOG.md

### CI/CD

**CodeQL Workflow** (`.github/workflows/codeql.yml`):
- Security analysis
- Extended query suite
- Currently disabled (`# on: push`)

**To Enable**:
```yaml
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
```

## Important Implementation Details

### Chrome Automation

**When Used**: JavaScript-heavy sites that don't render without JS

**Example** (`internal/httpclient/chrome.go`):
```go
func ChromeGet(url string) (string, error) {
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    var htmlContent string
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady("body"),
        chromedp.OuterHTML("html", &htmlContent),
    )
    return htmlContent, err
}
```

**Usage**:
```go
// Try standard HTTP first
body, err := httpclient.Get(client, url)
if err != nil || isJavaScriptRendered(body) {
    // Fall back to Chrome
    body, err = httpclient.ChromeGet(url)
}
```

**Requirements**:
- Chrome/Chromium installed on system
- Falls back gracefully if unavailable
- Use sparingly (slower than HTTP)

**Current Users**:
- `net` command (dynamic mode, bgp.he.net)
- `netdiscovery` package (ipinfo.io)

---

### Parallel Execution Pattern

**Implementation** (`internal/runner/runner.go`):

```go
func RunParallel(inputs []string, concurrency int, process func(string)) {
    var wg sync.WaitGroup
    sem := make(chan struct{}, concurrency)

    for _, input := range inputs {
        wg.Add(1)
        sem <- struct{}{} // Acquire semaphore

        go func(in string) {
            defer wg.Done()
            defer func() { <-sem }() // Release semaphore

            process(in)
        }(input)
    }

    wg.Wait()
}
```

**Benefits**:
- Concurrent processing with controlled parallelism
- Worker pool pattern (limits resource usage)
- Safe for high concurrency
- Each input processed independently

**Usage**:
```go
runner.RunParallel(inputs, opt.Concurrency, func(input string) {
    // This runs concurrently
    result := processInput(input)
    w.WriteString(result) // Writer is thread-safe
})
```

---

### Output Writer Pattern

**Thread-Safe Deduplication** (`internal/output/writer.go`):

```go
type Writer struct {
    file     *os.File
    mu       sync.Mutex
    seen     map[string]bool
    jsonMode bool
}

func (w *Writer) WriteString(s string) {
    w.mu.Lock()
    defer w.mu.Unlock()

    if w.seen[s] {
        return // Deduplicate
    }
    w.seen[s] = true

    fmt.Println(s) // stdout
    if w.file != nil {
        fmt.Fprintln(w.file, s) // file
    }
}
```

**Features**:
- Automatic deduplication
- Thread-safe (multiple goroutines can write)
- Dual output (stdout + file)
- JSON mode support

## Common Patterns

### Pattern: OSINT Source Integration

When adding a new OSINT source:

```go
// 1. Create fetcher function
func FetchFromSource(client *retryablehttp.Client, query string) ([]string, error) {
    url := fmt.Sprintf("https://source.com/search?q=%s", url.QueryEscape(query))

    body, err := httpclient.Get(client, url)
    if err != nil {
        return nil, err
    }

    return parseResponse(body), nil
}

// 2. Parse response
func parseResponse(body string) []string {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
    if err != nil {
        output.Debug("Parse error: %v", err)
        return nil
    }

    var results []string
    doc.Find("selector").Each(func(i int, s *goquery.Selection) {
        result := s.Text()
        results = append(results, result)
    })
    return results
}

// 3. Handle pagination
func FetchAllPages(client *retryablehttp.Client, query string, maxPages int) []string {
    var allResults []string

    for page := 1; page <= maxPages; page++ {
        results := FetchFromSource(client, query, page)
        allResults = append(allResults, results...)

        if len(results) == 0 || page >= maxPages {
            break
        }

        time.Sleep(5 * time.Second) // Rate limit
    }

    return allResults
}

// 4. Provide both text and JSON output
type Result struct {
    Field string `json:"field"`
}

func (r Result) String() string {
    return r.Field
}

// In command:
if opt.JSONOutput {
    w.WriteJSON(result)
} else {
    w.WriteString(result.String())
}
```

### Pattern: Error Recovery

**Continue on error, don't fail**:

```go
runner.RunParallel(inputs, opt.Concurrency, func(input string) {
    result, err := processInput(input)
    if err != nil {
        output.Error("Failed %s: %v", input, err)
        return // Continue with next input
    }

    w.WriteString(result)
})
```

### Pattern: Graceful Degradation

**Try best option first, fall back as needed**:

```go
// Try fast local database first
results := staticLookup(db, query)
if len(results) == 0 {
    output.Verbose("Local lookup empty, trying live sources")
    results = dynamicLookup(client, query)
}

// Try HTTP first, fall back to Chrome
body, err := httpclient.Get(client, url)
if err != nil || requiresJS(body) {
    output.Verbose("Falling back to Chrome for JS rendering")
    body, err = httpclient.ChromeGet(url)
}
```

## Debugging Tips

1. **Use Debug Mode**: `--debug` shows HTTP requests/responses
   ```bash
   echo "example.com" | metabigor cert --debug
   ```

2. **Verbose output is now default**: Progress messages show by default
   ```bash
   # Normal mode (shows progress)
   echo "example.com" | metabigor cert

   # Silent mode (errors only, useful for piping)
   echo "example.com" | metabigor cert -q
   ```

3. **Test with Single Input**: Easier to debug than batch
   ```bash
   # Good for debugging
   echo "example.com" | metabigor cert

   # Harder to debug
   cat 1000-domains.txt | metabigor cert
   ```

4. **Check Source Manually**: Verify queries work in browser
   - Visit grep.app with your query
   - Check crt.sh HTML structure
   - Test API endpoints with curl

5. **Monitor Rate Limits**: Watch for 429 responses
   ```bash
   # Look for rate limit errors in debug output
   echo "query" | metabigor github --debug
   ```

6. **Use JSON for Parsing**: Easier to debug structure
   ```bash
   echo "example.com" | metabigor cert --json | jq .
   ```

## Gotchas & Known Issues

### crt.sh

**Issue**: HTML structure changes periodically
**Solution**: `ParseSnippet()` has fallback to `CleanSnippet()`

**Issue**: Sometimes returns 503
**Solution**: Retry logic in HTTP client handles this

### grep.app

**Issue**: Rate limiting without clear documentation
**Solution**: 5 second delay between pages, `--page` limit

**Issue**: HTML snippet parsing can fail
**Solution**: Fallback to regex-based extraction

### ASN Database

**Issue**: Must run `update` before first use
**Solution**: Embedded ZIP provides fallback, auto-extracts on first run

**Issue**: Database can become outdated
**Solution**: Run `metabigor update` monthly

### Chrome Automation

**Issue**: Requires Chrome/Chromium in PATH
**Solution**: Falls back to HTTP if Chrome unavailable

**Issue**: Slow compared to HTTP
**Solution**: Only used when necessary (JS-heavy sites)

### Wildcard Certificates

**Issue**: `*.example.com` may appear in results
**Solution**: Use `--clean` flag to strip `*.` prefix
**Alternatively**: Use `--wildcard` to show only wildcards

## Recent Changes & Improvements

### CLI Flag and Output Improvements (2026-02-15)

**BREAKING CHANGES** to input flags and verbose behavior for better UX:

#### 1. Input Flags Renamed

- `--input` now has shorthand `-i` (previously no shorthand)
- `-f` (inputFile) changed to `-I` (uppercase I)
- **Old**: `metabigor net -f domains.txt`
- **New**: `metabigor net -I domains.txt`

**Rationale**: `-i` is conventional for input flags (matches curl, wget, etc.), `-I` follows convention for file variants.

#### 2. Verbose Behavior Changed

- **Removed** `--verbose/-v` flag entirely
- **Verbose output is now DEFAULT** (all progress messages show by default)
- **Added** `--silent/-q` flag to hide progress messages (errors only)
- **Old**: `metabigor net --input AS13335 -v` (needed -v to see progress)
- **New**: `metabigor net -i AS13335` (verbose by default)
- **Silent**: `metabigor net -i AS13335 -q` (errors only)

**Rationale**: Better UX for new users (informative by default), better control for scripting/piping scenarios.

#### 3. Log Prefix Changes

- `[*]` changed to `[info]` for informational messages (clearer naming)
- All other prefixes unchanged: `[+]`, `[-]`, `[!]`, `[verbose]`, `[debug]`

#### Migration Guide for Users

Update scripts using the old flags:

```bash
# Old scripts
metabigor net -f domains.txt -v
metabigor cert --input example.com --verbose

# New scripts (verbose by default, just change -f to -I and remove -v)
metabigor net -I domains.txt
metabigor cert -i example.com

# New silent mode (if you don't want progress)
metabigor net -I domains.txt -q
metabigor cert -i example.com -q
```

**Files Modified**:
- `internal/options/options.go` - Changed Verbose to Silent field
- `internal/output/logger.go` - Implemented silent mode, changed `[*]` to `[info]`
- `internal/cli/root.go` - Updated flag definitions
- All 8 command files - Updated SetupLogger calls
- `internal/cli/helptext.go` - Updated all examples
- `README.md` - Updated flag references
- `test/run-e2e.sh` - Updated tests

---

### Country Database Integration (2026-02-15)

**Added**:
- New `internal/countrydb/` package
- IP-to-country lookup with 2M+ records
- Auto-download and embedded fallback
- Binary search for fast lookups

**Impact**:
- `net` command now shows country info
- Format: `AS | CIDR | Org | Continent (Country)`
- Example: `AS13335 | 1.1.1.0/24 | Cloudflare | OC (AU)`

**Files**:
- `internal/countrydb/db.go` - Database loading
- `internal/countrydb/download.go` - Downloader
- `public/ip-to-country.csv.zip` - Embedded data

---

### Enhanced Certificate Transparency (2026-02-15)

**Added**:
- Rich certificate metadata extraction
- Domain grouping with deduplication
- `--simple` flag for backward compatibility

**Before**:
```
*.example.com
example.com
api.example.com
```

**After (default)**:
```
Domain: api.example.com
  Cert IDs (3): 12345, 67890, 11111
  Not Before: 2024-01-01
  Not After: 2025-12-31
  Issuers: Let's Encrypt
```

**Changes**:
- Enhanced `CertEntry` struct (7 fields instead of 3)
- Added `DomainGroup` struct for aggregation
- `GroupByDomain()` function for deduplication
- Numeric validation for cert IDs (filters malformed rows)

---

### GitHub Search Improvements (2026-02-15)

**Changes**:
- Log prefixes: `[v]` → `[verbose]`, `[d]` → `[debug]`
- User-Agent: Chrome 80 → Chrome 119
- Added modern `Sec-Ch-Ua` headers
- Rate limit: 2s → 5s delay between pages
- Added `--page` flag for pagination control

**Improvements**:
- Better bot detection avoidance
- More respectful rate limiting
- User control over resource usage
- Improved code snippet formatting

---

### Testing Infrastructure (2026-02-15)

**Added**:
- E2E test framework (`test/run-e2e.sh`)
- 29 automated test cases
- golangci-lint configuration
- GoReleaser setup with UPX

**Coverage**:
- All commands
- All input methods
- All output modes
- Global and command flags

**Run**: `make e2e`

---

### Database Management (2026-02-15)

**Improvements**:
- `make update` now downloads both databases
- Embedded ZIP fallbacks for offline use
- Graceful degradation if databases missing
- Atomic file replacement during updates

## Questions?

When uncertain about implementation:

1. **Check similar commands**:
   - `cert` and `github` both query external APIs
   - `net` and `ipc` both use ASN database
   - `related` combines multiple sources

2. **Review core packages**:
   - `runner` for input/parallel execution patterns
   - `output` for logging standards
   - `httpclient` for HTTP best practices

3. **Read command Cobra definitions**:
   - Check flags and their usage
   - Review example commands
   - Understand expected inputs/outputs

4. **Test thoroughly**:
   - Use `-v` and `--debug` flags
   - Try all input methods
   - Verify both text and JSON output
   - Test edge cases (empty input, errors, etc.)

5. **Follow the patterns**:
   - Consistent error handling
   - Thread-safe output writing
   - Respectful rate limiting
   - Graceful degradation

## Philosophy Summary

> metabigor = metadata + bigor (bigger)

**Make OSINT bigger by aggregating metadata from multiple free sources.**

Keep it:
- **Simple**: Unix philosophy - do one thing well, compose with pipes
- **Free**: No API keys, no accounts, no rate limit headaches
- **Respectful**: Don't abuse free services (delays, concurrency limits)
- **Reliable**: Retry logic, fallbacks, graceful degradation
- **Hackable**: Easy to extend, clear code, good examples

---

*This guide is maintained for AI agents working on metabigor. Keep it updated as the codebase evolves.*

**Last Updated**: 2026-02-15
**Version**: 2.1.0
**Maintainer**: j3ssie
