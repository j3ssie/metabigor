# The `url` command in depth

Collect known URLs for a target from web archives and indexes, including
endpoints mined out of GhostArchive WARC files.

## Sources

| Source | Key required | Notes |
|--------|:------------:|-------|
| `wayback` | no | Wayback Machine CDX index (web.archive.org) |
| `commoncrawl` | no | Common Crawl CDX indexes; index list cached 30 days |
| `otx` | no | AlienVault OTX URL lists |
| `urlscan` | optional | urlscan.io scan history; `URLSCAN_API_KEY` raises page size |
| `ghostarchive` | no | Archived pages **plus WARC mining** for sub-request URLs |
| `virustotal` | yes | v3 domain relationships; needs `VT_API_KEY`/`VIRUSTOTAL_API_KEY` |
| `intelx` | yes | Intelligence X phonebook; needs `INTELX_API_KEY` |

The five keyless sources run by default. The credentialed two join automatically
when their keys are present in the **environment**. Naming a credentialed source
with `--sources` when its key is absent is an **error**, not a silent skip.

```bash
metabigor url hackerone.com                       # every keyless source
metabigor url tesla.com --sources wayback,urlscan # pick specific sources
```

## Subdomains are included by default

The query is `*.target`, not `target` — the single biggest lever on how much a
run returns.

```bash
metabigor url hackerone.com                  # *.hackerone.com
metabigor url blog.hackerone.com --no-subs   # exactly one hostname
metabigor url hackerone.com/reports          # scope to a path prefix
```

## GhostArchive is mined, not just listed

Every archived page also has a raw WARC, and a WARC records every sub-request the
browser made while capturing the page — XHR calls, JSON endpoints, dynamically
built script URLs. None of that appears in a CDX index, which only indexes
captures of the page URL itself. Bound the cost with `--limit-requests`.

```bash
metabigor url tesla.com --sources ghostarchive --limit-requests 8 -f flat
```

## Filters (all off by default, so nothing is dropped unasked)

Pushed server-side into Wayback/Common Crawl and re-checked locally:

- `--match-status` / `--filter-status`
- `--match-mime` / `--filter-mime`
- `--from` / `--to` (date bounds)
- `--keywords <regex>`

Applied locally:

- `--blacklist <name|list>` (e.g. `--blacklist default`)
- `--no-params` (strip query strings)

Limits:

- `--limit-requests` caps requests per source.
- `--limit-collections` sets how deep into Common Crawl's ~126 index collections
  to search (`0` = all, back to 2008).

## Worked examples

```bash
# JavaScript files only, subdomains included, ready to pipe.
metabigor url tesla.com --keywords '\.js(\?|$)' -f flat | httpx -silent

# Live 200s from the last two years, params stripped, common noise removed.
metabigor url tesla.com --from 2023 --match-status 200 \
  --no-params --blacklist default -f flat

# WARC-mined API endpoints only.
metabigor url tesla.com --sources ghostarchive --limit-requests 12 -f json | jq .
```
