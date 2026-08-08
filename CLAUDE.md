# Metabigor - CLAUDE.md

**Project**: Metabigor - OSINT intelligence tool without API key hassle
**Version**: v2.2.0
**Language**: Go 1.24.0
**Author**: [@j3ssie](https://twitter.com/j3ssie)
**License**: MIT

## Project Overview

Metabigor is a command-line OSINT (Open Source Intelligence) tool designed to perform network reconnaissance and intelligence gathering without requiring API keys. It's part of the Osmedeus Engine ecosystem and focuses on seven core capabilities:

1. **Network Discovery** (`net`) - Find IP ranges (CIDRs) from ASN, organization, domain, or IP
2. **Certificate Transparency** (`cert`) - Discover subdomains via crt.sh certificate logs
3. **IP Enrichment** (`ip`) - Get port/service/vulnerability data via Shodan InternetDB (free)
4. **GitHub Code Search** (`github`) - Find secrets and credentials in public repos via grep.app
5. **IP Clustering** (`cluster`) - Group IPs by ASN for infrastructure mapping
6. **Related Domains** (`related`) - Discover related domains via cert logs, WHOIS, analytics
7. **CDN/WAF Detection** (`cdn`) - Identify if IPs are behind CDN or WAF providers
8. **URL Collection** (`url`) - Collect known URLs from web archives and indexes, including
   endpoints mined out of GhostArchive WARC files

It also ships a maintenance command, **Skills** (`skills`), that lists, prints, and installs the
embedded [agentskills.io](https://agentskills.io) skill bundle teaching a coding agent how to drive
the CLI.

## Architecture

### Directory Structure

```
metabigor/
├── cmd/metabigor/          # Main application entry point
├── internal/               # Internal packages (not importable by external projects)
│   ├── asndb/             # ASN database management (local CSV lookups)
│   ├── cert/              # Certificate transparency search (crt.sh)
│   ├── cli/               # Cobra CLI commands and subcommands
│   ├── core/              # Core constants and configuration
│   ├── countrydb/         # Country database management
│   ├── gitsearch/         # GitHub code search via grep.app
│   ├── httpclient/        # HTTP client utilities (retryable, Chrome-based)
│   ├── ipinfo/            # IP enrichment (Shodan InternetDB) and clustering
│   ├── netdiscovery/      # Network discovery (static DB + dynamic sources)
│   ├── options/           # Global CLI options and configuration
│   ├── output/            # Output formatting (JSON, CSV, flat) and logging
│   ├── related/           # Related domain discovery (crt, WHOIS, analytics)
│   ├── runner/            # Core execution runner and input processing
│   ├── skill/             # SKILL.md frontmatter parsing for the skills command
│   └── urlsource/         # URL collection sources (Wayback, Common Crawl, ...)
├── public/                # Embedded assets: ASN/country databases + skills/ bundles
│   └── skills/            # Coding-agent skill bundles (SKILL.md + references/)
├── build/                 # Release tooling (not compiled into the binary)
│   ├── npm/               # @j3ssie/metabigor packaging: build.mjs + launcher/
│   └── scripts/           # bump-version.sh, github-release.sh
└── test/                  # End-to-end test scripts
```

### Key Design Patterns

- **Internal-only packages**: All logic is in `internal/` to prevent external imports
- **Cobra CLI framework**: Each command is a separate file in `internal/cli/`
- **Runner pattern**: `internal/runner` processes input (stdin, flags, files) and routes to handlers
- **Output abstraction**: `internal/output` provides consistent formatting across all commands
- **Embedded databases**: `public/` contains CSV databases embedded via `//go:embed` for offline use

### Data Flow

1. User input → CLI command (`internal/cli/`)
2. CLI initializes runner → `internal/runner/runner.go`
3. Runner processes input sources (stdin, `-i`, `-I`, `--input`)
4. Runner calls module-specific handler (`cert`, `net`, `ip`, etc.)
5. Handler queries data sources (local DB, APIs, web scraping)
6. Results formatted via `internal/output/writer.go`
7. Output to stdout or file (`-o` flag)

## Key Technical Details

### Database Management

- **ASN Database**: `~/.metabigor/ip-asn-combined.csv` (2M+ entries)
  - Downloaded via `metabigor update`
  - Source: https://github.com/iplocate/ip-address-databases
  - Used by `net` and `cluster` commands for offline ASN lookups

- **Country Database**: `~/.metabigor/ip-country-combined.csv`
  - Used for geolocation enrichment
  - Same source as ASN database

### HTTP Client Strategy

- **Retryable HTTP**: Uses `hashicorp/go-retryablehttp` for resilient API calls
- **Chrome CDP**: Uses `chromedp` for JavaScript-heavy sites (grep.app, builtwith.com)
- **Rate limiting**: Concurrent execution controlled via `-c` flag (default: 10).
  `github` ignores it and runs sequentially, to stay inside grep.app's limit.

### Data Sources

- **crt.sh**: Certificate transparency logs (cert, related commands)
- **Shodan InternetDB**: Free IP enrichment API (no key required)
- **grep.app**: GitHub code search
- **bgp.he.net**: Live BGP routing data (dynamic network discovery)
- **viewdns.info**: Reverse WHOIS lookups
- **builtwith.com**: Analytics tracking correlation (Google Analytics, GTM)
- **projectdiscovery/cdncheck**: CDN/WAF detection library
- **web.archive.org**: Wayback Machine CDX index (`url` command)
- **index.commoncrawl.org**: Common Crawl CDX indexes (`url`); index list cached 30 days in `~/.metabigor`
- **otx.alienvault.com**: AlienVault OTX URL lists (`url`)
- **urlscan.io**: Scan history search (`url`); `URLSCAN_API_KEY` optional
- **ghostarchive.org**: Archived pages plus WARC mining for sub-request URLs (`url`)
- **virustotal.com**: v3 domain relationships (`url`); requires `VT_API_KEY`
- **intelx.io**: Phonebook search (`url`); requires `INTELX_API_KEY`

### API Keys

The tool remains key-free by default. The `url` command reads optional keys from the environment
only (`VT_API_KEY`/`VIRUSTOTAL_API_KEY`, `INTELX_API_KEY`, `URLSCAN_API_KEY`) — never from a config
file or flag. Sources requiring a key are skipped silently when it is absent, unless the user named
that source explicitly, which is an error.

## Development Guidelines

### Building

```bash
make build              # Build and install to $GOPATH/bin
make install            # Install directly via go install
make test               # Run unit tests with race detection
make e2e                # Run end-to-end tests
make build-all          # Cross-compile for all platforms
```

### Code Style

- **No external imports**: Keep all logic in `internal/`
- **Error handling**: Always check errors; use `output.Error()` for user-facing messages
- **Logging**: Use `output` package methods (`Info`, `Good`, `Warn`, `Error`, `Verbose`, `Debug`).
  Results go to stdout; every log line goes to stderr.
- **Log levels**: `Verbose` requires `-v`; `Debug` requires `--debug`; `-q` silences all but errors
- **Input flexibility**: Always support stdin, `-i`, `-I` file, and `--input` flag
- **Output formats**: Results are `output.Record` implementations (`Text`, `Flat`, `CSV`); the
  writer picks the rendering from `-f/--format`. Never format results in the CLI layer.
- **Exit codes**: Commands use `RunE` and return errors; `Execute` maps them to exit 1

### Version Management

- Version is defined in `internal/core/constants.go`
- Build metadata (commit, date) injected via ldflags in Makefile
- Use semantic versioning (vMAJOR.MINOR.PATCH)

### Testing

- **Unit tests**: Place in same package as code (`*_test.go`)
- **E2E tests**: Shell scripts in `test/` directory
- **Test commands**: `make test` (unit), `make e2e` (end-to-end)

## Common Workflows

### Adding a New Command

1. Create new CLI command file in `internal/cli/` (e.g., `internal/cli/newcmd.go`)
2. Implement Cobra command with flags and input handling
3. Create handler package in `internal/` (e.g., `internal/newfeature/`)
4. Add handler logic and data source integration
5. Give the result type `Text() []string`, `Flat() []string`, and `CSV() ([]string, [][]string)`
   so it satisfies `output.Record` and renders in all four formats for free
6. Start the command with `setup(cmd)` for input reading and writer creation
7. Put `Long` and `Example` (via the `examples()` helper) in the command file itself
8. Register the command in its `init()` with a `GroupID` and an `Annotations["sample"]` target
9. Add examples to README.md

### Updating Embedded Databases

```bash
make update-ip-data    # Downloads latest ASN and country databases to public/
```

Then rebuild to embed the new databases:
```bash
make build
```

### Release Process

`VERSION` in `internal/core/constants.go` is the single source of truth. It drives the Makefile
ldflags, the goreleaser build, the npm package version, and the git tag — never bump any of them
by hand.

```bash
make bump-version                  # v2.2.0 -> v2.2.1 in constants.go
                                   #   PART=minor|major|pre|release, LABEL=beta, SET=v2.3.0
git commit -am "Release v2.2.1"    # goreleaser refuses a dirty worktree
make npm-publish                   # -> npm  (needs NPM_TOKEN; DRY_RUN=1 to preview)
make github-release                # -> tag + GitHub release (needs GITHUB_TOKEN)
```

`npm-publish` rebuilds the binaries via `make snapshot` whenever `dist/` is empty or was built
for a different version, so a bump can never ship a stale binary under a fresh npm version
(npm versions are immutable — a bad publish cannot be replaced, only deprecated).

Update README.md with any new features before bumping.

## Important Context for AI Assistants

### When Making Changes

- **Input handling**: ALL commands must support stdin, `-i`, `-I`, and `--input` — use `setup()`
- **Output modes**: One `-f/--format` flag covers text, flat, json, and csv. Do not add
  per-command format flags; implement `output.Record` instead
- **Log levels**: Progress is quiet by default; keep step-by-step detail in `output.Verbose`
- **Error handling**: Return errors from `RunE`; use `output.Error()` for non-fatal problems
- **Conflicting flags**: Declare them with `cmd.MarkFlagsMutuallyExclusive` rather than
  resolving conflicts silently in code
- **Concurrency**: Respect `-c` flag for concurrent operations

### Common Pitfalls

- **Don't break stdin piping**: Always test with `echo "input" | metabigor cmd`
- **Don't read stdin when targets were already given**: it blocks forever in scripts and CI
- **Don't hardcode paths**: Use `options.DataDir()` for database paths
- **Don't skip retries**: Use retryable HTTP client for external API calls
- **Don't assume online**: Commands should work offline when using local DB
- **Don't ignore cleanup**: Close HTTP clients, Chrome instances, file handles

### Security Considerations

- **No credentials in code**: This tool specifically avoids API keys
- **Input validation**: Sanitize user input before passing to external commands
- **Safe web scraping**: Respect rate limits, use retries, handle timeouts
- **No destructive operations**: Tool is read-only OSINT, never modifies targets

### Performance Tips

- **Use goroutines**: For bulk operations (IP scanning, subdomain enumeration)
- **Batch processing**: Process inputs in chunks when possible
- **Database caching**: Load ASN/country DB once, reuse across lookups
- **HTTP connection pooling**: Reuse HTTP clients across requests

## Dependencies

### Critical Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/projectdiscovery/cdncheck` - CDN/WAF detection
- `github.com/projectdiscovery/mapcidr` - CIDR manipulation
- `github.com/chromedp/chromedp` - Headless Chrome for JS-heavy sites
- `github.com/hashicorp/go-retryablehttp` - Resilient HTTP client
- `github.com/PuerkitoBio/goquery` - HTML parsing
- `github.com/charmbracelet/glamour` - Markdown rendering in terminal

## Testing Checklist

Before committing changes:

- [ ] Run `make test` - all unit tests pass
- [ ] Run `make fmt` - code is formatted
- [ ] Run `make vet` - no vet warnings
- [ ] Test stdin input: `echo "input" | metabigor <cmd>`
- [ ] Test file input: `metabigor <cmd> -I file.txt`
- [ ] Test output file: `metabigor <cmd> -o output.txt`
- [ ] Test every format: `metabigor <cmd> -f text|flat|json|csv`
- [ ] Test quiet mode: `metabigor <cmd> -q` (results only, no logs)
- [ ] Confirm failures exit non-zero: `metabigor <cmd>; echo $?`
- [ ] Run `make e2e` - the CLI contract suite passes
- [ ] Update README.md if adding features, including the upgrade table for renames

## Useful Commands

```bash
# Build and test
make build              # Build binary to bin/metabigor
make test               # Run tests
make e2e                # End-to-end tests
make lint               # Run golangci-lint

# Database management
make update-ip-data     # Update embedded ASN/country databases
metabigor update        # Download databases at runtime (user command)

# Release
make bump-version       # Bump VERSION in internal/core/constants.go
make snapshot           # Test goreleaser build (cross-platform binaries in dist/)
make npm-pack           # Stage npm packages + .tgz tarballs in build/dist-npm/
make npm-publish        # Publish @j3ssie/metabigor (needs NPM_TOKEN; DRY_RUN=1 previews)
make github-release     # Tag + GitHub release via goreleaser (needs GITHUB_TOKEN)

# Development
go run ./cmd/metabigor  # Run without building
go mod tidy             # Clean up dependencies
```

## Philosophy

Metabigor's core philosophy is **API-free OSINT**. When adding features:

1. **Prefer free data sources** over API-based services
2. **Respect rate limits** and implement retries
3. **Work offline when possible** (local databases)
4. **Pipeline-friendly** (stdin/stdout, clean output)
5. **Zero configuration** (no config files, no setup beyond `metabigor update`)

## Support and Resources

- **GitHub Issues**: https://github.com/j3ssie/metabigor/issues
- **Documentation**: README.md and `metabigor <cmd> --help`
- **Part of**: Osmedeus Engine (@OsmedeusEngine)
- **Author**: @j3ssie
