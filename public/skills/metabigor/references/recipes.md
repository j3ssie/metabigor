# Metabigor recipes

Multi-step workflows. Single commands live in the SKILL.md command router.

## 1. Map an organization's footprint from a name

```bash
# Company name -> announced network ranges -> live hosts and their ports.
metabigor net --org "Tesla" -f flat \
  | metabigor ip -f flat -c 30 \
  | sort -u
```

`net --org` searches the offline database by organization string; pipe the CIDRs
into `ip` to see what is actually listening.

## 2. ASN to attack surface

```bash
# Every CIDR an ASN announces, enriched with ports/CVEs, saved as CSV.
metabigor net AS13335 -f flat \
  | metabigor ip -f csv -o cloudflare-surface.csv
```

## 3. Subdomains from every angle

```bash
# Certificate transparency + code search, deduplicated, then resolved.
{ metabigor cert tesla.com -f flat
  metabigor github tesla.com --subs -f flat
} | sort -u | dnsx -silent -resp-only > tesla-live.txt
```

`cert` covers issued certificates; `github --subs` catches hostnames hard-coded
in public code that never got a certificate.

## 4. Find candidate origin servers behind a CDN

```bash
# Resolve subdomains -> IPs -> drop anything behind a CDN/WAF.
metabigor cert tesla.com -f flat \
  | dnsx -silent -resp-only \
  | metabigor cdn --exclude -f flat \
  | sort -u > candidate-origins.txt
```

Whatever survives `--exclude` is a host answering directly rather than through a
CDN edge — the interesting targets for origin discovery.

## 5. Cluster a pile of IPs by who owns them

```bash
# Group resolved IPs by ASN, largest infrastructure footprint first.
cat all-ips.txt | metabigor cluster -f csv -o clusters.csv
```

Reveals which hosting providers and cloud regions a target leans on most.

## 6. Pivot from one domain to a company's other domains

```bash
# Related domains (cert + reverse WHOIS + shared analytics), then their subs.
metabigor related tesla.com --sources all -f flat \
  | metabigor cert -f flat \
  | sort -u
```

## 7. Harvest archived URLs, then probe the live ones

```bash
# Every URL the web archives have seen (subdomains included), then httpx.
metabigor url tesla.com -f flat \
  | httpx -silent -mc 200,301,302 \
  | tee live-urls.txt
```

Add `--keywords '\.js(\?|$)'` to `url` to pull JavaScript only, or
`--sources ghostarchive` to lean on WARC-mined API endpoints (see
`url-command.md`).

## 8. Keep pipelines quiet and CI-friendly

```bash
# -q silences progress; non-zero exit stops the chain on failure.
set -euo pipefail
metabigor net AS13335 -q -f flat \
  | metabigor ip -q -f flat \
  > surface.txt
```

Results still go to stdout under `-q`; only the `[info]`/`[verbose]` lines on
stderr are suppressed.
