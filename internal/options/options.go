// Package options defines configuration structures for all metabigor commands and global flags.
package options

// Options holds every global flag plus the per-command settings.
type Options struct {
	// Input
	Input     string
	InputFile string

	// Output
	Output string
	Format string
	Append bool

	// Execution
	Concurrency int
	Timeout     int
	Retry       int
	Proxy       string

	// Logging
	Quiet   bool
	Verbose bool
	Debug   bool
	NoColor bool

	Net     NetOptions
	Cert    CertOptions
	Related RelatedOptions
	IP      IPOptions
	Github  GithubOptions
	CDN     CDNOptions
	URL     URLOptions
}

// URLOptions holds configuration for the url command.
type URLOptions struct {
	Sources []string // any of: wayback, commoncrawl, otx, urlscan, ghostarchive, virustotal, intelx
	NoSubs  bool     // query the exact hostname instead of *.hostname
	Detail  bool     // add source, status, MIME, and timestamp to text output

	// Result filters. Match wins over Filter when both are set, matching the
	// upstream archive APIs, which have no way to express both at once.
	MatchStatus  []string
	FilterStatus []string
	MatchMIME    []string
	FilterMIME   []string
	From         string   // earliest capture date, YYYY[MM[DD[hhmmss]]]
	To           string   // latest capture date, same format
	Keywords     string   // regex a URL must match; pushed into the CDX APIs
	Blacklist    []string // file extensions to drop
	NoParams     bool     // keep one URL per host+path, dropping query variants

	// Request budgets, for targets whose archives are too large to walk fully.
	LimitRequests    int // per source; 0 means unlimited
	LimitCollections int // Common Crawl index collections; 0 means all
}

// NetOptions holds configuration for the net command.
type NetOptions struct {
	// Input type overrides. Mutually exclusive; empty means auto-detect.
	ASN    bool
	Org    bool
	IP     bool
	Domain bool

	Live   bool // query live online sources instead of the local database
	Detail bool // add ASN, organization, and country columns to text output
}

// CertOptions holds configuration for the cert command.
type CertOptions struct {
	Clean    bool
	Wildcard bool
	Detail   bool // show certificate IDs, issuers, and validity per domain
}

// RelatedOptions holds configuration for the related command.
type RelatedOptions struct {
	Sources []string // any of: crt, whois, analytics
}

// IPOptions holds configuration for the ip command.
type IPOptions struct {
	All bool // keep IPs that InternetDB returned no data for
}

// GithubOptions holds configuration for the github command.
type GithubOptions struct {
	Pages  int
	Detail bool // render the matching code snippet for each hit
	Subs   bool // print only subdomains extracted from the hits
}

// CDNOptions holds configuration for the cdn command.
type CDNOptions struct {
	Exclude bool // drop CDN/WAF IPs, leaving candidate origins
	Only    bool // keep only CDN/WAF IPs
}
