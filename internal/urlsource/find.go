package urlsource

import (
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

// Config controls one URL discovery run.
type Config struct {
	// Sources to query, already validated by ParseSources.
	Sources []Source
	// Filters narrows results before and after they leave each source.
	Filters Filters
	// IncludeSubs queries `*.host` rather than the exact hostname. This is the
	// single biggest lever on how much a run returns.
	IncludeSubs bool
	// Detail is propagated to each Result for text rendering.
	Detail bool
	// LimitRequests caps the HTTP requests each source may spend. 0 is
	// unlimited.
	LimitRequests int
	// LimitCollections caps how many Common Crawl index collections are
	// searched. 0 searches every one.
	LimitCollections int
	// Concurrency caps HTTP requests in flight across all sources at once.
	Concurrency int
}

// Find queries every configured source for URLs belonging to one target.
//
// Sources run concurrently and are isolated from each other: a source that
// fails, times out, or gets rate-limited logs the reason and leaves whatever it
// already collected in place, rather than aborting the run. Results are
// deduplicated across sources, keeping the first source to report each URL.
func Find(client *retryablehttp.Client, input string, cfg Config) ([]Result, error) {
	target, err := ParseTarget(input)
	if err != nil {
		return nil, err
	}

	r := newRun(client, target, cfg)
	output.Verbose("Scope for %s: %s", target.Host, target.CDXScope(cfg.IncludeSubs))

	var wg sync.WaitGroup
	for _, src := range cfg.Sources {
		wg.Add(1)
		go func(src Source) {
			defer wg.Done()
			// These parsers consume untrusted HTML, WARC, and JSON from seven
			// different services. A malformed response should cost one source,
			// not the whole run, so a panic is contained and reported here.
			defer func() {
				if p := recover(); p != nil {
					output.Error("%s: internal error while parsing response: %v", src, p)
				}
			}()

			if err := r.dispatch(src); err != nil {
				output.Error("%s: %v", src, err)
			}
			// Count this source's own contribution. A global before/after delta
			// would fold in whatever the concurrent sources added meanwhile.
			output.Verbose("%s: %d new URL(s) for %s", src, r.addedFor(src), target.Host)
		}(src)
	}
	wg.Wait()

	results := r.snapshot()
	sort.Slice(results, func(i, j int) bool { return results[i].URL < results[j].URL })
	return results, nil
}

// run holds the per-target state every source in one invocation shares.
type run struct {
	client *retryablehttp.Client
	target Target
	cfg    Config

	mu      sync.Mutex
	seen    map[string]bool
	results []Result
	// added counts what each source contributed, for the per-source verbose
	// line. Sources run concurrently, so this cannot be derived from the total.
	added map[Source]int

	// slots caps HTTP requests in flight. Each source paginates in parallel, so
	// without a shared cap seven sources would multiply into a burst that the
	// archives would rightly rate-limit.
	slots chan struct{}

	// keys holds each source's API key, resolved once from the environment when
	// the run starts rather than re-read on every request.
	keys map[Source]string
}

func newRun(client *retryablehttp.Client, target Target, cfg Config) *run {
	concurrency := cfg.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	keys := make(map[Source]string, len(cfg.Sources))
	for _, src := range cfg.Sources {
		if key := APIKey(src); key != "" {
			keys[src] = key
		}
	}

	return &run{
		client: client,
		target: target,
		cfg:    cfg,
		seen:   make(map[string]bool),
		added:  make(map[Source]int),
		slots:  make(chan struct{}, concurrency),
		keys:   keys,
	}
}

// key returns the API key configured for a source, or "" when it has none.
func (r *run) key(src Source) string { return r.keys[src] }

// dispatch runs the source's implementation, taken from the registry so there
// is no second list of sources to keep in step.
func (r *run) dispatch(src Source) error {
	d, ok := lookup(src)
	if !ok {
		return fmt.Errorf("unknown source %q", src)
	}
	return d.Fetch(r)
}

// add records one discovered URL, after normalizing it, confirming it belongs
// to the target, and applying the local filters. Duplicates are dropped.
func (r *run) add(src Source, rawURL, status, mime, timestamp string) {
	normalized := Normalize(rawURL)
	if normalized == "" {
		return
	}
	if !InScope(normalized, r.target) {
		return
	}
	if !r.cfg.Filters.Allow(normalized, status, mime) {
		return
	}
	if !r.cfg.Filters.AllowTimestamp(timestamp) {
		return
	}

	// --no-params keeps one URL per endpoint, so a page reachable with fifty
	// query-string permutations reports once.
	key := normalized
	if r.cfg.Filters.NoParams {
		key = EndpointKey(normalized)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seen[key] {
		return
	}
	r.seen[key] = true
	r.added[src]++
	r.results = append(r.results, Result{
		Input:     r.target.Raw,
		URL:       normalized,
		Source:    src,
		Status:    status,
		MIME:      mime,
		Timestamp: timestamp,
		Detail:    r.cfg.Detail,
	})
}

// addedFor returns how many URLs a source contributed that no earlier source
// had already reported.
func (r *run) addedFor(src Source) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.added[src]
}

func (r *run) snapshot() []Result {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Result, len(r.results))
	copy(out, r.results)
	return out
}

// get issues a GET, waiting for a free request slot first.
func (r *run) get(targetURL string, headers map[string]string) (*httpclient.Response, error) {
	return r.do("GET", targetURL, headers, nil)
}

// post issues a POST with a JSON body, waiting for a free request slot first.
func (r *run) post(targetURL string, headers map[string]string, body io.Reader) (*httpclient.Response, error) {
	return r.do("POST", targetURL, headers, body)
}

func (r *run) do(method, targetURL string, headers map[string]string, body io.Reader) (*httpclient.Response, error) {
	r.slots <- struct{}{}
	defer func() { <-r.slots }()
	return httpclient.Fetch(r.client, method, targetURL, headers, body)
}

// budget tracks how many requests a single source has spent, so one enormous
// target cannot consume the whole run.
type budget struct {
	src    Source
	limit  int
	mu     sync.Mutex
	spent  int
	warned bool
}

func (r *run) newBudget(src Source) *budget {
	return &budget{src: src, limit: r.cfg.LimitRequests}
}

// clamp caps a planned page count at what the budget can still afford, so a
// target spanning thousands of pages does not queue work that is certain to be
// refused. Sources use this instead of reading the limit out of the config
// themselves, which kept three copies of the same arithmetic in step by luck.
func (b *budget) clamp(pages int) int {
	if b.limit <= 0 {
		return pages
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if remaining := b.limit - b.spent; pages > remaining {
		return max(remaining, 0)
	}
	return pages
}

// claim reserves one request, reporting false once the budget is exhausted.
// Hitting the cap is logged once, because silently truncating results would
// read as "this is everything there is".
func (b *budget) claim() bool {
	if b.limit <= 0 {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.spent >= b.limit {
		if !b.warned {
			b.warned = true
			output.Warn("%s: stopped at the --limit-requests cap of %d; results are partial", b.src, b.limit)
		}
		return false
	}
	b.spent++
	return true
}

// fan runs tasks concurrently and returns once every one has finished.
func (r *run) fan(tasks []func()) {
	parallel(r.cfg.Concurrency, tasks)
}

// fetchPages retrieves pages [first, last) concurrently, handing each response
// to parse. urlFor builds the request URL for a page number.
//
// The CDX sources all page the same way once their page count is known, so the
// budget accounting, per-page error reporting, and fan-out live here rather
// than being restated - and drifting - in each of them. A page that fails is
// logged and skipped: partial results beat none.
func (r *run) fetchPages(b *budget, label string, first, last int, urlFor func(int) string, parse func(*httpclient.Response) error) {
	tasks := make([]func(), 0, max(last-first, 0))
	for page := first; page < last; page++ {
		tasks = append(tasks, func() {
			resp, err := r.claimGet(b, urlFor(page), nil)
			if err != nil {
				output.Verbose("%s: page %d failed: %v", label, page, err)
				return
			}
			if resp == nil {
				return
			}
			if err := parse(resp); err != nil {
				output.Verbose("%s: page %d: %v", label, page, err)
			}
		})
	}
	r.fan(tasks)
}

// parallel runs tasks concurrently, at most limit at a time.
//
// HTTP requests are already capped by the run's shared slots; this second bound
// exists so a target spanning hundreds of index pages across a hundred index
// collections does not create tens of thousands of goroutines that all block.
func parallel(limit int, tasks []func()) {
	if limit < 1 {
		limit = 1
	}

	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(fn func()) {
			defer wg.Done()
			defer func() { <-sem }()
			fn()
		}(task)
	}
	wg.Wait()
}

// claimGet spends one unit of a source's request budget and issues the GET. It
// returns (nil, nil) when the budget is exhausted, which callers treat as "stop
// here, keep what you have".
func (r *run) claimGet(b *budget, targetURL string, headers map[string]string) (*httpclient.Response, error) {
	if !b.claim() {
		return nil, nil
	}
	return r.get(targetURL, headers)
}
