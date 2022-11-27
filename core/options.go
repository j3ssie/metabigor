package core

// Options global options
type Options struct {
	Input     string
	InputFile string

	Inputs     []string
	Output     string
	TmpOutput  string
	ConfigFile string
	LogFile    string
	Proxy      string

	Concurrency   int
	Delay         int
	SaveRaw       bool
	Timeout       int
	Retry         int
	JsonOutput    bool
	PipeTheOutput bool
	Verbose       bool
	Quiet         bool
	Debug         bool
	Scan          ScanOptions
	Net           NetOptions
	Search        SearchOptions
	CVE           CVEOptions
	Cert          CertOptions
	Tld           TldOptions
}

// TldOptions options for tld command
type TldOptions struct {
	Source string
}

// CertOptions options for cert command
type CertOptions struct {
	Clean        bool
	OnlyWildCard bool
}

// ScanOptions options for net command
type ScanOptions struct {
	Ports             string
	Rate              string
	Retry             string
	Timeout           string
	NmapTemplate      string
	NmapOverview      bool
	ZmapOverview      bool
	Detail            bool
	Flat              bool
	All               bool
	IPv4              bool
	IPv6              bool
	Skip80And443      bool
	SkipOverview      bool
	InputFromRustScan bool
	TmpOutput         string
	NmapScripts       string
	GrepString        string
	InputFile         string
}

// NetOptions options for net command
type NetOptions struct {
	Asn        string
	Org        string
	IP         string
	Domain     string
	SearchType string
	Optimize   bool
	ExactMatch bool
}

// SearchOptions options for net command
type SearchOptions struct {
	Source   string
	Query    string
	Optimize bool
	More     bool
}

// CVEOptions options for cve command
type CVEOptions struct {
	Software string
	Version  string
}

// Request all information about request
type Request struct {
	Timeout  int
	Repeat   int
	Scheme   string
	Host     string
	Port     string
	Path     string
	URL      string
	Proxy    string
	Method   string
	Redirect bool
	Headers  []map[string]string
	Body     string
	Beautify string
}

// Response all information about response
type Response struct {
	HasPopUp       bool
	StatusCode     int
	Status         string
	Headers        []map[string]string
	Body           string
	ResponseTime   float64
	Length         int
	Beautify       string
	BeautifyHeader string
}

type RelatedDomain struct {
	Domain    string
	RawData   string
	Technique string
	Source    string
	Output    string
}
