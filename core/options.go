package core

// Options global options
type Options struct {
	Input      string
	Output     string
	TmpOutput  string
	ConfigFile string
	LogFile    string
	Proxy      string

	Concurrency int
	Delay       int
	SaveRaw     bool
	Timeout     int
	JsonOutput  bool
	Verbose     bool
	Debug       bool
	Scan        ScanOptions
	Net         NetOptions
	Search      SearchOptions
	CVE         CVEOptions
	Cert        CertOptions
}

// CertOptions options for net command
type CertOptions struct {
	Clean        bool
	OnlyWildCard bool
}

// ScanOptions options for net command
type ScanOptions struct {
	Ports        string
	Rate         string
	NmapTemplate string
	NmapOverview bool
	ZmapOverview bool
	Detail       bool
	Flat         bool
	All          bool
	IPv4         bool
	IPv6         bool
	SkipOverview bool
	TmpOutput    string
	NmapScripts  string
	GrepString   string
	InputFile    string
}

// NetOptions options for net command
type NetOptions struct {
	Asn      string
	Org      string
	IP       string
	Domain   string
	Optimize bool
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
