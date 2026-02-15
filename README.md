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

Metabigor is Intelligence tool, its goal is to do OSINT tasks and more but without any API key.

## Features

- **Network Discovery** - Find IP ranges (CIDRs) from ASN, organization, domain, or IP address
- **Certificate Transparency** - Discover subdomains via crt.sh certificate logs
- **IP Enrichment** - Get port, service, and vulnerability data via Shodan InternetDB (free, no API key)
- **GitHub Code Search** - Find secrets, credentials, and subdomains in public repositories via grep.app
- **IP Clustering** - Group IPs by ASN for infrastructure mapping
- **Related Domains** - Discover related domains via certificate logs, reverse WHOIS, and analytics tracking
- **CDN/WAF Detection** - Identify if IPs are behind CDN or WAF providers

## Installation

### Pre-built Binaries

You can download pre-built binaries for your platform from the [**releases page**](https://github.com/j3ssie/metabigor/releases). Choose the appropriate binary for your operating system and architecture, download it, and place it in your `PATH`. 

### Build from repository

```bash
git clone https://github.com/j3ssie/metabigor.git
cd metabigor
make build
# binary at ./bin/metabigor
```

## Commands

### `net` — Network Discovery

Discover CIDRs and network ranges for an ASN, IP, domain, or organization. Auto-detects input type.

```bash
# Find CIDRs for an ASN
echo "AS13335" | metabigor net
metabigor net --input AS13335

# Lookup which ASN owns an IP
echo "1.1.1.1" | metabigor net --ip

# Find CIDRs associated with a domain
echo "cloudflare.com" | metabigor net --domain

# Search by organization name
echo "Cloudflare" | metabigor net --org

# Use live online sources instead of local DB
echo "AS13335" | metabigor net --dynamic

# Batch input from file
metabigor net -I asn-list.txt -o results.txt

# Force input type with override flags
echo "13335" | metabigor net --asn
```

### `cert` — Certificate Transparency Search

Query crt.sh for domains and certificates.

```bash
# Find domains in certificate transparency logs
echo "hackerone.com" | metabigor cert

# Search by organization name
echo "HackerOne Inc" | metabigor cert

# Strip wildcard prefixes (*.) from results
echo "example.com" | metabigor cert --clean

# Show only wildcard entries
echo "example.com" | metabigor cert --wildcard

# JSON output
echo "example.com" | metabigor cert --json

# Save to file
echo "example.com" | metabigor cert --clean -o domains.txt
```

### `ip` — IP Enrichment (Shodan InternetDB)

Query Shodan's free InternetDB API for open ports, hostnames, vulnerabilities, and tags.

```bash
# Lookup a single IP
echo "1.1.1.1" | metabigor ip

# Scan a CIDR range (auto-expands)
echo "1.1.1.0/28" | metabigor ip

# Flat IP:PORT output (useful for piping to other tools)
echo "1.1.1.0/28" | metabigor ip --flat

# CSV output
echo "1.1.1.0/28" | metabigor ip --csv

# JSON output
echo "1.1.1.1" | metabigor ip --json

# Bulk scan from file with higher concurrency
metabigor ip -I ips.txt -c 20 --flat -o open-ports.txt
```

### `github` — Code Search (grep.app)

Search public GitHub code via grep.app.

```bash
# Search for a domain in code
echo "hackerone.com" | metabigor github

# Search for API keys or secrets patterns
echo "AKIA" | metabigor github

# Paginate results
echo "example.com" | metabigor github --page 2

# JSON output with repo, path, snippet
echo "example.com" | metabigor github --json

# Save results
echo "target.com" | metabigor github -o github-results.txt
```

### `ipc` — IP Clustering

Group a list of IPs by ASN using the local database. Useful for understanding the infrastructure behind a set of IPs.

```bash
# Cluster IPs from stdin
cat ips.txt | metabigor ipc

# JSON output with full details
cat ips.txt | metabigor ipc --json

# From file
metabigor ipc -I ips.txt -o clusters.txt
```

### `related` — Related Domain Discovery

Find domains related to a target via multiple OSINT sources.

```bash
# Use all sources (crt.sh, WHOIS, analytics)
echo "hackerone.com" | metabigor related

# Certificate transparency only
echo "hackerone.com" | metabigor related -s crt

# Reverse WHOIS via viewdns.info
echo "hackerone.com" | metabigor related -s whois

# Google Analytics / GTM correlation
echo "hackerone.com" | metabigor related -s ua

# Save results
echo "target.com" | metabigor related -o related-domains.txt
```

### `cdn` — CDN/WAF Detection

Check if IPs belong to CDN or WAF providers. Shows vendor and type by default.

```bash
# Check if IP is behind CDN/WAF (shows vendor and type)
echo "1.1.1.1" | metabigor cdn

# Check multiple IPs from file
cat ips.txt | metabigor cdn

# Strip CDN/WAF IPs from output (show only non-CDN IPs)
cat ips.txt | metabigor cdn --strip-cdn

# JSON output with full details
echo "1.1.1.1" | metabigor cdn --json

# Find origin servers (non-CDN IPs only)
cat ips.txt | metabigor cdn --strip-cdn -o origin-ips.txt

# Pipeline: Resolve domains then check for CDN
cat domains.txt | dnsx -silent -resp-only | metabigor cdn
```

### `update` — Update ASN Database

```bash
metabigor update
```

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

# Disclaimer

This tool is for educational purposes only. You are responsible for your own actions. If you mess something up or break
any laws while using this software, it's your fault, and your fault only.

# License

`Metabigor` is made with ♥ by [@j3ssie](https://twitter.com/j3ssie) and it is released under the MIT license.

# Donation

[![paypal](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://paypal.me/j3ssiejjj)

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/j3ssie)
