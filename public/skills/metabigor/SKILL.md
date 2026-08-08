---
name: metabigor
description: >-
  Use when operating the metabigor CLI for OSINT recon and infrastructure
  mapping without API keys. Covers finding network ranges from an ASN, org,
  domain, or IP (net); enumerating subdomains from certificate logs (cert);
  enriching IPs with ports/CVEs via Shodan InternetDB (ip); searching public
  GitHub code for secrets and subdomains (github); clustering IPs by ASN
  (cluster); pivoting to related domains via crt.sh/WHOIS/analytics (related);
  detecting CDN/WAF vendors and isolating candidate origins (cdn); collecting
  archived URLs and WARC-mined endpoints (url); refreshing the offline
  databases (update); and chaining these commands into a recon pipeline.
license: MIT
tags:
  - osint
  - recon
  - subdomain-enumeration
  - asn
  - network-discovery
---

# Metabigor

CLI-first OSINT tool that maps a target's infrastructure from **free sources,
no API keys**. Part of the Osmedeus Engine. Every command takes targets the same
four ways and renders through one `-f/--format` flag, so any command pipes
cleanly into the next.

## TL;DR

```bash
metabigor net AS13335                  # network ranges (CIDRs) behind an ASN
metabigor cert hackerone.com           # subdomains from certificate logs
metabigor ip 1.1.1.1                   # open ports + CVEs for an IP (free, no key)
metabigor related tesla.com            # other domains the target owns
metabigor cdn --exclude -I ips.txt     # drop CDN/WAF IPs, keep candidate origins
metabigor url hackerone.com -f flat    # every archived URL, ready to pipe
```

`metabigor <command> -h` is authoritative for the version you have installed.

## Mental model

- **Zero configuration, no API keys.** `net`/`cluster` work fully offline from a
  bundled database. The `url` command *optionally* reads keys from the
  environment only (`VT_API_KEY`/`VIRUSTOTAL_API_KEY`, `INTELX_API_KEY`,
  `URLSCAN_API_KEY`) — never from a flag or config file.
- **Results to stdout, logs to stderr.** Pipes stay clean at any log level, so
  `metabigor cert x.com | other-tool` always works. Progress is quiet by
  default; add `-v` for step-by-step lines.
- **One output flag.** `-f/--format text|flat|json|csv` covers every command.
  There are no per-command format flags.
- **Everything composes.** The `flat` format emits the bare primary value so one
  command's output is the next command's input.
- **Non-zero exit on failure**, so metabigor drops into `&&`, `set -e`, and CI.

## The four ways to pass targets

Identical on every command; the merged list is deduplicated.

```bash
metabigor cert example.com                 # as an argument
metabigor cert example.com tesla.com       # several arguments
metabigor cert -i example.com              # with -i (use when a target looks like a flag)
metabigor cert -I domains.txt              # from a file, one per line (# comments ignored)
cat domains.txt | metabigor cert           # on stdin
```

Stdin is read **only when no target was given another way**, so
`metabigor net AS13335` never blocks inside a script or CI job.

## Command router

| I need to… | Use |
|---|---|
| Find CIDRs announced by an ASN | `metabigor net AS13335` |
| Find which ASN owns an IP | `metabigor net 1.1.1.1 --detail` |
| Find ranges by company name | `metabigor net --org Cloudflare` |
| Query live BGP sources, not the local DB | `metabigor net --live tesla.com` |
| Enumerate subdomains from cert logs | `metabigor cert hackerone.com` |
| Cert search by organization | `metabigor cert "HackerOne Inc"` |
| Strip `*.` from wildcard cert entries | `metabigor cert example.com --clean` |
| Cert IDs, issuers, validity dates | `metabigor cert example.com --detail` |
| Ports / hostnames / CVEs for an IP | `metabigor ip 1.1.1.1` |
| Enrich a whole CIDR | `metabigor ip 1.1.1.0/28` |
| Search public GitHub code (grep.app) | `metabigor github hackerone.com` |
| Pull only subdomains out of code matches | `metabigor github tesla.com --subs` |
| Show the matching code lines | `metabigor github "api_key=" --detail` |
| Group IPs by owning ASN | `metabigor cluster -I ips.txt` |
| Pivot to related domains (all sources) | `metabigor related hackerone.com` |
| Pivot via specific sources | `metabigor related tesla.com --sources crt,whois` |
| Collect archived URLs (keyless sources) | `metabigor url hackerone.com` |
| Scope URL collection to one host | `metabigor url blog.hackerone.com --no-subs` |
| Detect CDN/WAF vendor for an IP | `metabigor cdn 1.1.1.1` |
| Drop CDN/WAF IPs, keep origins | `metabigor cdn --exclude -I ips.txt` |
| Keep only CDN/WAF-protected IPs | `metabigor cdn --only -I ips.txt` |
| Refresh the offline ASN/country DBs | `metabigor update` |
| Print / install these agent skills | `metabigor skills get --full` |

## Output formats

`-f/--format` on every command. Default is `text`.

| Format | Flag | What you get | Use it for |
|--------|------|--------------|------------|
| Text *(default)* | `-f text` | Readable `a | b | c` columns | Reading in a terminal |
| Flat | `-f flat` | The bare primary value, one per line | Piping into other tools |
| JSON | `-f json` | One JSON object per line | `jq`, automation |
| CSV | `-f csv` | Rows behind one header | Spreadsheets, reports |

```bash
metabigor ip 1.1.1.0/28                 # 1.1.1.1 | 80,443 | one.one.one.one
metabigor ip 1.1.1.0/28 -f flat         # 1.1.1.1:80
metabigor ip 1.1.1.0/28 -f json | jq .  # {"ip":"1.1.1.1","ports":[80,443],...}
metabigor ip 1.1.1.0/28 -f csv -o ports.csv
```

`-o <file>` also writes results to a file (overwrite; `--append` to add).

## Per-command notes

Things `-h` alone won't make obvious.

- **`net`** auto-detects the target type; override with `--asn`, `--ip`,
  `--domain`, or `--org` (mutually exclusive). `--detail` adds ASN / org /
  country columns. `--live` swaps the offline DB for live BGP sources
  (`bgp.he.net`) — slower, but current.
- **`cert`** prints a plain domain list by default. `--detail` gives the grouped
  certificate view (IDs, issuers, dates); `--clean` strips `*.`; `--wildcard`
  keeps only wildcard entries. Feed it into a resolver: `metabigor cert t.com |
  dnsx -silent`.
- **`ip`** uses Shodan InternetDB (free, no key). IPs it knows nothing about are
  **skipped**; pass `--all` to keep them. CIDRs expand to hosts.
- **`github`** needs **Chrome/Chromium** (grep.app blocks plain HTTP clients).
  Searches run **one at a time regardless of `-c`** to respect grep.app's rate
  limit. `--subs` returns subdomains only; `--detail` shows code; `--pages N`
  goes deeper.
- **`cluster`** groups IPs/CIDRs by owning ASN, **largest cluster first**. Fully
  offline.
- **`related`** sources: `crt` (crt.sh), `whois` (viewdns.info reverse WHOIS),
  `analytics` (shared Google Analytics / Tag Manager IDs), or `all` (default).
  Results are deduped across sources and tagged with the first source that found
  each.
- **`cdn`** classifies IPs by CDN/WAF vendor. `--exclude` drops protected IPs
  (leaving candidate origins); `--only` keeps just the protected ones. The two
  are mutually exclusive.
- **`url`** includes subdomains by default (queries `*.target`) — the single
  biggest lever on volume; `--no-subs` narrows it. GhostArchive is **mined**:
  its WARCs expose sub-request URLs (XHR/JSON endpoints) that no CDX index
  lists. Filters are off by default. See `references/url-command.md`.
- **`update`** refreshes both the IP-to-ASN and IP-to-country databases in
  `~/.metabigor`. A copy ships with the binary and unpacks on first use, so this
  is only needed to pick up newer routing data.

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

## Recon pipeline

The point of the `flat` format: chain commands into a workflow.

```bash
# ASN -> ranges -> live hosts with ports
metabigor net AS13335 -f flat | metabigor ip -f flat

# domain -> related domains -> their subdomains
metabigor related tesla.com -f flat | metabigor cert

# resolve subdomains, then split CDN from candidate origins
metabigor cert tesla.com | dnsx -silent -resp-only | metabigor cdn --exclude

# collect archived URLs and probe them
metabigor url tesla.com -f flat | httpx -silent
```

More multi-step workflows: `references/recipes.md`.

## Escape hatches

```bash
metabigor <command> -h     # authoritative flags for your installed version
metabigor version          # version, build date, commit
metabigor skills get --full  # print this skill and every reference
```

## Resources

- **GitHub**: [github.com/j3ssie/metabigor](https://github.com/j3ssie/metabigor)
- **Author**: [@j3ssie](https://twitter.com/j3ssie)
- **Part of**: Osmedeus Engine ([@OsmedeusEngine](https://twitter.com/OsmedeusEngine))
