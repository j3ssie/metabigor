package cli

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

// hasNoColorFlag checks if --no-color is present in os.Args before Cobra parses it.
func hasNoColorFlag() bool {
	for _, arg := range os.Args {
		if arg == "--no-color" {
			return true
		}
	}
	return false
}

// Color helper functions (initialized in init based on --no-color and NO_COLOR)
var (
	commentColor    func(a ...interface{}) string
	commandColor    func(a ...interface{}) string
	subcommandColor func(a ...interface{}) string
	flagColor       func(a ...interface{}) string
	valueColor      func(a ...interface{}) string
	pipeColor       func(a ...interface{}) string
)

// Example text variables (built in init after colors are initialized)
var (
	rootExample    string
	certExample    string
	netExample     string
	ipExample      string
	ipcExample     string
	githubExample  string
	relatedExample string
	cdnExample     string
	updateExample  string
	versionExample string
)

func init() {
	// Detect if colors should be disabled
	if hasNoColorFlag() || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}

	// Initialize color functions
	commentColor = color.New(color.FgHiBlack).SprintFunc()
	commandColor = color.New(color.FgCyan, color.Bold).SprintFunc()
	subcommandColor = color.New(color.FgGreen).SprintFunc()
	flagColor = color.New(color.FgYellow).SprintFunc()
	valueColor = color.New(color.FgMagenta).SprintFunc()
	pipeColor = color.New(color.FgHiBlack).SprintFunc()

	// Build example text now that colors are initialized
	buildExamples()

	// Set example fields on commands (must happen after buildExamples())
	applyExamplesToCommands()
}

func buildExamples() {
	rootExample = buildRootExample()
	certExample = buildCertExample()
	netExample = buildNetExample()
	ipExample = buildIPExample()
	ipcExample = buildIPCExample()
	githubExample = buildGithubExample()
	relatedExample = buildRelatedExample()
	cdnExample = buildCDNExample()
	updateExample = buildUpdateExample()
	versionExample = buildVersionExample()
}

func applyExamplesToCommands() {
	// Apply examples to commands after they've been built
	rootCmd.Example = rootExample
	certCmd.Example = certExample
	netCmd.Example = netExample
	ipCmd.Example = ipExample
	ipcCmd.Example = ipcExample
	githubCmd.Example = githubExample
	relatedCmd.Example = relatedExample
	cdnCmd.Example = cdnExample
	updateCmd.Example = updateExample
	versionCmd.Example = versionExample
}

// Helper functions for building colored command examples

// cmd builds a command line with the main command colored
func cmd(parts ...string) string {
	return commandColor(strings.Join(parts, " "))
}

// subcmd builds a command with subcommand colored
func subcmd(sub string, parts ...string) string {
	all := []string{commandColor("metabigor"), subcommandColor(sub)}
	all = append(all, parts...)
	return strings.Join(all, " ")
}

// echo builds an echo command with colored value
func echo(v string) string {
	return cmd("echo") + " " + valueColor(`"`+v+`"`)
}

// pipe joins parts with the pipe character
func pipe(parts ...string) string {
	return strings.Join(parts, " "+pipeColor("|")+" ")
}

// exampleBlock formats a list of comment + command pairs
// Takes alternating comment, command, comment, command, ...
func exampleBlock(lines ...string) string {
	var result []string
	for i := 0; i < len(lines); i += 2 {
		comment := lines[i]
		var command string
		if i+1 < len(lines) {
			command = lines[i+1]
		}
		result = append(result, "  "+commentColor("# "+comment))
		if command != "" {
			result = append(result, "  "+command)
			if i+2 < len(lines) {
				result = append(result, "") // blank line between examples
			}
		}
	}
	return strings.Join(result, "\n")
}

// Help text constants (Long descriptions - plain text, no colors)

const rootLong = `metabigor — OSINT power without API key hassle

An OSINT tool for network discovery, domain relationships, certificate searching,
IP enrichment, and code search without requiring API keys.`

func buildRootExample() string {
	return exampleBlock(
		"Update ASN database before first use",
		cmd("metabigor update"),

		"Network discovery from ASN",
		pipe(echo("AS13335"), subcmd("net")),

		"Certificate transparency search",
		pipe(echo("hackerone.com"), subcmd("cert", flagColor("--clean"))),

		"IP enrichment with Shodan InternetDB",
		pipe(echo("1.1.1.0/28"), subcmd("ip", flagColor("--flat"))),

		"Search GitHub code",
		pipe(echo("example.com"), subcmd("github")),

		"Find related domains",
		pipe(echo("target.com"), subcmd("related", flagColor("-s"), valueColor("all"))),

		"Cluster IPs by ASN",
		pipe("cat "+valueColor("ips.txt"), subcmd("ipc", flagColor("--json"))),

		"Pipeline: ASN -> CIDRs -> IP scan",
		pipe(echo("AS13335"), subcmd("net"), subcmd("ip", flagColor("--flat"))),
	)
}

const certLong = `Query crt.sh for domains and certificates found in Certificate Transparency logs.
Search by domain name or organization name to discover subdomains and related certificates.

Default output shows domains grouped with their associated certificate IDs and metadata.
Use --simple for a plain list of domain names (old behavior).`

func buildCertExample() string {
	return exampleBlock(
		"Find domains with certificate details (default grouped view)",
		pipe(echo("hackerone.com"), subcmd("cert")),

		"Alternative input method (long form)",
		subcmd("cert", flagColor("--input"), valueColor(`"tesla.com"`)),

		"Alternative input method (short form)",
		subcmd("cert", flagColor("-i"), valueColor(`"tesla.com"`)),

		"Simple mode: just domain names (old behavior)",
		pipe(echo("hackerone.com"), subcmd("cert", flagColor("--simple"))),

		"Save simple output to file",
		subcmd("cert", flagColor("-i"), valueColor(`"tesla.com"`), flagColor("--simple"), flagColor("-o"), valueColor("domains.txt")),

		"Search by organization name",
		pipe(echo("HackerOne Inc"), subcmd("cert")),

		"Organization search with input flag",
		subcmd("cert", flagColor("-i"), valueColor(`"Tesla Motors"`)),

		"Strip wildcard prefixes (*.) from results",
		pipe(echo("example.com"), subcmd("cert", flagColor("--clean"))),

		"Clean mode with input flag",
		subcmd("cert", flagColor("-i"), valueColor(`"spacex.com"`), flagColor("--clean")),

		"Show only wildcard entries",
		pipe(echo("example.com"), subcmd("cert", flagColor("--wildcard"))),

		"JSON output with full certificate details (grouped)",
		pipe(echo("example.com"), subcmd("cert", flagColor("--json"))),

		"JSON to file",
		subcmd("cert", flagColor("-i"), valueColor(`"apple.com"`), flagColor("--json"), flagColor("-o"), valueColor("certs.json")),

		"Save grouped output to file",
		pipe(echo("example.com"), subcmd("cert", flagColor("-o"), valueColor("cert-report.txt"))),

		"Clean mode with file output",
		subcmd("cert", flagColor("-i"), valueColor(`"microsoft.com"`), flagColor("--clean"), flagColor("-o"), valueColor("cert-report.txt")),

		"Batch search from file",
		subcmd("cert", flagColor("-I"), valueColor("domains.txt"), flagColor("--clean"), flagColor("-o"), valueColor("all-cert-info.txt")),

		"Batch with JSON output",
		subcmd("cert", flagColor("-I"), valueColor("companies.txt"), flagColor("--json"), flagColor("-o"), valueColor("all-certs.json")),

		"Silent mode (errors only, no progress)",
		subcmd("cert", flagColor("-i"), valueColor(`"example.com"`), flagColor("-q")),
	)
}

const netLong = `Discover CIDRs and network ranges for an ASN, IP, domain, or organization.
Auto-detects input type unless overridden with --asn, --ip, --domain, or --org flags.

Uses local ASN database by default (fast). Use --dynamic for live online sources.`

func buildNetExample() string {
	return exampleBlock(
	"Find CIDRs for an ASN",
	pipe(echo("AS13335"), subcmd("net")),

	"Alternative input method (long form)",
	subcmd("net", flagColor("--input"), valueColor("AS13335")),

	"Alternative input method (short form)",
	subcmd("net", flagColor("-i"), valueColor("AS13335")),

	"Lookup which ASN owns an IP",
	pipe(echo("1.1.1.1"), subcmd("net", flagColor("--ip"))),

	"Find CIDRs associated with a domain",
	pipe(echo("cloudflare.com"), subcmd("net", flagColor("--domain"))),

	"Search by organization name",
	pipe(echo("Cloudflare"), subcmd("net", flagColor("--org"))),

	"Organization with input flag",
	subcmd("net", flagColor("-i"), valueColor(`"Tesla"`), flagColor("--org")),

	"Use live online sources instead of local DB",
	pipe(echo("AS13335"), subcmd("net", flagColor("--dynamic"))),

	"Dynamic with JSON output",
	subcmd("net", flagColor("--dynamic"), flagColor("-i"), valueColor(`"SpaceX"`), flagColor("--json")),

	"JSON output for automation",
	pipe(echo("Tesla"), subcmd("net", flagColor("--org"), flagColor("--json"))),

	"IP lookup with JSON to file",
	subcmd("net", flagColor("-i"), valueColor(`"1.1.1.1"`), flagColor("--ip"), flagColor("--json"), flagColor("-o"), valueColor("results.json")),

	"Batch input from file",
	subcmd("net", flagColor("-I"), valueColor("asn-list.txt"), flagColor("-o"), valueColor("results.txt")),

	"Dynamic lookup",
	subcmd("net", flagColor("-i"), valueColor(`"Microsoft"`), flagColor("--dynamic")),

	"Force input type with override flags",
	pipe(echo("13335"), subcmd("net", flagColor("--asn"))),

	"Company name with dynamic sources",
	subcmd("net", flagColor("-i"), valueColor(`"Apple Inc"`), flagColor("--org"), flagColor("--dynamic")),

	"Silent mode (errors only, no progress)",
	subcmd("net", flagColor("-i"), valueColor("AS13335"), flagColor("-q")),
	)
}

const ipLong = `Query Shodan's free InternetDB API for IP enrichment data including open ports,
hostnames, vulnerabilities, and tags. Supports single IPs and CIDR ranges (auto-expanded).`

func buildIPExample() string {
	return exampleBlock(
	"Lookup a single IP",
	pipe(echo("1.1.1.1"), subcmd("ip")),

	"Alternative input (long form)",
	subcmd("ip", flagColor("--input"), valueColor(`"8.8.8.8"`)),

	"Alternative input (short form)",
	subcmd("ip", flagColor("-i"), valueColor(`"8.8.8.8"`)),

	"Scan a CIDR range (auto-expands)",
	pipe(echo("1.1.1.0/28"), subcmd("ip")),

	"CIDR with input flag",
	subcmd("ip", flagColor("-i"), valueColor(`"8.8.8.0/24"`)),

	"Flat IP:PORT output (useful for piping to other tools)",
	pipe(echo("1.1.1.0/28"), subcmd("ip", flagColor("--flat"))),

	"Flat output to file",
	subcmd("ip", flagColor("-i"), valueColor(`"8.8.8.0/29"`), flagColor("--flat"), flagColor("-o"), valueColor("ports.txt")),

	"CSV output for spreadsheet analysis",
	pipe(echo("1.1.1.0/28"), subcmd("ip", flagColor("--csv"))),

	"CSV to file",
	subcmd("ip", flagColor("-i"), valueColor(`"8.8.8.8"`), flagColor("--csv"), flagColor("-o"), valueColor("results.csv")),

	"JSON output for automation",
	pipe(echo("1.1.1.1"), subcmd("ip", flagColor("--json"))),

	"JSON to file",
	subcmd("ip", flagColor("-i"), valueColor(`"8.8.8.8"`), flagColor("--json"), flagColor("-o"), valueColor("ip-info.json")),

	"Bulk scan from file with higher concurrency",
	subcmd("ip", flagColor("-I"), valueColor("ips.txt"), flagColor("-c"), valueColor("20"), flagColor("--flat"), flagColor("-o"), valueColor("open-ports.txt")),

	"Batch CIDR scan with JSON",
	subcmd("ip", flagColor("-I"), valueColor("cidrs.txt"), flagColor("-c"), valueColor("15"), flagColor("--json"), flagColor("-o"), valueColor("results.json")),

	"Silent mode (errors only)",
	subcmd("ip", flagColor("-i"), valueColor(`"1.1.1.1"`), flagColor("-q")),

	"Pipeline: Get CIDRs then scan IPs",
	pipe(echo("AS13335"), subcmd("net"), subcmd("ip", flagColor("--flat"))),
	)
}

const ipcLong = `Group a list of IPs by ASN using the local database. Useful for understanding
the infrastructure behind a set of IPs and identifying cloud providers and hosting companies.`

func buildIPCExample() string {
	return exampleBlock(
	"Cluster IPs from stdin",
	pipe("cat "+valueColor("ips.txt"), subcmd("ipc")),

	"Single IP",
	pipe(echo("1.1.1.1"), subcmd("ipc")),

	"Alternative input (long form)",
	subcmd("ipc", flagColor("--input"), valueColor(`"8.8.8.8"`)),

	"Alternative input (short form)",
	subcmd("ipc", flagColor("-i"), valueColor(`"8.8.8.8"`)),

	"JSON output with full ASN details",
	pipe("cat "+valueColor("ips.txt"), subcmd("ipc", flagColor("--json"))),

	"Input flag with JSON",
	subcmd("ipc", flagColor("-i"), valueColor(`"8.8.8.8"`), flagColor("--json")),

	"From file",
	subcmd("ipc", flagColor("-I"), valueColor("ips.txt"), flagColor("-o"), valueColor("clusters.txt")),

	"File input with JSON output",
	subcmd("ipc", flagColor("-I"), valueColor("ips.txt"), flagColor("--json"), flagColor("-o"), valueColor("clusters.json")),

	"Multiple IPs",
	"echo -e "+valueColor(`"1.1.1.1\n8.8.8.8\n9.9.9.9"`)+pipeColor(" |")+" "+subcmd("ipc"),

	"Single IP to file",
	subcmd("ipc", flagColor("-i"), valueColor(`"1.1.1.1"`), flagColor("-o"), valueColor("cluster-info.txt")),

	"Silent mode (errors only)",
	subcmd("ipc", flagColor("-i"), valueColor(`"1.1.1.1"`), flagColor("-q")),

	"Pipeline: Resolve domains then cluster IPs",
	pipe("cat "+valueColor("domains.txt"), cmd("dnsx")+flagColor(" -silent -resp-only"), subcmd("ipc")),

	"Cluster IPs from multiple CIDRs",
	"echo -e "+valueColor(`"1.1.1.0/24\n8.8.8.0/24"`)+pipeColor(" |")+" "+subcmd("ipc", flagColor("--json")),
	)
}

const githubLong = `Search public GitHub code via grep.app to find domains, API keys, credentials,
and other sensitive information exposed in public repositories. Automatically extracts
subdomains matching the input domain, or shows full code snippets with --detail flag.`

func buildGithubExample() string {
	return exampleBlock(
	"Search for a domain in code",
	pipe(echo("hackerone.com"), subcmd("github")),

	"Alternative input",
	subcmd("github", flagColor("-i"), valueColor(`"example.com"`)),

	"Search for API keys or secrets patterns",
	pipe(echo("AKIA"), subcmd("github")),

	"Show formatted code snippets with repo and path",
	pipe(echo("example.com"), subcmd("github", flagColor("--detail"))),

	"JSON output with repo, path, and snippet",
	pipe(echo("example.com"), subcmd("github", flagColor("--json"))),

	"Save results to file",
	pipe(echo("target.com"), subcmd("github", flagColor("-o"), valueColor("github-results.txt"))),

	"Search for specific patterns",
	pipe(echo("api_key="), subcmd("github", flagColor("--detail"))),

	"Batch search from file",
	subcmd("github", flagColor("-I"), valueColor("keywords.txt"), flagColor("--json"), flagColor("-o"), valueColor("code-findings.json")),

	"Silent mode (errors only)",
	subcmd("github", flagColor("-i"), valueColor(`"example.com"`), flagColor("-q")),
	)
}

const relatedLong = `Find domains related to a target via multiple OSINT sources including:
  - crt: Certificate Transparency logs (crt.sh)
  - whois: Reverse WHOIS lookups (viewdns.info)
  - ua/gtm: Google Analytics and Tag Manager correlation
  - all: Query all available sources (default)`

func buildRelatedExample() string {
	return exampleBlock(
	"Use all sources (crt.sh, WHOIS, analytics)",
	pipe(echo("hackerone.com"), subcmd("related")),

	"All sources with input flag",
	subcmd("related", flagColor("-i"), valueColor(`"tesla.com"`)),

	"Certificate transparency only",
	pipe(echo("hackerone.com"), subcmd("related", flagColor("-s"), valueColor("crt"))),

	"CRT source with JSON output",
	subcmd("related", flagColor("-i"), valueColor(`"spacex.com"`), flagColor("-s"), valueColor("crt"), flagColor("--json")),

	"Reverse WHOIS via viewdns.info",
	pipe(echo("hackerone.com"), subcmd("related", flagColor("-s"), valueColor("whois"))),

	"WHOIS with input flag",
	subcmd("related", flagColor("-i"), valueColor(`"example.com"`), flagColor("-s"), valueColor("whois")),

	"Google Analytics / GTM correlation",
	pipe(echo("hackerone.com"), subcmd("related", flagColor("-s"), valueColor("ua"))),

	"Analytics with JSON",
	subcmd("related", flagColor("-i"), valueColor(`"target.com"`), flagColor("-s"), valueColor("ua"), flagColor("--json")),

	"Multiple sources with JSON output",
	subcmd("related", flagColor("-i"), valueColor(`"apple.com"`), flagColor("-s"), valueColor("crt,whois"), flagColor("--json")),

	"Save results to file",
	pipe(echo("target.com"), subcmd("related", flagColor("-o"), valueColor("related-domains.txt"))),

	"Specific source to file",
	subcmd("related", flagColor("-i"), valueColor(`"microsoft.com"`), flagColor("-s"), valueColor("crt"), flagColor("-o"), valueColor("domains.txt")),

	"Batch processing from file",
	subcmd("related", flagColor("-I"), valueColor("domains.txt"), flagColor("-s"), valueColor("all"), flagColor("-o"), valueColor("all-related.txt")),

	"Batch with JSON output",
	subcmd("related", flagColor("-I"), valueColor("domains.txt"), flagColor("--json"), flagColor("-o"), valueColor("all-related.json")),

	"Silent mode (errors only)",
	subcmd("related", flagColor("-i"), valueColor(`"example.com"`), flagColor("-q")),

	"Pipeline: Find related, then search their certs",
	pipe(echo("target.com"), subcmd("related", flagColor("-s"), valueColor("crt")), subcmd("cert", flagColor("--clean"))),
	)
}

const cdnLong = `Check if IP addresses belong to a CDN or WAF provider using the cdncheck library.
Useful for filtering out CDN/WAF IPs to find origin servers, or identifying the
CDN/WAF vendors protecting a target.`

func buildCDNExample() string {
	return exampleBlock(
	"Check if IPs are behind CDN/WAF (shows vendor and type)",
	pipe(echo("1.1.1.1"), subcmd("cdn")),

	"Alternative input (long form)",
	subcmd("cdn", flagColor("--input"), valueColor(`"8.8.8.8"`)),

	"Alternative input (short form)",
	subcmd("cdn", flagColor("-i"), valueColor(`"8.8.8.8"`)),

	"Check from file",
	pipe("cat "+valueColor("ips.txt"), subcmd("cdn")),

	"Single IP with input flag",
	subcmd("cdn", flagColor("-i"), valueColor(`"1.1.1.1"`)),

	"Strip CDN/WAF IPs from output (show only non-CDN IPs)",
	pipe("cat "+valueColor("ips.txt"), subcmd("cdn", flagColor("--strip-cdn"))),

	"Strip CDN from file to file",
	subcmd("cdn", flagColor("-I"), valueColor("ips.txt"), flagColor("--strip-cdn"), flagColor("-o"), valueColor("origins.txt")),

	"JSON output with full details",
	pipe(echo("1.1.1.1"), subcmd("cdn", flagColor("--json"))),

	"JSON to file",
	subcmd("cdn", flagColor("-i"), valueColor(`"8.8.8.8"`), flagColor("--json"), flagColor("-o"), valueColor("cdn-info.json")),

	"From file",
	subcmd("cdn", flagColor("-I"), valueColor("ips.txt"), flagColor("-o"), valueColor("cdn-results.txt")),

	"File input with JSON",
	subcmd("cdn", flagColor("-I"), valueColor("ips.txt"), flagColor("--json"), flagColor("-o"), valueColor("cdn-full.json")),

	"Pipeline: Resolve domains then check for CDN",
	pipe("cat "+valueColor("domains.txt"), cmd("dnsx")+flagColor(" -silent -resp-only"), subcmd("cdn")),

	"Find origin servers (non-CDN IPs only)",
	pipe("cat "+valueColor("ips.txt"), subcmd("cdn", flagColor("--strip-cdn"), flagColor("-o"), valueColor("origin-ips.txt"))),

	"Strip CDN from file",
	subcmd("cdn", flagColor("-I"), valueColor("all-ips.txt"), flagColor("--strip-cdn"), flagColor("-o"), valueColor("non-cdn-ips.txt")),

	"Silent mode (errors only)",
	subcmd("cdn", flagColor("-i"), valueColor(`"1.1.1.1"`), flagColor("-q")),
	)
}

const updateLong = `Download or update the local ASN database required for 'net' and 'ipc' commands.
The database is saved to ~/.metabigor/ip-asn-combined.csv and should be updated
periodically to ensure accurate results.`

func buildUpdateExample() string {
	return exampleBlock(
		"Download/update ASN database (run before first use)",
		cmd("metabigor update"),

		"Database location: ~/.metabigor/ip-asn-combined.csv",
		"",
	)
}

const versionLong = `Display the version, build date, commit hash, and author information`

func buildVersionExample() string {
	return exampleBlock(
		"Show version information",
		cmd("metabigor version"),

		"Alternative version flag",
		cmd("metabigor")+flagColor(" --version"),
	)
}
