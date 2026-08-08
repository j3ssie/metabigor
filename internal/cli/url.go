package cli

import (
	"strings"

	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/urlsource"
	"github.com/spf13/cobra"
)

const urlLong = `Collect every URL public archives and indexes have seen for a target.

Sources:
  wayback       Wayback Machine CDX index (web.archive.org)
  commoncrawl   Common Crawl CDX indexes (index.commoncrawl.org)
  otx           AlienVault OTX url_list (otx.alienvault.com)
  urlscan       urlscan.io search - page URL, task URL, and PTR record
  ghostarchive  ghostarchive.org, including URLs mined from its WARC files
  virustotal    VirusTotal v3 domain relationships - needs VT_API_KEY
  intelx        Intelligence X phonebook - needs INTELX_API_KEY

The five keyless sources run by default. VirusTotal and Intelligence X join in
only when their key is present in the environment, so there is still nothing to
configure; naming one explicitly without its key is an error rather than a
silent skip. urlscan works anonymously and uses URLSCAN_API_KEY when it is set.

Subdomains are included by default: the query is *.target, not target. Pass
--no-subs for the exact hostname only. Scoping the input to a path
(example.com/admin) searches that prefix instead.

GhostArchive is worth the extra requests it costs. Alongside the pages it has
archived, it serves the raw WARC of each capture, and a WARC records every
sub-request the browser made while archiving - XHR calls, JSON endpoints,
dynamically built script URLs. None of that appears in any CDX index. Use
--limit-requests to bound how much of it a run will pull.

Filters are off by default so nothing is dropped without being asked. Both CDX
sources apply status, MIME, date, and keyword conditions server-side, which cuts
the response size; the rest are re-checked locally, and sources that report no
MIME type or status keep results the filters cannot judge.`

var urlCmd = &cobra.Command{
	Use:         "url [domain...]",
	Short:       "Collect known URLs for a target from web archives",
	Long:        urlLong,
	GroupID:     groupRecon,
	Annotations: map[string]string{"sample": "hackerone.com"},
	Example: examples(
		"collect every URL, subdomains included", "metabigor url hackerone.com",
		"one hostname only", "metabigor url blog.hackerone.com --no-subs",
		"search a path prefix", "metabigor url hackerone.com/reports",
		"pick sources", "metabigor url tesla.com --sources wayback,urlscan",
		"search the whole Common Crawl history", "metabigor url tesla.com --sources commoncrawl --limit-collections 0",
		"JavaScript files only", `metabigor url tesla.com --keywords '\.js(\?|$)'`,
		"drop static assets and query-string duplicates", "metabigor url tesla.com --blacklist default --no-params",
		"live pages captured since 2023", "metabigor url tesla.com --from 2023 --filter-status 404,301,302",
		"show which source found what", "metabigor url tesla.com --detail",
		"feed the next tool", "metabigor url tesla.com -f flat | httpx",
		"batch a list of targets", "metabigor url -I domains.txt -o urls.txt",
	),
	RunE: runURL,
}

func init() {
	f := urlCmd.Flags()
	f.SortFlags = false

	f.StringSliceVarP(&opt.URL.Sources, "sources", "s", nil,
		"Sources to query: "+strings.Join(urlsource.SourceNames(), ", ")+", or all (default all available)")
	f.BoolVar(&opt.URL.NoSubs, "no-subs", false, "Query the exact hostname instead of *.hostname")
	f.BoolVar(&opt.URL.Detail, "detail", false, "Add source, status, MIME, and timestamp to each line")

	f.StringSliceVar(&opt.URL.MatchStatus, "match-status", nil, "Keep only these HTTP status codes, e.g. 200,301")
	f.StringSliceVar(&opt.URL.FilterStatus, "filter-status", nil, "Drop these HTTP status codes, e.g. 404,301,302")
	f.StringSliceVar(&opt.URL.MatchMIME, "match-mime", nil, "Keep only these MIME types, e.g. text/html")
	f.StringSliceVar(&opt.URL.FilterMIME, "filter-mime", nil, "Drop these MIME types, e.g. text/css,image/png")
	f.StringVar(&opt.URL.From, "from", "", "Earliest capture date: YYYY, YYYYMM, or YYYYMMDD")
	f.StringVar(&opt.URL.To, "to", "", "Latest capture date, same format as --from")
	f.StringVar(&opt.URL.Keywords, "keywords", "", `Keep only URLs matching this regex, e.g. '\.js(\?|$)'`)
	f.StringSliceVar(&opt.URL.Blacklist, "blacklist", nil,
		"Drop these file extensions, or 'default' for the built-in static-asset list")
	f.BoolVar(&opt.URL.NoParams, "no-params", false, "Keep one URL per endpoint, dropping query-string duplicates")

	f.IntVar(&opt.URL.LimitRequests, "limit-requests", 0, "Cap the requests each source may spend (0 = unlimited)")
	f.IntVar(&opt.URL.LimitCollections, "limit-collections", 3,
		"Common Crawl index collections to search, newest first (0 = all)")

	urlCmd.MarkFlagsMutuallyExclusive("match-status", "filter-status")
	urlCmd.MarkFlagsMutuallyExclusive("match-mime", "filter-mime")

	rootCmd.AddCommand(urlCmd)
}

func runURL(cmd *cobra.Command, _ []string) error {
	// Validate sources and filters before opening the writer, so a typo or a
	// broken regex fails once, up front, rather than once per target.
	sources, err := urlsource.ParseSources(opt.URL.Sources)
	if err != nil {
		return err
	}

	filters := urlsource.Filters{
		MatchStatus:  opt.URL.MatchStatus,
		FilterStatus: opt.URL.FilterStatus,
		MatchMIME:    opt.URL.MatchMIME,
		FilterMIME:   opt.URL.FilterMIME,
		From:         opt.URL.From,
		To:           opt.URL.To,
		Keywords:     opt.URL.Keywords,
		Blacklist:    opt.URL.Blacklist,
		NoParams:     opt.URL.NoParams,
	}
	if err := filters.Compile(); err != nil {
		return err
	}

	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	cfg := urlsource.Config{
		Sources:          sources,
		Filters:          filters,
		IncludeSubs:      !opt.URL.NoSubs,
		Detail:           opt.URL.Detail,
		LimitRequests:    opt.URL.LimitRequests,
		LimitCollections: opt.URL.LimitCollections,
		Concurrency:      opt.Concurrency,
	}

	output.Info("Collecting URLs for %d target(s) via %d source(s)", len(rc.Inputs), len(sources))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	// Targets run one at a time on purpose. Every target already fans out across
	// seven sources and their pages, bounded by -c, so running targets in
	// parallel too would multiply that into a burst the archives rate-limit.
	for _, input := range rc.Inputs {
		results, err := urlsource.Find(client, input, cfg)
		if err != nil {
			output.Error("%s: %v", input, err)
			continue
		}

		output.Verbose("Found %d URL(s) for %s", len(results), input)
		for _, result := range results {
			rc.Writer.Write(result)
		}
	}
	return nil
}
