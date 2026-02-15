// Package options defines configuration structures for all metabigor commands and global flags.
package options

// Options holds all global and subcommand-specific settings.
type Options struct {
	// Global flags
	Input      string
	InputFile  string
	Output     string
	Concurrency int
	Timeout    int
	Retry      int
	Proxy      string
	Silent     bool
	Debug      bool
	JSONOutput bool
	NoColor    bool

	// net subcommand
	Net NetOptions

	// cert subcommand
	Cert CertOptions

	// related subcommand
	Related RelatedOptions

	// ip subcommand
	IP IPOptions

	// github subcommand
	Github GithubOptions

	// cdn subcommand
	CDN CDNOptions
}

// NetOptions holds configuration for the net command.
type NetOptions struct {
	// Input type overrides
	ASN     bool
	Org     bool
	IP      bool
	Domain  bool
	Dynamic bool
	Detail  bool // Show detailed BGP info (type, description, country)
}

// CertOptions holds configuration for the cert command.
type CertOptions struct {
	Clean    bool
	Wildcard bool
	Simple   bool // Output only domain names (old behavior)
}

// RelatedOptions holds configuration for the related command.
type RelatedOptions struct {
	Source string // crt, whois, ua, gtm, all
}

// IPOptions holds configuration for the ip command.
type IPOptions struct {
	Flat bool // IP:PORT flat output
	CSV  bool
}

// GithubOptions holds configuration for the github command.
type GithubOptions struct{
	Page   int
	Detail bool // show formatted code snippet per hit
}

// CDNOptions holds configuration for the cdn command.
type CDNOptions struct {
	StripCDN bool // only output non-CDN IPs
}
