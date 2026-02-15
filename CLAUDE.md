# Metabigor - CLAUDE.md

**Project**: Metabigor - OSINT intelligence tool without API key hassle
**Version**: v2.1.0
**Language**: Go 1.24.0
**Author**: [@j3ssie](https://twitter.com/j3ssie)
**License**: MIT

## Project Overview

Metabigor is a command-line OSINT (Open Source Intelligence) tool designed to perform network reconnaissance and intelligence gathering without requiring API keys. It's part of the Osmedeus Engine ecosystem and focuses on seven core capabilities:

1. **Network Discovery** (`net`) - Find IP ranges (CIDRs) from ASN, organization, domain, or IP
2. **Certificate Transparency** (`cert`) - Discover subdomains via crt.sh certificate logs
3. **IP Enrichment** (`ip`) - Get port/service/vulnerability data via Shodan InternetDB (free)
4. **GitHub Code Search** (`github`) - Find secrets and credentials in public repos via grep.app
5. **IP Clustering** (`ipc`) - Group IPs by ASN for infrastructure mapping
6. **Related Domains** (`related`) - Discover related domains via cert logs, WHOIS, analytics
7. **CDN/WAF Detection** (`cdn`) - Identify if IPs are behind CDN or WAF providers

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
│   └── runner/            # Core execution runner and input processing
├── public/                # Embedded databases (ASN, country CSV files)
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
  - Used by `net` and `ipc` commands for offline ASN lookups

- **Country Database**: `~/.metabigor/ip-country-combined.csv`
  - Used for geolocation enrichment
  - Same source as ASN database

### HTTP Client Strategy

- **Retryable HTTP**: Uses `hashicorp/go-retryablehttp` for resilient API calls
- **Chrome CDP**: Uses `chromedp` for JavaScript-heavy sites (grep.app, builtwith.com)
- **Rate limiting**: Concurrent execution controlled via `-c` flag (default: 10)

### Data Sources

- **crt.sh**: Certificate transparency logs (cert, related commands)
- **Shodan InternetDB**: Free IP enrichment API (no key required)
- **grep.app**: GitHub code search
- **bgp.he.net**: Live BGP routing data (dynamic network discovery)
- **viewdns.info**: Reverse WHOIS lookups
- **builtwith.com**: Analytics tracking correlation (Google Analytics, GTM)
- **projectdiscovery/cdncheck**: CDN/WAF detection library

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
- **Logging**: Use `output` package methods (`Info`, `Error`, `Debug`, `Success`)
- **Silent mode**: Respect `-q/--quiet` flag - no progress messages, errors only
- **Input flexibility**: Always support stdin, `-i`, `-I` file, and `--input` flag

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
5. Use `internal/output` for consistent output formatting
6. Add help text to `internal/cli/helptext.go`
7. Register command in `internal/cli/root.go`
8. Add examples to README.md

### Updating Embedded Databases

```bash
make update    # Downloads latest ASN and country databases to public/
```

Then rebuild to embed the new databases:
```bash
make build
```

### Release Process

1. Update version in `internal/core/constants.go`
2. Update README.md with new features
3. Commit changes: `git commit -m "Release vX.Y.Z"`
4. Tag release: `git tag vX.Y.Z`
5. Push with tags: `git push origin main --tags`
6. Run `make release` (requires goreleaser and GITHUB_TOKEN)

## Important Context for AI Assistants

### When Making Changes

- **Input handling**: ALL commands must support stdin, `-i`, `-I`, and `--input`
- **Output modes**: Consider JSON (`--json`), CSV (`--csv`), and flat formats
- **Silent mode**: Progress messages should respect `-q/--quiet` flag
- **Error handling**: Use `output.Error()` not `fmt.Println()` for errors
- **Concurrency**: Respect `-c` flag for concurrent operations

### Common Pitfalls

- **Don't break stdin piping**: Always test with `echo "input" | metabigor cmd`
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
- [ ] Test JSON output: `metabigor <cmd> --json`
- [ ] Test silent mode: `metabigor <cmd> -q`
- [ ] Update README.md if adding features
- [ ] Update help text in `internal/cli/helptext.go`

## Useful Commands

```bash
# Build and test
make build              # Build binary to bin/metabigor
make test               # Run tests
make e2e                # End-to-end tests
make lint               # Run golangci-lint

# Database management
make update             # Update embedded ASN/country databases
metabigor update        # Download databases at runtime (user command)

# Release
make snapshot           # Test goreleaser build
make release            # Create GitHub release (needs tag + GITHUB_TOKEN)

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
