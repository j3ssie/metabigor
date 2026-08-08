<p align="center">
  <img alt="Metabigor" src="https://user-images.githubusercontent.com/23289085/143042137-28f8e7e5-e485-4dc8-a09b-10759a593210.png" height="140" />
  <br />
  <strong>Metabigor - OSINT power without API key hassle</strong>

  <p align="center">
  <a href="https://docs.osmedeus.org/donation/"><img src="https://img.shields.io/badge/Sponsors-0078D4?style=for-the-badge&logo=GitHub-Sponsors&logoColor=39ff14&labelColor=black&color=black"></a>
  <a href="https://twitter.com/OsmedeusEngine"><img src="https://img.shields.io/badge/%40OsmedeusEngine-0078D4?style=for-the-badge&logo=Twitter&logoColor=39ff14&labelColor=black&color=black"></a>
  <a href="https://github.com/j3ssie/osmedeus/releases"><img src="https://img.shields.io/github/release/j3ssie/metabigor?style=for-the-badge&labelColor=black&color=2fc414&logo=Github"></a>
  </p>
</p>

***

## What is Metabigor?

Metabigor maps a target's infrastructure without asking you to register for a single API key.
Network ranges, certificates, related domains, exposed ports, CDN vendors, and code leaks — all
from free sources, all pipeable into each other.

## Features

- **Network Discovery** — find IP ranges (CIDRs) from an ASN, organization, domain, or IP
- **Certificate Transparency** — discover subdomains via crt.sh
- **IP Enrichment** — ports, hostnames, and CVEs via Shodan InternetDB (free, no key)
- **GitHub Code Search** — find secrets, credentials, and subdomains in public repos via grep.app
- **IP Clustering** — group IPs by ASN to map infrastructure
- **Related Domains** — pivot via certificate logs, reverse WHOIS, and analytics IDs
- **CDN/WAF Detection** — separate protected addresses from candidate origins
- **URL Collection** — every URL the web archives have seen, including endpoints mined from WARCs

## Installation

```bash
npm install -g @j3ssie/metabigor      # macOS, Linux, Windows
brew install j3ssie/tap/metabigor     # macOS, Linux
```

Both ship the same prebuilt binary for your platform — no Go toolchain needed.

Or build from source:

```bash
git clone https://github.com/j3ssie/metabigor.git
cd metabigor
make build          # binary at ./bin/metabigor
```

Or download a binary from the [releases page](https://github.com/j3ssie/metabigor/releases).

## The 30-second tour

```bash
metabigor net AS13335                  # network ranges behind an ASN
metabigor cert hackerone.com           # subdomains from certificate logs
metabigor ip 1.1.1.1                   # open ports and CVEs for an IP
metabigor related tesla.com            # other domains the target owns
metabigor cdn --exclude -I ips.txt     # candidate origin servers
```

## How every command works

The same conventions apply everywhere, so once you learn one command you know them all.

### Four ways to pass targets

```bash
metabigor cert example.com                  # as an argument
metabigor cert example.com tesla.com        # several arguments
metabigor cert -i example.com               # with -i
metabigor cert -I domains.txt               # from a file, one per line (# comments are ignored)
cat domains.txt | metabigor cert            # on stdin
```

They combine, and the merged list is deduplicated:

```bash
metabigor cluster 1.1.1.1 -i 8.8.8.8 -I more-ips.txt
```

Arguments are the shortest path and work on every command. Reach for `-i` when the target would
otherwise look like a flag, or when a script builds the command for you:

```bash
metabigor github -i "--endpoint-url"
metabigor net --input "$TARGET" -f json
```

Stdin is only read when no target was given any other way, so `metabigor net AS13335` never blocks
waiting on input inside a script or CI job.

### One flag for output shape

`-f/--format` works on every command:

| Format | What you get | Use it for |
|---|---|---|
| `text` *(default)* | Readable columns | Reading in a terminal |
| `flat` | The bare primary value | Piping into other tools |
| `json` | One JSON object per line | `jq`, automation |
| `csv` | Rows behind one header | Spreadsheets, reports |

```bash
metabigor ip 1.1.1.0/28                # 1.1.1.1 | 80,443 | one.one.one.one
metabigor ip 1.1.1.0/28 -f flat        # 1.1.1.1:80
metabigor ip 1.1.1.0/28 -f json        # {"ip":"1.1.1.1","ports":[80,443],...}
metabigor ip 1.1.1.0/28 -f csv -o ports.csv
```

Results always go to stdout and logs always go to stderr, so pipes stay clean at any log level.
`-o` writes results to a file as well, overwriting it unless you pass `--append`. The file is
only rewritten once there is a result to write, so a run that fails or finds nothing leaves the
previous results intact.

### Commands chain together

```bash
metabigor net AS13335 -f flat | metabigor ip -f flat
metabigor related tesla.com -f flat | metabigor cert
cat domains.txt | dnsx -silent -resp-only | metabigor cdn --exclude
```

## Commands

### `net` — Network Discovery

Find the CIDRs behind an ASN, IP, domain, or organization. Uses the bundled offline database.

```bash
metabigor net AS13335                       # every CIDR announced by an ASN
metabigor net 1.1.1.1 --detail              # which ASN owns this IP
metabigor net --org Cloudflare              # search by company name
metabigor net --live tesla.com              # query live sources instead of the local DB
metabigor net -I asn-list.txt -f csv -o ranges.csv
```

The target type is auto-detected. Override it with `--asn`, `--ip`, `--domain`, or `--org`
(mutually exclusive). `--detail` adds ASN, organization, and country columns.

### `cert` — Certificate Transparency

```bash
metabigor cert hackerone.com                # one domain per line
metabigor cert "HackerOne Inc"              # search by organization
metabigor cert example.com --clean          # strip the *. from wildcards
metabigor cert example.com --wildcard       # only wildcard entries
metabigor cert example.com --detail         # cert IDs, issuers, validity dates
metabigor cert tesla.com | dnsx -silent
```

### `ip` — IP Enrichment (Shodan InternetDB)

```bash
metabigor ip 1.1.1.1                        # ports, hostnames, CVEs
metabigor ip 1.1.1.0/28                     # CIDRs expand to hosts
metabigor ip 1.1.1.0/28 -f flat             # IP:PORT pairs
metabigor ip -I ips.txt -c 30 -f csv -o ports.csv
```

IPs InternetDB knows nothing about are skipped; pass `--all` to keep them.

### `github` — Code Search (grep.app)

```bash
metabigor github hackerone.com              # repo and path per match
metabigor github tesla.com --subs           # only subdomains found in matches
metabigor github "api_key=" --detail        # show the matching code
metabigor github AKIA --pages 3 --detail
```

Requires Chrome or Chromium — grep.app answers plain HTTP clients with a bot challenge.
Searches run one at a time regardless of `-c`, to respect grep.app's rate limit.

### `cluster` — IP Clustering

Group IPs by the ASN that owns them, largest cluster first. Accepts IPs or CIDRs. Works offline.

```bash
cat ips.txt | metabigor cluster
metabigor cluster -I ips.txt -f csv -o clusters.csv
cat domains.txt | dnsx -silent -resp-only | metabigor cluster
```

### `related` — Related Domain Discovery

```bash
metabigor related hackerone.com             # all sources
metabigor related tesla.com --sources crt
metabigor related tesla.com --sources crt,whois
```

Sources: `crt` (crt.sh), `whois` (viewdns.info reverse WHOIS), `analytics` (shared Google
Analytics / Tag Manager IDs), or `all`. Results are deduplicated across sources and tagged with
the source that found them first.

### `url` — URL Collection from Web Archives

```bash
metabigor url hackerone.com                       # every keyless source, subdomains included
metabigor url blog.hackerone.com --no-subs        # one hostname only
metabigor url hackerone.com/reports               # scope to a path prefix
metabigor url tesla.com --sources wayback,urlscan
metabigor url tesla.com --keywords '\.js(\?|$)'   # JavaScript only
metabigor url tesla.com --blacklist default --no-params
metabigor url tesla.com -f flat | httpx
```

Sources: `wayback`, `commoncrawl`, `otx`, `urlscan`, `ghostarchive`, plus `virustotal` and
`intelx` when their keys are set. The five keyless sources run by default; the credentialed two
join in automatically when `VT_API_KEY` / `INTELX_API_KEY` are present in the environment, and
`URLSCAN_API_KEY` raises urlscan's page size. Naming a credentialed source without its key is an
error rather than a silent skip.

**Subdomains are included by default** — the query is `*.target`, not `target`. This is the single
biggest lever on how much a run returns; `--no-subs` narrows it.

**GhostArchive is mined, not just listed.** Every archived page also has a raw WARC, and a WARC
records every sub-request the browser made while capturing the page — XHR calls, JSON endpoints,
dynamically built script URLs. None of that appears in a CDX index, which only indexes captures of
the page URL itself. In testing, 8 requests against `tesla.com` returned 147 URLs including live
JSON API endpoints. Bound the cost with `--limit-requests`.

Filters are off by default so nothing is dropped unasked. `--match-status` / `--filter-status`,
`--match-mime` / `--filter-mime`, `--from` / `--to`, and `--keywords` are pushed into the Wayback
and Common Crawl APIs server-side and re-checked locally; `--blacklist` and `--no-params` are
local. `--limit-requests` caps requests per source and `--limit-collections` sets how deep into
Common Crawl's ~126 index collections to search (`0` for all of them, back to 2008).

### `cdn` — CDN/WAF Detection

```bash
metabigor cdn 1.1.1.1                       # vendor and type
cat ips.txt | metabigor cdn --exclude       # candidate origins
cat ips.txt | metabigor cdn --only          # only protected addresses
```

### `update` — Refresh Local Databases

```bash
metabigor update
```

Refreshes both the IP-to-ASN and IP-to-country databases in `~/.metabigor`. A copy ships with the
binary and unpacks on first use, so this is only needed to pick up newer routing data.

### `skills` — Coding-Agent Skills

Metabigor ships an [agentskills.io](https://agentskills.io) skill that teaches an AI coding agent
(Claude Code, Codex, or any compatible agent) how to drive the CLI. The skill is embedded in the
binary, so it always matches the version you have installed.

```bash
metabigor skills                            # list the bundled skills
metabigor skills get metabigor --full       # print the skill and its references
metabigor skills install                    # install into .claude/skills (this project)
metabigor skills install --scope global     # install into ~/.claude/skills
metabigor skills install --agent agents     # install into .agents/skills for Codex
metabigor skills install --dir ./skills     # install into a directory you choose
```

`get` prints the raw Markdown to stdout (pipe it, or use `-o` to save it). `install` copies the
whole bundle — `SKILL.md` plus its `references/` — and skips an existing copy unless you pass
`--force`. Once installed, the agent auto-triggers on the skill whenever you mention metabigor.

## Global flags

```
  -i, --input string        Target to look up (also accepts arguments or stdin)
  -I, --input-file string   File of targets, one per line (use - for stdin)
  -o, --output string       Also write results to this file
  -f, --format string       Output format: text, json, csv, flat (default "text")
      --append              Append to the output file instead of overwriting it
  -c, --concurrency int     Number of parallel workers (default 10)
  -t, --timeout int         Request timeout in seconds (default 40)
      --retry int           Retries per failed request (default 3)
      --proxy string        Upstream proxy, e.g. http://127.0.0.1:8080
  -v, --verbose             Show step-by-step progress
  -q, --quiet               Show results and errors only
      --debug               Show HTTP traffic and internal traces (implies --verbose)
      --no-color            Disable colored log output
```

Metabigor exits non-zero on failure, so it composes with `&&`, `set -e`, and CI.

## Upgrading from v2

v3 renames flags and commands for consistency. There are no aliases — update your scripts.

| v2 | v3 | Why |
|---|---|---|
| `metabigor ipc` | `metabigor cluster` | `ipc` was unguessable |
| `-I, --inputFile` | `-I, --input-file` | camelCase is not a CLI convention |
| `-q, --silent` | `-q, --quiet` | `-q` should match its long form |
| `--json` | `-f json` | one flag controls output shape |
| `ip --csv` / `ip --flat` | `-f csv` / `-f flat` | same |
| `cert --simple` | *(now the default)* | `--detail` opts into the verbose block |
| `net --dynamic` | `net --live` | plain language |
| `cdn --strip-cdn` | `cdn --exclude` | pairs with the new `--only` |
| `related -s ua` / `-s gtm` | `--sources analytics` | both did the same thing |
| `related -s` | `-s, --sources` | now genuinely accepts a list |
| `github` *(subdomains by default)* | `github --subs` | output shape no longer depends on results |

Behavior changes worth knowing:

- **Progress output is quiet by default.** `[verbose]` lines now require `-v`.
- **`-o` overwrites** instead of silently appending. Use `--append` for the old behavior.
- **`ip` prints readable text**, not raw JSON. Use `-f json` for JSON.
- **`cert` prints a plain domain list.** Use `--detail` for the grouped certificate view.
- **`net` on a CIDR reports the owning network** instead of expanding to every host.
- **Errors exit 1.** Previously every failure still exited 0.
- **`metabigor update` refreshes both databases.** It used to skip the country database.

## Painless integrate Metabigor into your recon workflow?

<p align="center">
  <img alt="OsmedeusEngine" src="https://raw.githubusercontent.com/osmedeus/assets/main/part-of-osmedeus-banner.png" />
  <p align="center">
    This project was part of Osmedeus Engine. Check out how it was integrated at <a href="https://twitter.com/OsmedeusEngine">@OsmedeusEngine</a>
  </p>
</p>

# Credits

Logo from [flaticon](https://image.flaticon.com/icons/svg/1789/1789851.svg) by [freepik](https://www.flaticon.com/authors/freepik)

## 📊 Data Sources
- **Local Database** - IP-to-ASN and IP-to-Country mappings (2M+ entries)
- **crt.sh** - Certificate Transparency logs
- **Shodan InternetDB** - Free IP enrichment API
- **grep.app** - GitHub code search
- **bgp.he.net** - Live BGP routing data
- **viewdns.info** - Reverse WHOIS lookups
- **builtwith.com** - Analytics tracking correlation
- **projectdiscovery/cdncheck** - CDN/WAF detection library
- **web.archive.org** - Wayback Machine CDX index
- **index.commoncrawl.org** - Common Crawl CDX indexes
- **otx.alienvault.com** - AlienVault OTX URL lists
- **urlscan.io** - Scan history search
- **ghostarchive.org** - Archived pages and their WARC files
- **virustotal.com** - Domain relationships (optional API key)
- **intelx.io** - Intelligence X phonebook (optional API key)

# Disclaimer

This tool is for educational purposes only. You are responsible for your own actions. If you mess something up or break
any laws while using this software, it's your fault, and your fault only.

# License

`Metabigor` is made with ♥ by [@j3ssie](https://twitter.com/j3ssie) and it is released under the MIT license.

# Donation

[![paypal](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://paypal.me/j3ssiejjj)

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/j3ssie)
